package rmnp

import "fmt"

type congestionMode uint8

const (
	None congestionMode = iota
	Good
	Bad
)

type congestionHandler struct {
	mode congestionMode
	rtt  int64

	lastChangeTime int64
	requiredTime   int64

	unreliableCount byte

	ResendTimeout    int64
	MaxPacketResends int64
	ReackTimeout     int64
}

func NewCongestionHandler() *congestionHandler {
	handler := new(congestionHandler)
	handler.reset()
	return handler
}

func (handler *congestionHandler) reset() {
	handler.changeMode(None)
	handler.rtt = 0
	handler.requiredTime = DefaultCongestionRequiredTime
	handler.unreliableCount = 0
}

func (handler *congestionHandler) check(sendTime int64) {
	time := currentTime()
	rtt := time - sendTime

	if handler.rtt == 0 {
		handler.rtt = rtt
	} else {
		handler.rtt += int64(float32(rtt-handler.rtt) * RTTSmoothFactor)
	}

	switch handler.mode {
	case None:
		handler.changeMode(Good)
	case Good:
		if rtt > CongestionThreshold {
			if time-handler.lastChangeTime <= BadRTTPunishTimeout {
				handler.requiredTime = Min(MaxCongestionRequiredTime, handler.requiredTime*2)
			}

			handler.changeMode(Bad)
		} else if time-handler.lastChangeTime >= GoodRTTRewardInterval {
			handler.requiredTime = Max(1, handler.requiredTime/2)
			handler.lastChangeTime = time
		}
	case Bad:
		if rtt > CongestionThreshold {
			handler.lastChangeTime = time
		}

		if time-handler.lastChangeTime >= handler.requiredTime {
			handler.changeMode(Good)
		}
	}
}

func (handler *congestionHandler) changeMode(mode congestionMode) {
	switch mode {
	case None:
		fallthrough
	case Good:
		handler.ResendTimeout = ResendTimeout
		handler.MaxPacketResends = MaxPacketResends
		handler.ReackTimeout = ReackTimeout
		fmt.Println("============================> congestion mode: good")
	case Bad:
		handler.ResendTimeout = int64(float32(ResendTimeout) * BadModeMultiplier)
		handler.MaxPacketResends = int64(float32(MaxPacketResends) / BadModeMultiplier)
		handler.ReackTimeout = int64(float32(ReackTimeout) * BadModeMultiplier)
		fmt.Println("============================> congestion mode: bad")
	}

	handler.mode = mode
	handler.lastChangeTime = currentTime()
}

// for unreliable packets only
func (handler *congestionHandler) shouldDropUnreliable() bool {
	switch handler.mode {
	case Good:
		return false
	case Bad:
		handler.unreliableCount++
		return handler.unreliableCount%CongestionPacketReduction == 0
	}

	return false
}

package rmnp

import "fmt"

type congestionMode uint8

const (
	Good congestionMode = iota
	Bad
)

type congestionHandler struct {
	mode congestionMode
	rtt  int64

	lastChangeTime int64
	requiredTime   int64

	multiplier      float32
	unreliableCount byte
}

func NewCongestionHandler() *congestionHandler {
	return &congestionHandler{
		mode:           Good,
		lastChangeTime: currentTime(),
		requiredTime:   DefaultCongestionRequiredTime,
		multiplier:     1.0,
	}
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
	case Good:
		handler.multiplier = 1.0
		fmt.Println("============================> congestion mode: good")
	case Bad:
		handler.multiplier = BadModeMultiplier
		fmt.Println("============================> congestion mode: bad")
	}

	handler.mode = mode
	handler.lastChangeTime = currentTime()
}

func (handler *congestionHandler) div(i int64) int64 {
	return int64(float32(i) / handler.multiplier)
}

func (handler *congestionHandler) mul(i int64) int64 {
	return int64(float32(i) * handler.multiplier)
}

func (handler *congestionHandler) shouldDrop() bool {
	switch handler.mode {
	case Good:
		return false
	case Bad:
		handler.unreliableCount++
		return handler.unreliableCount%CongestionPacketReduction == 0
	}

	return false
}

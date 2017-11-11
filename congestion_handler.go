package rmnp

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

	multiplier float32
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
		handler.rtt += (rtt - handler.rtt) * RTTSmoothFactor
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
	case Bad:
		handler.multiplier = BadModeMultiplier
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

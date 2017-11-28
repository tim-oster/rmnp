// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

type congestionMode uint8

const (
	congestionModeNone congestionMode = iota
	congestionModeGood
	congestionModeBad
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

func newCongestionHandler() *congestionHandler {
	handler := new(congestionHandler)
	handler.reset()
	return handler
}

func (handler *congestionHandler) reset() {
	handler.changeMode(congestionModeNone)
	handler.rtt = 0
	handler.requiredTime = CfgDefaultCongestionRequiredTime
	handler.unreliableCount = 0
}

func (handler *congestionHandler) check(sendTime int64) {
	time := currentTime()
	rtt := time - sendTime

	if handler.rtt == 0 {
		handler.rtt = rtt
	} else {
		handler.rtt += int64(float32(rtt-handler.rtt) * CfgRTTSmoothFactor)
	}

	switch handler.mode {
	case congestionModeNone:
		handler.changeMode(congestionModeGood)
	case congestionModeGood:
		if rtt > CfgCongestionThreshold {
			if time-handler.lastChangeTime <= CfgBadRTTPunishTimeout {
				handler.requiredTime = min(CfgMaxCongestionRequiredTime, handler.requiredTime*2)
			}

			handler.changeMode(congestionModeBad)
		} else if time-handler.lastChangeTime >= CfgGoodRTTRewardInterval {
			handler.requiredTime = max(1, handler.requiredTime/2)
			handler.lastChangeTime = time
		}
	case congestionModeBad:
		if rtt > CfgCongestionThreshold {
			handler.lastChangeTime = time
		}

		if time-handler.lastChangeTime >= handler.requiredTime {
			handler.changeMode(congestionModeGood)
		}
	}
}

func (handler *congestionHandler) changeMode(mode congestionMode) {
	switch mode {
	case congestionModeNone:
		fallthrough
	case congestionModeGood:
		handler.ResendTimeout = CfgResendTimeout
		handler.MaxPacketResends = CfgMaxPacketResends
		handler.ReackTimeout = CfgReackTimeout
	case congestionModeBad:
		handler.ResendTimeout = int64(float32(CfgResendTimeout) * CfgBadModeMultiplier)
		handler.MaxPacketResends = int64(float32(CfgMaxPacketResends) / CfgBadModeMultiplier)
		handler.ReackTimeout = int64(float32(CfgReackTimeout) * CfgBadModeMultiplier)
	}

	handler.mode = mode
	handler.lastChangeTime = currentTime()
}

// for unreliable packets only
func (handler *congestionHandler) shouldDropUnreliable() bool {
	switch handler.mode {
	case congestionModeGood:
		return false
	case congestionModeBad:
		handler.unreliableCount++
		return handler.unreliableCount%CfgCongestionPacketReduction == 0
	}

	return false
}

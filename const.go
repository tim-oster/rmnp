// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

const (
	MTU        = 1024
	ProtocolId = 231

	MaxSkippedPackets = 25

	ParallelListenerCount   = 4
	MaxSendReceiveQueueSize = 100

	// size needs to ensure that every slot in buffers is at least overwritten 2 times, otherwise acks will fail (max_sequence % size > 32 && max_sequence / size >= 2)
	SequenceBufferSize = 200

	UpdateLoopInterval = 10
	AutoPingInterval   = 15
	SendRemoveTimeout  = 1600

	TimeoutThreshold = 4000
	MaxPing          = 150

	RTTSmoothFactor               = 0.1
	CongestionThreshold           = 250
	GoodRTTRewardInterval         = 10 * 1000
	BadRTTPunishTimeout           = 10 * 1000
	MaxCongestionRequiredTime     = 60 * 1000
	DefaultCongestionRequiredTime = 4 * 1000
	CongestionPacketReduction     = 4
)

// super annoying but golang does not allow to cast float to int if the variable is declared as static for some reason
var (
	BadModeMultiplier         = 2.5
	ResendTimeout       int64 = 50
	MaxPacketResends    int64 = 15
	ReackTimeout        int64 = 50
	BadResendTimeout          = int64(float64(ResendTimeout) * BadModeMultiplier)
	BadMaxPacketResends       = int64(float64(MaxPacketResends) / BadModeMultiplier)
	BadReackTimeout           = int64(float64(ReackTimeout) * BadModeMultiplier)
)

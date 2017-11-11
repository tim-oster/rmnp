// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

const (
	MTU        = 1024
	ProtocolId = 231

	MaxSkippedPackets = 25
	MaxPacketResends  = 15

	// size needs to ensure that every slot in buffers is at least overwritten 2 times, otherwise acks will fail (max_sequence % size > 32 && max_sequence / size >= 2)
	SequenceBufferSize = 200

	UpdateLoopInterval = 10
	ReackTimeout       = 50
	ResendTimeout      = 50
	SendRemoveTimeout  = 1600

	TimeoutThreshold = 8000

	RTTSmoothFactor               = 0.1
	BadModeMultiplier             = 2.5
	CongestionThreshold           = 250
	GoodRTTRewardInterval         = 10 * 1000
	BadRTTPunishTimeout           = 10 * 1000
	MaxCongestionRequiredTime     = 60 * 1000
	DefaultCongestionRequiredTime = 4 * 1000
)

// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "time"

var (
	MTU             = 1024
	ProtocolId byte = 231

	ParallelListenerCount   = 4
	MaxSendReceiveQueueSize = 100
)

var (
	// size needs to ensure that every slot in buffers is at least overwritten 2 times, otherwise acks will fail (max_sequence % size > 32 && max_sequence / size >= 2)
	SequenceBufferSize sequenceNumber = 200
	MaxSkippedPackets  sequenceNumber = 25

	UpdateLoopInterval time.Duration = 10
	SendRemoveTimeout  int64         = 1600
	AutoPingInterval   uint8         = 15
)

var (
	TimeoutThreshold time.Duration = 4000
	MaxPing          int16         = 150
)

var (
	RTTSmoothFactor               float32 = 0.1
	CongestionThreshold           int64   = 250
	GoodRTTRewardInterval         int64   = 10 * 1000
	BadRTTPunishTimeout           int64   = 10 * 1000
	MaxCongestionRequiredTime     int64   = 60 * 1000
	DefaultCongestionRequiredTime int64   = 4 * 1000
	CongestionPacketReduction     uint8   = 4
)

var (
	BadModeMultiplier float32 = 2.5
	ResendTimeout     int64   = 50
	MaxPacketResends  int64   = 15
	ReackTimeout      int64   = 50
)

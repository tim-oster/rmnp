// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "time"

var (
	CfgMTU             = 1024
	CfgProtocolId byte = 231

	CfgParallelListenerCount        = 4
	CfgMaxSendReceiveQueueSize      = 100
	CfgMaxPacketChainLength    byte = 255
)

var (
	// size needs to ensure that every slot in buffers is at least overwritten 2 times, otherwise acks will fail (max_sequence % size > 32 && max_sequence / size >= 2)
	CfgSequenceBufferSize sequenceNumber = 200
	CfgMaxSkippedPackets  sequenceNumber = 25

	CfgUpdateLoopInterval time.Duration = 10
	CfgSendRemoveTimeout  int64         = 1600
	CfgChainSkipTimeout   int64         = 3000
	CfgAutoPingInterval   uint8         = 15
)

var (
	CfgTimeoutThreshold time.Duration = 4000
	CfgMaxPing          int16         = 150
)

var (
	CfgRTTSmoothFactor               float32 = 0.1
	CfgCongestionThreshold           int64   = 250
	CfgGoodRTTRewardInterval         int64   = 10 * 1000
	CfgBadRTTPunishTimeout           int64   = 10 * 1000
	CfgMaxCongestionRequiredTime     int64   = 60 * 1000
	CfgDefaultCongestionRequiredTime int64   = 4 * 1000
	CfgCongestionPacketReduction     uint8   = 4
)

var (
	CfgBadModeMultiplier float32 = 2.5
	CfgResendTimeout     int64   = 50
	CfgMaxPacketResends  int64   = 15
	CfgReackTimeout      int64   = 50
)

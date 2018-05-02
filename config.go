// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "time"

var (
	// CfgMTU is the maximum byte size of a packet (header included).
	CfgMTU = 1024

	// CfgProtocolID is the identification number send with every rmnp packet to filter out unwanted traffic.
	CfgProtocolID byte = 231

	// CfgParallelListenerCount is the amount of goroutines that will be spawned to listen on incoming requests.
	CfgParallelListenerCount = 4

	// CfgMaxSendReceiveQueueSize is the max size of packets that can be queued up before they are processed.
	CfgMaxSendReceiveQueueSize = 100

	// CfgMaxPacketChainLength is the max length of packets that are chained when waiting for missing sequence.
	CfgMaxPacketChainLength byte = 255
)

var (
	// CfgSequenceBufferSize is the size of the buffer that store the last received packets in order to ack them.
	// Size should be big enough that packets are at least overridden twice (max_sequence % size > 32 && max_sequence / size >= 2).
	// (max_sequence = highest possible sequence number = max value of sequenceNumber)
	CfgSequenceBufferSize sequenceNumber = 200

	// CfgMaxSkippedPackets is the max amount of that are allowed to be skipped during packet loss (should be less than 32).
	CfgMaxSkippedPackets sequenceNumber = 25

	// CfgUpdateLoopTimeout is the max wait duration for the connection update loop (should be less than other timeout variables).
	CfgUpdateLoopTimeout time.Duration = 10

	// CfgSendRemoveTimeout is the time after which packets that have not being acked yet stop to be resend.
	CfgSendRemoveTimeout int64 = 1600

	// CfgChainSkipTimeout is the time after which a missing packet for a complete sequence is ignored and skipped.
	CfgChainSkipTimeout int64 = 3000

	// CfgAutoPingInterval defines the interval for sending a ping packet.
	CfgAutoPingInterval uint8 = 15
)

var (
	// CfgTimeoutThreshold is the time after which a connection times out if no packets have being send for the specified amount of time.
	CfgTimeoutThreshold time.Duration = 4000

	// CfgMaxPing is the max ping before a connection times out.
	CfgMaxPing int16 = 150
)

var (
	// CfgRTTSmoothFactor is the factor used to slowly adjust the RTT.
	CfgRTTSmoothFactor float32 = 0.1

	// CfgCongestionThreshold is the max RTT before the connection enters bad mode.
	CfgCongestionThreshold int64 = 250

	// CfgGoodRTTRewardInterval the time after how many seconds a good connection is rewarded.
	CfgGoodRTTRewardInterval int64 = 10 * 1000

	// CfgBadRTTPunishTimeout the time after how many seconds a bad connection is punished.
	CfgBadRTTPunishTimeout int64 = 10 * 1000

	// CfgMaxCongestionRequiredTime the max time it should take to switch between modes.
	CfgMaxCongestionRequiredTime int64 = 60 * 1000

	// CfgDefaultCongestionRequiredTime the initial time it should take to switch back from bad to good mode.
	CfgDefaultCongestionRequiredTime int64 = 4 * 1000

	// CfgCongestionPacketReduction is the amount of unreliable packets that get dropped in bad mode.
	CfgCongestionPacketReduction uint8 = 4
)

var (
	// CfgBadModeMultiplier is the multiplier for variables in bad mode.
	CfgBadModeMultiplier float32 = 2.5

	// CfgResendTimeout is the default timeout for packets before resend.
	CfgResendTimeout int64 = 50

	// CfgMaxPacketResends is the default max amount of packets to resend during one update.
	CfgMaxPacketResends int64 = 15

	// CfgReackTimeout is the default timeout before a manual ack packet gets send.
	CfgReackTimeout int64 = 50
)

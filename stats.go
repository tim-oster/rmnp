// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

var (
	StatSendBytes      uint64 = 0
	StatReceivedBytes  uint64 = 0
	StatProcessedBytes uint64 = 0

	StatRunningRoutines uint64 = 0
	StatGoRoutinePanics uint64 = 0

	StatConnects    uint64 = 0
	StatDisconnects uint64 = 0
	StatTimeouts    uint64 = 0
)

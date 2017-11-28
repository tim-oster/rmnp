// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

var (
	// StatSendBytes (atomic) counts the total amount of bytes send.
	StatSendBytes uint64 = 0

	// StatReceivedBytes (atomic) counts the total amount of bytes received.
	// Not the same as StatProcessedBytes because received packets may be discarded.
	StatReceivedBytes uint64 = 0

	// StatProcessedBytes (atomic) counts the total size of all processed packets.
	StatProcessedBytes uint64 = 0
)

var (
	// StatRunningGoRoutines (atomic) counts all currently active goroutines spawned by rmnp
	StatRunningGoRoutines uint64 = 0

	// StatGoRoutinePanics (atomic) counts the amount of caught goroutine panics
	StatGoRoutinePanics uint64 = 0
)

var (
	// StatConnects (atomic) counts all successful connects
	StatConnects uint64 = 0

	// StatDeniedConnects (atomic) counts all denied connection attempts
	StatDeniedConnects uint64 = 0

	// StatDisconnects (atomic) counts all disconnects
	StatDisconnects uint64 = 0

	// StatTimeouts (atomic) counts all timeouts
	StatTimeouts uint64 = 0
)

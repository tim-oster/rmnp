// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "sync"

type sequenceBuffer struct {
	size      sequenceNumber
	sequences []sequenceNumber
	states    []bool
	mutex     sync.Mutex
}

func newSequenceBuffer(size sequenceNumber) *sequenceBuffer {
	buffer := new(sequenceBuffer)
	buffer.size = size
	buffer.sequences = make([]sequenceNumber, size)
	buffer.states = make([]bool, size)
	return buffer
}

func (buffer *sequenceBuffer) reset() {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	for i := sequenceNumber(0); i < buffer.size; i++ {
		buffer.sequences[i] = 0
		buffer.states[i] = false
	}
}

func (buffer *sequenceBuffer) get(sequence sequenceNumber) bool {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	if buffer.sequences[sequence%buffer.size] != sequence {
		return false
	}

	return buffer.states[sequence%buffer.size]
}

func (buffer *sequenceBuffer) set(sequence sequenceNumber, value bool) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	buffer.sequences[sequence%buffer.size] = sequence
	buffer.states[sequence%buffer.size] = value
}

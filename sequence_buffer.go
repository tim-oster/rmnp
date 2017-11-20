// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "sync"

type SequenceBuffer struct {
	size      sequenceNumber
	sequences []sequenceNumber
	states    []bool
	mutex     sync.Mutex
}

func NewSequenceBuffer(size sequenceNumber) *SequenceBuffer {
	buffer := new(SequenceBuffer)
	buffer.size = size
	buffer.sequences = make([]sequenceNumber, size)
	buffer.states = make([]bool, size)
	return buffer
}

func (buffer *SequenceBuffer) reset() {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	for i := sequenceNumber(0); i < buffer.size; i++ {
		buffer.sequences[i] = 0
		buffer.states[i] = false
	}
}

func (buffer *SequenceBuffer) Get(sequence sequenceNumber) bool {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	if sequence < 0 {
		sequence += buffer.size
	}

	if buffer.sequences[sequence%buffer.size] != sequence {
		return false
	}

	return buffer.states[sequence%buffer.size]
}

func (buffer *SequenceBuffer) Set(sequence sequenceNumber, value bool) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	if sequence < 0 {
		sequence += buffer.size
	}

	buffer.sequences[sequence%buffer.size] = sequence
	buffer.states[sequence%buffer.size] = value
}

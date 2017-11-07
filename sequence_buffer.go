// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

type SequenceBuffer struct {
	size      byte
	sequences []byte
	states    []bool
}

func NewSequenceBuffer(size byte) *SequenceBuffer {
	buffer := new(SequenceBuffer)
	buffer.size = size
	buffer.sequences = make([]byte, size)
	buffer.states = make([]bool, size)
	return buffer
}

func (buffer *SequenceBuffer) Get(sequence byte) bool {
	if sequence < 0 {
		sequence += buffer.size
	}

	if buffer.sequences[sequence%buffer.size] != sequence {
		return false
	}

	return buffer.states[sequence%buffer.size]
}

func (buffer *SequenceBuffer) Set(sequence byte, value bool) {
	if sequence < 0 {
		sequence += buffer.size
	}

	buffer.sequences[sequence%buffer.size] = sequence
	buffer.states[sequence%buffer.size] = value
}

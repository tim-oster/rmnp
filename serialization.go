// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"bytes"
	"encoding/binary"
)

// Serializer reads and writes binary data
type Serializer struct {
	buffer *bytes.Buffer
}

// NewSerializer is a new serializer with an empty buffer
func NewSerializer() *Serializer {
	return &Serializer{new(bytes.Buffer)}
}

// NewSerializerFor creates a serializer based on the contents of the buffer
func NewSerializerFor(buffer []byte) *Serializer {
	return &Serializer{bytes.NewBuffer(buffer)}
}

// Read reads binary data from the serializer
func (s *Serializer) Read(data interface{}) error {
	return binary.Read(s.buffer, binary.LittleEndian, data)
}

// ReadPanic reads from the serializer and panics if no data is read
func (s *Serializer) ReadPanic(data interface{}) {
	if s.Read(data) != nil {
		panic(0)
	}
}

// Write writes binary data into the serializer
func (s *Serializer) Write(data interface{}) error {
	return binary.Write(s.buffer, binary.LittleEndian, data)
}

// Bytes is the byte data stored in the serializer
func (s *Serializer) Bytes() []byte {
	return s.buffer.Bytes()
}

// RemainingSize is the buffer size minus the bytes that have already been read
func (s *Serializer) RemainingSize() int {
	return s.buffer.Len()
}

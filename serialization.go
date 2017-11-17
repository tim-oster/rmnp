// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"bytes"
	"encoding/binary"
)

type Serializer struct {
	buffer *bytes.Buffer
}

func NewSerializer() *Serializer {
	return &Serializer{new(bytes.Buffer)}
}

func NewSerializerFor(buffer []byte) *Serializer {
	return &Serializer{bytes.NewBuffer(buffer)}
}

func (s *Serializer) Read(data interface{}) error {
	return binary.Read(s.buffer, binary.BigEndian, data)
}

func (s *Serializer) Write(data interface{}) error {
	return binary.Write(s.buffer, binary.BigEndian, data)
}

func (s *Serializer) Bytes() []byte {
	return s.buffer.Bytes()
}

func (s *Serializer) RemainingSize() int {
	return s.buffer.Len()
}

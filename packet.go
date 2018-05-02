// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"encoding/binary"
	"hash/crc32"
)

type sequenceNumber uint16
type orderNumber uint8
type descriptor byte

const (
	descReliable descriptor = 1 << iota
	descAck
	descOrdered

	descConnect
	descDisconnect
)

type packet struct {
	protocolID byte
	crc32      uint32
	descriptor descriptor

	// only contained in Reliable or Unreliable Ordered packets
	sequence sequenceNumber

	// only for Reliable Ordered packets
	order orderNumber

	// only contained in Ack packets
	ack     sequenceNumber
	ackBits uint32

	// body
	data []byte
}

func (p *packet) serialize() []byte {
	s := NewSerializer()

	s.Write(p.protocolID)
	s.Write(p.crc32)
	s.Write(p.descriptor)

	if p.flag(descReliable) || p.flag(descOrdered) {
		s.Write(p.sequence)
	}

	if p.flag(descReliable) && p.flag(descOrdered) {
		s.Write(p.order)
	}

	if p.flag(descAck) {
		s.Write(p.ack)
		s.Write(p.ackBits)
	}

	if p.data != nil && len(p.data) > 0 {
		s.Write(p.data)
	}

	return s.Bytes()
}

func (p *packet) deserialize(packet []byte) bool {
	s := NewSerializerFor(packet)

	// head is valid (validated before data processing)
	s.Read(&p.protocolID)
	s.Read(&p.crc32)
	s.Read(&p.descriptor)

	if p.flag(descReliable) || p.flag(descOrdered) {
		if s.Read(&p.sequence) != nil {
			return false
		}
	}

	if p.flag(descReliable) && p.flag(descOrdered) {
		if s.Read(&p.order) != nil {
			return false
		}
	}

	if p.flag(descAck) {
		if s.Read(&p.ack) != nil {
			return false
		}

		if s.Read(&p.ackBits) != nil {
			return false
		}
	}

	if size := s.RemainingSize(); size > 0 {
		p.data = make([]byte, size)
		s.Read(&p.data)
	}

	return true
}

func (p *packet) calculateHash() {
	p.crc32 = 0
	buffer := p.serialize()
	p.crc32 = crc32.ChecksumIEEE(buffer)
}

func (p *packet) flag(flag descriptor) bool {
	return p.descriptor&flag != 0
}

func validateHeader(packet []byte) bool {
	// 1b protocolId + 4b crc32 + 1b descriptor
	if len(packet) < 6 {
		return false
	}

	if packet[0] != CfgProtocolID {
		return false
	}

	if len(packet) < headerSize(packet) {
		return false
	}

	hash1 := binary.LittleEndian.Uint32(packet[1:5])
	hash2 := crc32.ChecksumIEEE(append([]byte{packet[0], 0, 0, 0, 0}, packet[5:]...))
	return hash1 == hash2
}

func headerSize(packet []byte) int {
	desc := descriptor(packet[5])
	size := 0

	// protocolId (1) + crc (4) + descriptor (1)
	size += 6

	if desc&descReliable != 0 || desc&descOrdered != 0 {
		// sequence (2)
		size += 2
	}

	if desc&descReliable != 0 && desc&descOrdered != 0 {
		// order (1)
		size++
	}

	if desc&descAck != 0 {
		// ack (2) + ackBits (4)
		size += 6
	}

	return size
}

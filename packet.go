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
	// Send Flags
	Reliable descriptor = 1 << iota
	Ack
	Ordered

	// Basic Packet Types (only single use possible)
	Connect
	Disconnect
)

type Packet struct {
	protocolId byte
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

func (p *Packet) Serialize() []byte {
	s := NewSerializer()

	s.Write(p.protocolId)
	s.Write(p.crc32)
	s.Write(p.descriptor)

	if p.Flag(Reliable) || p.Flag(Ordered) {
		s.Write(p.sequence)
	}

	if p.Flag(Reliable) && p.Flag(Ordered) {
		s.Write(p.order)
	}

	if p.Flag(Ack) {
		s.Write(p.ack)
		s.Write(p.ackBits)
	}

	if p.data != nil && len(p.data) > 0 {
		s.Write(p.data)
	}

	return s.Bytes()
}

func (p *Packet) Deserialize(packet []byte) bool {
	s := NewSerializerFor(packet)

	// head is valid (validated before data processing)
	s.Read(&p.protocolId)
	s.Read(&p.crc32)
	s.Read(&p.descriptor)

	if p.Flag(Reliable) || p.Flag(Ordered) {
		if s.Read(&p.sequence) != nil {
			return false
		}
	}

	if p.Flag(Reliable) && p.Flag(Ordered) {
		if s.Read(&p.order) != nil {
			return false
		}
	}

	if p.Flag(Ack) {
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

func (p *Packet) CalculateHash() {
	p.crc32 = 0
	buffer := p.Serialize()
	p.crc32 = crc32.ChecksumIEEE(buffer)
}

func (p *Packet) Flag(flag descriptor) bool {
	return p.descriptor&flag != 0
}

func validateHeader(packet []byte) bool {
	// 1b protocolId + 4b crc32 + 1b descriptor
	if len(packet) < 6 {
		return false
	}

	if packet[0] != ProtocolId {
		return false
	}

	if len(packet) < headerSize(packet) {
		return false
	}

	hash1 := binary.BigEndian.Uint32(packet[1:5])
	hash2 := crc32.ChecksumIEEE(append([]byte{packet[0], 0, 0, 0, 0}, packet[5:]...))
	return hash1 == hash2
}

func headerSize(packet []byte) int {
	desc := descriptor(packet[5])
	size := 0

	// protocolId (1) + crc (4) + descriptor (1)
	size += 6

	if desc&Reliable != 0 || desc&Ordered != 0 {
		// sequence (2)
		size += 2
	}

	if desc&Reliable != 0 && desc&Ordered != 0 {
		// order (1)
		size += 2
	}

	if desc&Ack != 0 {
		// ack (2) + ackBits (4)
		size += 6
	}

	return size
}

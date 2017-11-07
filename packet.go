// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"encoding/binary"
	"hash/crc32"
)

type descriptor byte

const (
	Reliable descriptor = 1 << iota
	Ack
)

type Packet struct {
	protocolId byte
	crc32      uint32
	descriptor descriptor

	// only contained in Reliable packets
	sequence byte

	// only contained in Ack packets
	ack     byte
	ackBits uint32
}

func (p *Packet) Serialize() []byte {
	// TODO pool?
	s := NewSerializer()

	s.Write(p.protocolId)
	s.Write(p.crc32)
	s.Write(p.descriptor)

	if p.descriptor&Reliable != 0 {
		s.Write(p.sequence)
	}

	if p.descriptor&Ack != 0 {
		s.Write(p.ack)
		s.Write(p.ackBits)
	}

	return s.Bytes()
}

func (p *Packet) Deserialize(packet []byte) bool {
	s := NewSerializerFor(packet)

	// header is valid (validated before packet processing)
	s.Read(&p.protocolId)
	s.Read(&p.crc32)
	s.Read(&p.descriptor)

	if p.descriptor&Reliable != 0 {
		if s.Read(&p.sequence) != nil {
			return false
		}
	}

	if p.descriptor&Ack != 0 {
		if s.Read(&p.ack) != nil {
			return false
		}
		if s.Read(&p.ackBits) != nil {
			return false
		}
	}

	return true
}

func (p *Packet) CalculateHash() {
	p.crc32 = 0
	buffer := p.Serialize()
	p.crc32 = crc32.ChecksumIEEE(buffer)
}

func validatePacketBytes(packet []byte) bool {
	// 1b protocolId + 4b crc32 + 1b descriptor
	if len(packet) < 6 {
		return false
	}

	if packet[0] != ProtocolId {
		return false
	}

	hash1 := binary.BigEndian.Uint32(packet[1:5])
	hash2 := crc32.ChecksumIEEE(append([]byte{packet[0]}, packet[5:]...))
	return hash1 == hash2
}

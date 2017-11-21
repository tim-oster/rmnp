// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "testing"

var testPacketDescriptors = map[descriptor]int{
	0:                                    6,
	descReliable:                         8,
	descOrdered:                          8,
	descReliable | descOrdered:           9,
	descAck:                              12,
	descReliable | descOrdered | descAck: 15,
}

var testPacketDescriptorPermutations = []descriptor{
	descReliable,
	descAck,
	descOrdered,
	descReliable | descOrdered,
	descReliable | descOrdered | descAck,
	descReliable | descConnect,
	descReliable | descDisconnect,
}

func newTestPacket() *packet {
	return &packet{
		protocolId: CfgProtocolId,
		crc32:      244,
		descriptor: descReliable | descAck | descOrdered,
		sequence:   10,
		order:      5,
		ack:        18,
		ackBits:    24,
		data:       nil,
	}
}

func TestPacketSerialization(t *testing.T) {
	s := newTestPacket()
	d := new(packet)

	s.data = []byte{0, 1, 2, 3, 4, 5, 6, 7}
	d.deserialize(s.serialize())

	if d.protocolId != s.protocolId {
		t.Error("packet.protocolId not correctly serialized")
	}

	if d.crc32 != s.crc32 {
		t.Error("packet.crc32 not correctly serialized")
	}

	if d.descriptor != s.descriptor {
		t.Error("packet.descriptor not correctly serialized")
	}

	if d.sequence != s.sequence {
		t.Error("packet.sequence not correctly serialized")
	}

	if d.order != s.order {
		t.Error("packet.order not correctly serialized")
	}

	if d.ack != s.ack {
		t.Error("packet.ack not correctly serialized")
	}

	if d.ackBits != s.ackBits {
		t.Error("packet.ackBits not correctly serialized")
	}

	if (d.data != nil && s.data == nil) || (d.data == nil && s.data != nil) || len(d.data) != len(s.data) {
		t.Error("packet.data not correctly serialized")
	} else {
		for i, e := range d.data {
			if s.data[i] != e {
				t.Error("packet.data not correctly serialized")
				break
			}
		}
	}
}

func TestPacketHash(t *testing.T) {
	p1, p2 := newTestPacket(), newTestPacket()

	p1.calculateHash()
	p2.calculateHash()

	if p1.crc32 != p2.crc32 {
		t.Error("Packet hashes or not equal")
	}
}

func TestPacketFlag(t *testing.T) {
	p := newTestPacket()

	for i, desc := range testPacketDescriptorPermutations {
		p.descriptor = desc

		if !p.flag(desc) {
			t.Errorf("Expected flag (index: %v) to be present", i)
		}
	}
}

func TestPacketValidateHeader(t *testing.T) {
	p := newTestPacket()
	p.calculateHash()

	d := p.serialize()

	if !validateHeader(d) {
		t.Error("Valid packet cannot be validated")
	}

	if validateHeader(d[0:5]) {
		t.Error("Wrong min length for fixed header")
	}

	d[len(d)/2] += 1

	if validateHeader(d) {
		t.Error("Hash check not working")
	}
}

func TestPacketHeaderSize(t *testing.T) {
	p := newTestPacket()

	for desc, size := range testPacketDescriptors {
		p.descriptor = desc
		data := p.serialize()

		if s := headerSize(data); s != size {
			t.Errorf("Expected header size of %v but got %v", size, s)
		}

		if l := len(data); l != size {
			t.Errorf("Expected packet size of %v but got %v", size, l)
		}
	}
}

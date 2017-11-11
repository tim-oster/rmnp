// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"sync"
)

type packetChain struct {
	next    orderNumber
	packets []*Packet
	mutex   sync.Mutex
}

func NewPacketChain() *packetChain {
	c := new(packetChain)
	c.packets = make([]*Packet, 0, 255)
	return c
}

func (chain *packetChain) Chain(packet *Packet) {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()

	// TODO cleanup if len(packets) == max

	i := 0
	for _, p := range chain.packets {
		if greaterThanOrder(packet.order, p.order) {
			i++
		} else {
			break
		}
	}

	chain.packets = append(chain.packets[:i], append([]*Packet{packet}, chain.packets[i:]...)...)
}

// TODO optimize
func (chain *packetChain) PopConsecutive() []*Packet {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()

	// TODO pool?
	out := make([]*Packet, 0)

	for _, p := range chain.packets {
		if p.order == chain.next {
			chain.next++
			out = append(out, p)
		}
	}

	chain.packets = chain.packets[len(out):]

	return out
}

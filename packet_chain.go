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

func (c *packetChain) Chain(packet *Packet) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// TODO cleanup if len(packets) == max

	i := 0
	for _, p := range c.packets {
		if greaterThanOrder(packet.order, p.order) {
			i++
		} else {
			break
		}
	}

	c.packets = append(c.packets[:i], append([]*Packet{packet}, c.packets[i:]...)...)
}

// TODO optimize
func (c *packetChain) PopConsecutive() []*Packet {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// TODO pool?
	out := make([]*Packet, 0)

	for _, p := range c.packets {
		if p.order == c.next {
			c.next++
			out = append(out, p)
		}
	}

	c.packets = c.packets[len(out):]

	return out
}

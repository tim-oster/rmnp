// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"sync"
)

const maxPacketChainLength = 255

type packetChain struct {
	next   orderNumber
	start  *chainLink
	length int
	mutex  sync.Mutex
}

type chainLink struct {
	next   *chainLink
	packet *Packet
}

func NewPacketChain() *packetChain {
	return new(packetChain)
}

func (chain *packetChain) reset() {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()

	chain.next = 0
	chain.start = nil
	chain.length = 0
}

func (chain *packetChain) Chain(packet *Packet) {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()

	if chain.start == nil {
		chain.start = &chainLink{packet: packet}
	} else {
		var link *chainLink = nil

		for l := chain.start; l != nil; l = l.next {
			if greaterThanOrder(packet.order, l.packet.order) {
				link = l
			} else {
				break
			}
		}

		if link == nil {
			chain.start = &chainLink{next: chain.start, packet: packet}
		} else {
			link.next = &chainLink{next: link.next, packet: packet}
		}
	}

	chain.length++

	if chain.length > maxPacketChainLength {
		chain.start = chain.start.next
	}
}

func (chain *packetChain) PopConsecutive() *chainLink {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()

	var last *chainLink = nil

	for l := chain.start; l != nil; l = l.next {
		if l.packet.order == chain.next {
			chain.next++
			last = l
		} else {
			break
		}
	}

	if last != nil {
		start := chain.start
		chain.start = last.next
		last.next = nil
		return start
	}

	return nil
}

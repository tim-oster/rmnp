// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"sync"
	"fmt"
	"sort"
	"time"
)

type packetKeys []sequenceNumber

func (a packetKeys) Len() int           { return len(a) }
func (a packetKeys) Less(i, j int) bool { return a[i] < a[j] }
func (a packetKeys) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type sendPacket struct {
	packet   *Packet
	sendTime int64
}

type Connection struct {
	protocol *protocolImpl
	conn     *net.UDPConn
	addr     *net.UDPAddr

	localSequence  sequenceNumber
	remoteSequence sequenceNumber
	ackBits        uint32

	lastSendTime   int64
	lastResendTime int64
	sendMapMutex   sync.Mutex
	sendMap        map[sequenceNumber]*sendPacket // TODO use other data structure?
	recvBuffer     *SequenceBuffer
}

func (c *Connection) update() {
	for {
		currentTime := currentTime()

		if currentTime-c.lastResendTime > ResendTimeout {
			c.lastResendTime = currentTime

			c.sendMapMutex.Lock()

			keys := make([]sequenceNumber, 0)
			for key := range c.sendMap {
				keys = append(keys, key)
			}
			sort.Sort(packetKeys(keys))

			for _, key := range keys {
				packet := c.sendMap[key]

				if currentTime-packet.sendTime > SendRemoveTimeout {
					delete(c.sendMap, key)
				} else {
					c.sendPacket(packet.packet, true)
				}
			}

			c.sendMapMutex.Unlock()
		}

		if currentTime-c.lastSendTime > ReackTimeout {
			c.sendAckPacket()
		}

		time.Sleep(UpdateLoopTimeout * time.Millisecond)
	}
}

func (c *Connection) handlePacket(packet []byte) {
	// TODO pool?
	p := &Packet{}

	if !p.Deserialize(packet) {
		fmt.Println("error during packet deserialization")
		return
	}

	if p.descriptor&Reliable != 0 {
		c.handleReliablePacket(p)
	}

	if p.descriptor&Ack != 0 {
		c.handleAckPacket(p)
	}

	// TODO process
}

func (c *Connection) handleReliablePacket(packet *Packet) {
	fmt.Println("recveived sequences #", packet.sequence)

	if c.recvBuffer.Get(packet.sequence) {
		fmt.Println(":: was duplicate")
		return
	}

	// update receive states
	c.recvBuffer.Set(packet.sequence, true)

	// update remote sequences number
	if greaterThan(packet.sequence, c.remoteSequence) && difference(packet.sequence, c.remoteSequence) <= MaxSkippedPackets {
		c.remoteSequence = packet.sequence
	}

	// update ack bit mask for last 32 packets
	c.ackBits = 0
	for i := sequenceNumber(1); i <= 32; i++ {
		if c.recvBuffer.Get(c.remoteSequence - i) {
			c.ackBits |= 1 << (i - 1)
		}
	}

	c.sendAckPacket()
}

func (c *Connection) handleAckPacket(packet *Packet) {
	for i := sequenceNumber(0); i <= 32; i++ {
		if i == 0 || packet.ackBits&(1<<(i-1)) != 0 {
			key := packet.ack - i

			c.sendMapMutex.Lock()
			if _, ok := c.sendMap[key]; ok {
				delete(c.sendMap, key)
				fmt.Println("#", key, "acked")
			}
			c.sendMapMutex.Unlock()
		}
	}
}

func (c *Connection) sendPacket(packet *Packet, resend bool) {
	packet.protocolId = ProtocolId

	if packet.descriptor&Reliable != 0 && !resend {
		packet.sequence = c.localSequence
		c.localSequence++

		c.sendMapMutex.Lock()
		c.sendMap[packet.sequence] = &sendPacket{
			packet:   packet,
			sendTime: currentTime(),
		}
		c.sendMapMutex.Unlock()
	}

	if packet.descriptor&Ack != 0 {
		packet.ack = c.remoteSequence
		packet.ackBits = c.ackBits
	}

	if packet.sequence > 0 {
		fmt.Print("send sequences #", packet.sequence)
		if resend {
			fmt.Println(" resend")
		} else {
			fmt.Println()
		}
	}

	packet.CalculateHash()
	buffer := packet.Serialize()
	c.protocol.writeFunc(c, buffer)

	c.lastSendTime = currentTime()
}

func (c *Connection) sendAckPacket() {
	c.sendPacket(&Packet{descriptor: Ack}, false)
}

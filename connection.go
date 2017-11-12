// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"fmt"
	"sort"
	"time"
	"context"
	"sync"
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

	// for go routines
	ctx          context.Context
	stopRoutines context.CancelFunc

	// for Reliable packets
	localSequence   sequenceNumber
	remoteSequence  sequenceNumber
	ackBits         uint32
	orderedChain    *packetChain
	orderedSequence orderNumber

	// for Unreliable Ordered packets
	localUnreliableSequence  sequenceNumber
	remoteUnreliableSequence sequenceNumber

	lastSendTime      int64
	lastResendTime    int64
	lastReceivedTime  int64
	sendMapMutex      sync.Mutex
	sendMap           map[sequenceNumber]*sendPacket // TODO use other data structure?
	receiveBuffer     *SequenceBuffer
	congestionHandler *congestionHandler
}

func (c *Connection) destroy() {
	c.protocol = nil
	c.conn = nil
	c.addr = nil

	c.orderedChain = nil
	c.sendMap = nil
	c.receiveBuffer = nil
	c.congestionHandler = nil
}

func (c *Connection) startRoutines() {
	c.ctx, c.stopRoutines = context.WithCancel(context.Background())
	go c.update(c.ctx)
	go c.keepAlive(c.ctx)
}

func (c *Connection) update(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(UpdateLoopInterval * time.Millisecond):
		}

		currentTime := currentTime()

		if currentTime-c.lastResendTime > c.congestionHandler.mul(ResendTimeout) {
			c.lastResendTime = currentTime

			c.sendMapMutex.Lock()

			keys := make([]sequenceNumber, 0)
			for key := range c.sendMap {
				keys = append(keys, key)
			}
			sort.Sort(packetKeys(keys))

		resendLoop:
			for i, key := range keys {
				if int64(i) >= c.congestionHandler.div(MaxPacketResends) {
					break resendLoop
				}

				packet := c.sendMap[key]

				if currentTime-packet.sendTime > SendRemoveTimeout {
					delete(c.sendMap, key)
				} else {
					c.sendPacket(packet.packet, true)
				}
			}

			c.sendMapMutex.Unlock()
		}

		if currentTime-c.lastSendTime > c.congestionHandler.mul(ReackTimeout) {
			c.sendAckPacket()
		}
	}
}

func (c *Connection) keepAlive(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(TimeoutThreshold * time.Millisecond / 2):
		}

		currentTime := currentTime()

		if currentTime-c.lastReceivedTime > TimeoutThreshold {
			c.protocol.timeoutClient(c)
		}
	}
}

func (c *Connection) handlePacket(packet []byte) {
	c.lastReceivedTime = currentTime()

	// TODO pool?
	p := &Packet{}

	if !p.Deserialize(packet) {
		fmt.Println("error during packet deserialization")
		return
	}

	if p.Flag(Reliable) && !c.handleReliablePacket(p) {
		return
	}

	if p.Flag(Ack) && !c.handleAckPacket(p) {
		return
	}

	if p.Flag(Ordered) && !c.handleOrderedPacket(p) {
		return
	}

	// TODO process
}

func (c *Connection) handleReliablePacket(packet *Packet) bool {
	fmt.Println("recveived sequences #", packet.sequence)

	if c.receiveBuffer.Get(packet.sequence) {
		fmt.Println(":: was duplicate")
		return false
	}

	// update receive states
	c.receiveBuffer.Set(packet.sequence, true)

	// update remote sequences number
	if greaterThanSequence(packet.sequence, c.remoteSequence) && differenceSequence(packet.sequence, c.remoteSequence) <= MaxSkippedPackets {
		c.remoteSequence = packet.sequence
	}

	// update ack bit mask for last 32 packets
	c.ackBits = 0
	for i := sequenceNumber(1); i <= 32; i++ {
		if c.receiveBuffer.Get(c.remoteSequence - i) {
			c.ackBits |= 1 << (i - 1)
		}
	}

	c.sendAckPacket()

	return true
}

func (c *Connection) handleOrderedPacket(packet *Packet) bool {
	if packet.Flag(Reliable) {
		c.orderedChain.Chain(packet)

		// tmp
		c.orderedChain.PopConsecutive()

		/*for _, p := range c.orderedChain.PopConsecutive() {
			// TODO process
		}*/
	} else {
		if greaterThanSequence(packet.sequence, c.remoteUnreliableSequence) {
			c.remoteUnreliableSequence = packet.sequence
			return true
		}
	}

	return false
}

func (c *Connection) handleAckPacket(packet *Packet) bool {
	for i := sequenceNumber(0); i <= 32; i++ {
		if i == 0 || packet.ackBits&(1<<(i-1)) != 0 {
			key := packet.ack - i

			c.sendMapMutex.Lock()

			if p, ok := c.sendMap[key]; ok {
				c.congestionHandler.check(p.sendTime)
				delete(c.sendMap, key)
				fmt.Println("#", key, "acked")
			}

			c.sendMapMutex.Unlock()
		}
	}

	return true
}

func (c *Connection) sendPacket(packet *Packet, resend bool) {
	if !packet.Flag(Reliable) && c.congestionHandler.shouldDrop() {
		return
	}

	packet.protocolId = ProtocolId

	if !resend {
		if packet.Flag(Reliable) {
			packet.sequence = c.localSequence
			c.localSequence++

			if packet.Flag(Ordered) {
				packet.order = c.orderedSequence
				c.orderedSequence++
			}

			c.sendMapMutex.Lock()
			c.sendMap[packet.sequence] = &sendPacket{
				packet:   packet,
				sendTime: currentTime(),
			}
			c.sendMapMutex.Unlock()
		} else if packet.Flag(Ordered) {
			packet.sequence = c.localUnreliableSequence
			c.localUnreliableSequence++
		}
	}

	if packet.Flag(Ack) {
		packet.ack = c.remoteSequence
		packet.ackBits = c.ackBits
	}

	if packet.Flag(Reliable) {
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

func (c *Connection) sendLowLevelPacket(descriptor descriptor) {
	c.sendPacket(&Packet{descriptor: descriptor}, false)
}

func (c *Connection) sendAckPacket() {
	c.sendLowLevelPacket(Ack)
}

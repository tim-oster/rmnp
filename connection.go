// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"fmt"
	"time"
	"context"
)

type connectionState uint8

const (
	Disconnected connectionState = iota
	Connected
)

type Connection struct {
	protocol *protocolImpl
	conn     *net.UDPConn
	addr     *net.UDPAddr
	state    connectionState

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

	lastSendTime       int64
	lastResendTime     int64
	lastReceivedTime   int64
	pingPacketInterval uint8
	sendBuffer         *sendBuffer
	receiveBuffer      *SequenceBuffer
	congestionHandler  *congestionHandler
}

func newConnection() *Connection {
	return &Connection{
		state:             Disconnected,
		orderedChain:      NewPacketChain(),
		sendBuffer:        NewSendBuffer(),
		receiveBuffer:     NewSequenceBuffer(SequenceBufferSize),
		congestionHandler: NewCongestionHandler(),
	}
}

func (c *Connection) init(impl *protocolImpl, addr *net.UDPAddr) {
	c.protocol = impl
	c.conn = impl.socket
	c.addr = addr
}

func (c *Connection) reset() {
	c.protocol = nil
	c.conn = nil
	c.addr = nil
	c.state = Disconnected

	c.orderedChain.reset()
	c.sendBuffer.Reset()
	c.receiveBuffer.reset()
	c.congestionHandler.reset()

	c.localSequence = 0
	c.remoteSequence = 0
	c.ackBits = 0
	c.orderedSequence = 0

	c.localUnreliableSequence = 0
	c.remoteUnreliableSequence = 0

	c.lastSendTime = 0
	c.lastResendTime = 0
	c.lastReceivedTime = 0
	c.pingPacketInterval = 0
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

		func() {
			c.protocol.waitGroup.Add(1)
			defer c.protocol.waitGroup.Done()

			currentTime := currentTime()

			if currentTime-c.lastResendTime > c.congestionHandler.mul(ResendTimeout) {
				c.lastResendTime = currentTime

				c.sendBuffer.Iterate(func(i int, data *sendPacket) SendBufferOP {
					if int64(i) >= c.congestionHandler.div(MaxPacketResends) {
						return SendBufferCancel
					}

					if currentTime-data.sendTime > SendRemoveTimeout {
						return SendBufferDelete
					} else {
						c.sendPacket(data.packet, true)
					}

					return SendBufferContinue
				})
			}

			if currentTime-c.lastSendTime > c.congestionHandler.mul(ReackTimeout) {
				c.sendAckPacket()

				if c.pingPacketInterval%AutoPingInterval == 0 {
					c.sendLowLevelPacket(Reliable)
				}

				c.pingPacketInterval++
			}
		}()
	}
}

func (c *Connection) keepAlive(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(TimeoutThreshold * time.Millisecond / 2):
		}

		func() {
			c.protocol.waitGroup.Add(1)
			defer c.protocol.waitGroup.Done()

			currentTime := currentTime()

			if currentTime-c.lastReceivedTime > TimeoutThreshold || c.GetPing() > MaxPing {
				c.protocol.timeoutClient(c)
			}
		}()
	}
}

func (c *Connection) ProcessPacket(packet []byte) {
	defer c.protocol.packetPool.Put(packet)

	c.lastReceivedTime = currentTime()

	p := NewPacket()
	defer ReleasePacket(p)

	if size := headerSize(packet); len(packet)-size > 0 {
		p.data = packet[size:]
	}

	if !p.Deserialize(packet) {
		fmt.Println("error during data deserialization")
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

	if p.data != nil {
		c.process(p)
	}
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

		for l := c.orderedChain.PopConsecutive(); l != nil; l = l.next {
			c.process(l.packet)
		}
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
			s := packet.ack - i

			if packet, found := c.sendBuffer.Retrieve(s); found {
				c.congestionHandler.check(packet.sendTime)
				fmt.Println("#", s, "acked")
			}
		}
	}

	return true
}

func (c *Connection) process(packet *Packet) {
	invokePacketCallback(c.protocol.onPacket, c, packet)
}

func (c *Connection) sendPacket(packet *Packet, resend bool) {
	defer ReleasePacket(packet)

	if !packet.Flag(Reliable) && c.congestionHandler.shouldDrop() {
		return
	}

	packet.protocolId = ProtocolId

	if !resend {
		if packet.Flag(Reliable) {
			packet.sequence = c.localSequence
			c.localSequence++

			if packet.Flag(Ordered) {
				//packet.order = c.orderedSequence
				c.orderedSequence++
			}

			c.sendBuffer.Add(packet)
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
		fmt.Print("data sequences #", packet.sequence)
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
	packet := NewPacket()
	packet.descriptor = descriptor
	c.sendPacket(packet, false)
}

func (c *Connection) sendAckPacket() {
	c.sendLowLevelPacket(Ack)
}

func (c *Connection) GetPing() int32 {
	return int32(c.congestionHandler.rtt / 2)
}

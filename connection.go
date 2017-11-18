// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"fmt"
	"time"
	"context"
	"sync"
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
	remoteSequence  sequenceNumber // TODO atomic
	ackBits         uint32         // TODO atomic
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

	sendQueue    chan *Packet
	receiveQueue chan []byte
	waitGroup    sync.WaitGroup
}

func newConnection() *Connection {
	return &Connection{
		state:             Disconnected,
		orderedChain:      NewPacketChain(),
		sendBuffer:        NewSendBuffer(),
		receiveBuffer:     NewSequenceBuffer(SequenceBufferSize),
		congestionHandler: NewCongestionHandler(),
		sendQueue:         make(chan *Packet, MaxSendReceiveQueueSize),
		receiveQueue:      make(chan []byte, MaxSendReceiveQueueSize),
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
	go c.sendUpdate(c.ctx)
	go c.receiveUpdate(c.ctx)
	go c.keepAlive(c.ctx)
}

func (c *Connection) sendUpdate(ctx context.Context) {
	c.waitGroup.Add(1)
	defer c.waitGroup.Done()

	for {
		select {
		case <-time.After(UpdateLoopInterval * time.Millisecond):
		case <-ctx.Done():
			return
		case packet := <-c.sendQueue:
			c.processSend(packet, false)
		}

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
					c.processSend(data.packet, true)
				}

				return SendBufferContinue
			})
		}

		if currentTime-c.lastSendTime > c.congestionHandler.mul(ReackTimeout) {
			c.sendAckPacket()

			if c.pingPacketInterval%AutoPingInterval == 0 {
				c.sendLowLevelPacket(Reliable)
				c.pingPacketInterval = 0
			}

			c.pingPacketInterval++
		}
	}
}

func (c *Connection) receiveUpdate(ctx context.Context) {
	c.waitGroup.Add(1)
	defer c.waitGroup.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case packet := <-c.receiveQueue:
			c.processReceive(packet)
		}
	}
}

func (c *Connection) keepAlive(ctx context.Context) {
	c.waitGroup.Add(1)
	defer c.waitGroup.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(TimeoutThreshold * time.Millisecond / 2):
		}

		currentTime := currentTime()

		if currentTime-c.lastReceivedTime > TimeoutThreshold || c.GetPing() > MaxPing {
			c.protocol.timeoutClient(c)
		}
	}
}

func (c *Connection) processReceive(packet []byte) {
	c.lastReceivedTime = currentTime()

	p := new(Packet)

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

	// sendUpdate receive states
	c.receiveBuffer.Set(packet.sequence, true)

	// sendUpdate remote sequences number
	if greaterThanSequence(packet.sequence, c.remoteSequence) && differenceSequence(packet.sequence, c.remoteSequence) <= MaxSkippedPackets {
		c.remoteSequence = packet.sequence
	}

	// sendUpdate ack bit mask for last 32 packets
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

func (c *Connection) processSend(packet *Packet, resend bool) {
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

func (c *Connection) SendPacket(packet *Packet) {
	c.sendQueue <- packet
}

func (c *Connection) sendLowLevelPacket(descriptor descriptor) {
	c.SendPacket(&Packet{descriptor: descriptor})
}

func (c *Connection) sendAckPacket() {
	c.sendLowLevelPacket(Ack)
}

func (c *Connection) GetPing() int32 {
	return int32(c.congestionHandler.rtt / 2)
}

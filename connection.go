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
	"sync/atomic"
)

type connectionState uint8

const (
	stateDisconnected connectionState = iota
	stateConnecting
	stateConnected
)

type Connection struct {
	protocol *protocolImpl
	Conn     *net.UDPConn
	Addr     *net.UDPAddr
	state    connectionState

	// for go routines
	ctx          context.Context
	stopRoutines context.CancelFunc

	// for Reliable packets
	localSequence   sequenceNumber
	remoteSequence  sequenceNumber
	ackBits         uint32
	orderedChain    *chain
	orderedSequence orderNumber

	// for Unreliable Ordered packets
	localUnreliableSequence  sequenceNumber
	remoteUnreliableSequence sequenceNumber

	lastAckSendTime    int64
	lastResendTime     int64
	lastReceivedTime   int64
	pingPacketInterval uint8
	sendBuffer         *sendBuffer
	receiveBuffer      *sequenceBuffer
	congestionHandler  *congestionHandler

	sendQueue    chan *packet
	receiveQueue chan []byte
	waitGroup    sync.WaitGroup

	Values map[string]interface{}
}

func newConnection() *Connection {
	return &Connection{
		state:             stateDisconnected,
		orderedChain:      newChain(),
		sendBuffer:        newSendBuffer(),
		receiveBuffer:     newSequenceBuffer(CfgSequenceBufferSize),
		congestionHandler: newCongestionHandler(),
		sendQueue:         make(chan *packet, CfgMaxSendReceiveQueueSize),
		receiveQueue:      make(chan []byte, CfgMaxSendReceiveQueueSize),
		Values:            make(map[string]interface{}),
	}
}

func (c *Connection) init(impl *protocolImpl, addr *net.UDPAddr) {
	c.protocol = impl
	c.Conn = impl.socket
	c.Addr = addr
	c.state = stateConnecting

	t := currentTime()
	c.lastAckSendTime = t
	c.lastResendTime = t
	c.lastReceivedTime = t
}

func (c *Connection) reset() {
	c.protocol = nil
	c.Conn = nil
	c.Addr = nil
	c.state = stateDisconnected

	c.orderedChain.reset()
	c.sendBuffer.reset()
	c.receiveBuffer.reset()
	c.congestionHandler.reset()

	c.localSequence = 0
	c.remoteSequence = 0
	c.ackBits = 0
	c.orderedSequence = 0

	c.localUnreliableSequence = 0
	c.remoteUnreliableSequence = 0

	c.lastAckSendTime = 0
	c.lastResendTime = 0
	c.lastReceivedTime = 0
	c.pingPacketInterval = 0

	for {
		select {
		case <-c.sendQueue:
		case <-c.receiveQueue:
		default:
			break
		}
	}

	c.Values = make(map[string]interface{})
}

func (c *Connection) startRoutines() {
	c.ctx, c.stopRoutines = context.WithCancel(context.Background())
	go c.sendUpdate()
	go c.receiveUpdate()
	go c.keepAlive()
}

func (c *Connection) sendUpdate() {
	defer antiPanic(c.sendUpdate)

	c.waitGroup.Add(1)
	defer c.waitGroup.Done()

	for {
		select {
		case <-time.After(CfgUpdateLoopInterval * time.Millisecond):
		case <-c.ctx.Done():
			return
		case packet := <-c.sendQueue:
			c.processSend(packet, false)
		}

		currentTime := currentTime()

		if currentTime-c.lastResendTime > c.congestionHandler.ResendTimeout {
			c.lastResendTime = currentTime

			c.sendBuffer.iterate(func(i int, data *sendPacket) sendBufferOP {
				if int64(i) >= c.congestionHandler.MaxPacketResends {
					return sendBufferCancel
				}

				if currentTime-data.sendTime > CfgSendRemoveTimeout {
					return sendBufferDelete
				} else {
					c.processSend(data.packet, true)
				}

				return sendBufferContinue
			})
		}

		if c.state != stateConnected {
			continue
		}

		if currentTime-c.lastAckSendTime > c.congestionHandler.ReackTimeout {
			c.sendAckPacket()

			if c.pingPacketInterval%CfgAutoPingInterval == 0 {
				c.sendLowLevelPacket(descReliable | descAck)
				c.pingPacketInterval = 0
			}

			c.pingPacketInterval++
		}
	}
}

func (c *Connection) receiveUpdate() {
	defer antiPanic(c.receiveUpdate)

	c.waitGroup.Add(1)
	defer c.waitGroup.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case packet := <-c.receiveQueue:
			c.processReceive(packet)
		}
	}
}

func (c *Connection) keepAlive() {
	defer antiPanic(c.keepAlive)

	c.waitGroup.Add(1)
	defer c.waitGroup.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(CfgTimeoutThreshold * time.Millisecond / 2):
		}

		if c.state == stateDisconnected {
			continue
		}

		currentTime := currentTime()

		if currentTime-c.lastReceivedTime > int64(CfgTimeoutThreshold) || c.GetPing() > CfgMaxPing {
			// needs to be executed in goroutine; otherwise this method could not exit and therefore deadlock
			// the connection's waitGroup
			go func() {
				defer antiPanic(nil)
				c.protocol.timeoutClient(c)
			}()
		}
	}
}

func (c *Connection) processReceive(buffer []byte) {
	c.lastReceivedTime = currentTime()

	p := new(packet)

	if size := headerSize(buffer); len(buffer)-size > 0 {
		p.data = buffer[size:]
	}

	if !p.deserialize(buffer) {
		fmt.Println("error during data deserialization")
		return
	}

	if p.flag(descReliable) && !c.handleReliablePacket(p) {
		return
	}

	if p.flag(descAck) && !c.handleAckPacket(p) {
		return
	}

	if p.flag(descOrdered) && !c.handleOrderedPacket(p) {
		return
	}

	c.process(p)
}

func (c *Connection) handleReliablePacket(packet *packet) bool {
	fmt.Println("recveived sequences #", packet.sequence)

	if c.receiveBuffer.get(packet.sequence) {
		fmt.Println(":: was duplicate")
		return false
	}

	// sendUpdate receive states
	c.receiveBuffer.set(packet.sequence, true)

	// sendUpdate remote sequences number
	if greaterThanSequence(packet.sequence, c.remoteSequence) && differenceSequence(packet.sequence, c.remoteSequence) <= CfgMaxSkippedPackets {
		c.remoteSequence = packet.sequence
	}

	// sendUpdate ack bit mask for last 32 packets
	c.ackBits = 0
	for i := sequenceNumber(1); i <= 32; i++ {
		if c.receiveBuffer.get(c.remoteSequence - i) {
			c.ackBits |= 1 << (i - 1)
		}
	}

	c.sendAckPacket()

	return true
}

func (c *Connection) handleOrderedPacket(packet *packet) bool {
	if packet.flag(descReliable) {
		c.orderedChain.chain(packet)

		for l := c.orderedChain.popConsecutive(); l != nil; l = l.next {
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

func (c *Connection) handleAckPacket(packet *packet) bool {
	for i := sequenceNumber(0); i <= 32; i++ {
		if i == 0 || packet.ackBits&(1<<(i-1)) != 0 {
			s := packet.ack - i

			if packet, found := c.sendBuffer.retrieve(s); found {
				if !packet.noRTT {
					c.congestionHandler.check(packet.sendTime)
				}

				fmt.Println("#", s, "acked")
			}
		}
	}

	return true
}

func (c *Connection) process(packet *packet) {
	if packet.data != nil && len(packet.data) > 0 {
		invokePacketCallback(c.protocol.onPacket, c, packet.data)
	}
}

func (c *Connection) processSend(packet *packet, resend bool) {
	if !packet.flag(descReliable) && c.congestionHandler.shouldDropUnreliable() {
		return
	}

	packet.protocolId = CfgProtocolId

	if !resend {
		if packet.flag(descReliable) {
			packet.sequence = c.localSequence
			c.localSequence++

			if packet.flag(descOrdered) {
				packet.order = c.orderedSequence
				c.orderedSequence++
			}

			c.sendBuffer.add(packet, c.state != stateConnected)
		} else if packet.flag(descOrdered) {
			packet.sequence = c.localUnreliableSequence
			c.localUnreliableSequence++
		}
	}

	if packet.flag(descAck) {
		c.lastAckSendTime = currentTime()
		packet.ack = c.remoteSequence
		packet.ackBits = c.ackBits
	}

	if packet.flag(descReliable) {
		fmt.Print("data sequences #", packet.sequence)
		if resend {
			fmt.Println(" resend")
		} else {
			fmt.Println()
		}
	}

	packet.calculateHash()
	buffer := packet.serialize()
	c.protocol.writeFunc(c, buffer)
	atomic.AddUint64(&StatSendBytes, uint64(len(buffer)))
}

func (c *Connection) sendPacket(packet *packet) {
	c.sendQueue <- packet
}

func (c *Connection) sendLowLevelPacket(descriptor descriptor) {
	c.sendPacket(&packet{descriptor: descriptor})
}

func (c *Connection) sendHighLevelPacket(descriptor descriptor, data []byte) {
	c.sendPacket(&packet{descriptor: descriptor, data: data})
}

func (c *Connection) sendAckPacket() {
	c.sendLowLevelPacket(descAck)
}

func (c *Connection) SendUnreliable(data []byte) {
	c.sendHighLevelPacket(0, data)
}

func (c *Connection) SendUnreliableOrdered(data []byte) {
	c.sendHighLevelPacket(descOrdered, data)
}

func (c *Connection) SendReliable(data []byte) {
	c.sendHighLevelPacket(descReliable|descAck, data)
}

func (c *Connection) SendReliableOrdered(data []byte) {
	c.sendHighLevelPacket(descReliable|descAck|descOrdered, data)
}

func (c *Connection) GetPing() int16 {
	return int16(c.congestionHandler.rtt / 2)
}

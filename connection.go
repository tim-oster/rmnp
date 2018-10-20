// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type connectionState uint8

const (
	stateDisconnected connectionState = iota
	stateConnecting
	stateConnected
)

// Channel is the method of sending packets
type Channel byte

const (
	// ChannelUnreliable is fast delivery without any guarantee on arrival or order
	ChannelUnreliable Channel = iota
	// ChannelUnreliableOrdered is the same as ChannelUnreliable but only the most recent packet is accepted
	ChannelUnreliableOrdered
	// ChannelReliable guarantees packets to arrive but not in order
	ChannelReliable
	// ChannelReliableOrdered guarantees packets to arrive in order (mimics TCP)
	ChannelReliableOrdered
)

// Connection is a udp connection and handles sending of packets
type Connection struct {
	protocol *protocolImpl

	stateMutex sync.RWMutex
	state      connectionState

	Conn     *net.UDPConn
	Addr     *net.UDPAddr
	IsServer bool

	// for go routines
	ctx          context.Context
	stopRoutines context.CancelFunc

	// for reliable packets
	localSequence   sequenceNumber
	remoteSequence  sequenceNumber
	ackBits         uint32
	orderedChain    *chain
	orderedSequence orderNumber

	// for unreliable ordered packets
	localUnreliableSequence  sequenceNumber
	remoteUnreliableSequence sequenceNumber

	lastAckSendTime    int64
	lastResendTime     int64
	lastReceivedTime   int64
	lastChainTime      int64
	pingPacketInterval uint8
	sendBuffer         *sendBuffer
	receiveBuffer      *sequenceBuffer
	congestionHandler  *congestionHandler

	sendQueue    *dropChannel //chan *packet
	receiveQueue *dropChannel //chan []byte
	waitGroup    sync.WaitGroup

	values      map[byte]interface{}
	valuesMutex sync.RWMutex
}

func newConnection() *Connection {
	return &Connection{
		state:             stateDisconnected,
		orderedChain:      newChain(CfgMaxPacketChainLength),
		sendBuffer:        newSendBuffer(),
		receiveBuffer:     newSequenceBuffer(CfgSequenceBufferSize),
		congestionHandler: newCongestionHandler(),
		sendQueue:         newDropChannel(make(chan interface{}, CfgMaxSendReceiveQueueSize)),
		receiveQueue:      newDropChannel(make(chan interface{}, CfgMaxSendReceiveQueueSize)),
		values:            make(map[byte]interface{}),
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
	c.state = stateDisconnected

	c.Conn = nil
	c.Addr = nil
	c.IsServer = false

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
	c.lastChainTime = 0
	c.pingPacketInterval = 0

	c.sendQueue.clear()
	c.receiveQueue.clear()

	c.values = make(map[byte]interface{})
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

	atomic.AddUint64(&StatRunningGoRoutines, 1)
	defer atomic.AddUint64(&StatRunningGoRoutines, ^uint64(0))

	for {
		select {
		case <-time.After(CfgUpdateLoopTimeout * time.Millisecond):
		case <-c.ctx.Done():
			return
		case p := <-c.sendQueue.channel:
			c.processSend(p.(*packet), false)
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
				}

				c.processSend(data.packet, true)
				return sendBufferContinue
			})
		}

		if c.getState() != stateConnected {
			continue
		}

		if currentTime-c.lastChainTime > CfgChainSkipTimeout {
			c.orderedChain.skip()
			c.handleNextChainSequence()
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

	atomic.AddUint64(&StatRunningGoRoutines, 1)
	defer atomic.AddUint64(&StatRunningGoRoutines, ^uint64(0))

	for {
		select {
		case <-c.ctx.Done():
			return
		case p := <-c.receiveQueue.channel:
			c.processReceive(p.([]byte))
		}
	}
}

func (c *Connection) keepAlive() {
	defer antiPanic(c.keepAlive)

	c.waitGroup.Add(1)
	defer c.waitGroup.Done()

	atomic.AddUint64(&StatRunningGoRoutines, 1)
	defer atomic.AddUint64(&StatRunningGoRoutines, ^uint64(0))

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After((CfgTimeoutThreshold / 2) * time.Millisecond):
		}

		if c.getState() == stateDisconnected {
			continue
		}

		currentTime := currentTime()

		if currentTime-c.lastReceivedTime > int64(CfgTimeoutThreshold) || c.GetPing() > CfgMaxPing {
			// needs to be executed in goroutine; otherwise this method could not exit and therefore deadlock
			// the connection's waitGroup
			go func() {
				defer antiPanic(nil)
				c.protocol.disconnectClient(c, disconnectTypeTimeout, nil)
			}()
		}
	}
}

func (c *Connection) processReceive(buffer []byte) {
	c.lastReceivedTime = currentTime()

	p := new(packet)

	if !p.deserialize(buffer) {
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

	var ch Channel

	if p.flag(descReliable) {
		if p.flag(descOrdered) {
			ch = ChannelReliableOrdered
		} else {
			ch = ChannelReliable
		}
	} else {
		if p.flag(descOrdered) {
			ch = ChannelUnreliableOrdered
		} else {
			ch = ChannelUnreliable
		}
	}

	c.process(p, ch)
}

func (c *Connection) handleReliablePacket(packet *packet) bool {
	if c.receiveBuffer.get(packet.sequence) {
		return false
	}

	c.receiveBuffer.set(packet.sequence, true)

	if greaterThanSequence(packet.sequence, c.remoteSequence) && differenceSequence(packet.sequence, c.remoteSequence) <= CfgMaxSkippedPackets {
		c.remoteSequence = packet.sequence
	}

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
		c.handleNextChainSequence()
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
			}
		}
	}

	return true
}

func (c *Connection) process(packet *packet, channel Channel) {
	if packet.data != nil && len(packet.data) > 0 {
		invokePacketCallback(c.protocol.onPacket, c, packet.data, channel)
	}
}

func (c *Connection) handleNextChainSequence() {
	c.lastChainTime = currentTime()

	for l := c.orderedChain.popConsecutive(); l != nil; l = l.next {
		c.process(l.packet, ChannelReliableOrdered)
	}
}

func (c *Connection) processSend(packet *packet, resend bool) {
	if !packet.flag(descReliable) && c.congestionHandler.shouldDropUnreliable() {
		return
	}

	packet.protocolID = CfgProtocolID

	if !resend {
		if packet.flag(descReliable) {
			packet.sequence = c.localSequence
			c.localSequence++

			if packet.flag(descOrdered) {
				packet.order = c.orderedSequence
				c.orderedSequence++
			}

			c.sendBuffer.add(packet, c.getState() != stateConnected)
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

	packet.calculateHash()
	buffer := packet.serialize()
	c.protocol.writeFunc(c.Conn, c.Addr, buffer)
	atomic.AddUint64(&StatSendBytes, uint64(len(buffer)))
}

func (c *Connection) sendPacket(packet *packet) {
	c.sendQueue.push(packet)
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

func (c *Connection) getState() connectionState {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state
}

func (c *Connection) setState(state connectionState) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	c.state = state
}

func (c *Connection) updateState(state connectionState) bool {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	if c.state != state {
		c.state = state
		return true
	}

	return false
}

// SendUnreliable sends the data with no guarantee whether it arrives or not.
// Note that the packets or not guaranteed to arrive in order.
func (c *Connection) SendUnreliable(data []byte) {
	c.sendHighLevelPacket(0, data)
}

// SendUnreliableOrdered is the same as SendUnreliable but guarantees that if packets
// do not arrive chronologically the receiver only accepts newer packets and discards older
// ones.
func (c *Connection) SendUnreliableOrdered(data []byte) {
	c.sendHighLevelPacket(descOrdered, data)
}

// SendReliable send the data and guarantees that the data arrives.
// Note that packets are not guaranteed to arrive in the order they were sent.
// This method is not 100% reliable. (Read more in README)
func (c *Connection) SendReliable(data []byte) {
	c.sendHighLevelPacket(descReliable|descAck, data)
}

// SendReliableOrdered is the same as SendReliable but guarantees that packets
// will be processed in order.
// This method is not 100% reliable. (Read more in README)
func (c *Connection) SendReliableOrdered(data []byte) {
	c.sendHighLevelPacket(descReliable|descAck|descOrdered, data)
}

// SendOnChannel sends the data on the given channel using the dedicated send method
// for each channel
func (c *Connection) SendOnChannel(channel Channel, data []byte) {
	switch channel {
	case ChannelUnreliable:
		c.SendUnreliable(data)
	case ChannelUnreliableOrdered:
		c.SendUnreliableOrdered(data)
	case ChannelReliable:
		c.SendReliable(data)
	case ChannelReliableOrdered:
		c.SendReliableOrdered(data)
	}
}

// GetPing returns the current ping to this connection's socket
func (c *Connection) GetPing() int16 {
	return int16(c.congestionHandler.rtt / 2)
}

// Disconnect disconnects the connection
func (c *Connection) Disconnect(packet []byte) {
	go c.protocol.disconnectClient(c, disconnectTypeDefault, packet)
}

// Set stores a value associated with the given key in this connection instance.
// It is thread safe.
func (c *Connection) Set(key byte, value interface{}) {
	c.valuesMutex.Lock()
	defer c.valuesMutex.Unlock()
	c.values[key] = value
}

// TrySet stores a value associated with the given key in this connection instance if does not exist yet.
// It returns whether it was able to set the value. It is thread safe.
func (c *Connection) TrySet(key byte, value interface{}) bool {
	c.valuesMutex.Lock()
	defer c.valuesMutex.Unlock()

	if _, f := c.values[key]; !f {
		c.values[key] = value
		return true
	}

	return false
}

// Get retrieves a stored value from this connection instance and returns if it exists.
// It is thread safe.
func (c *Connection) Get(key byte) (interface{}, bool) {
	c.valuesMutex.RLock()
	defer c.valuesMutex.RUnlock()
	v, f := c.values[key]
	return v, f
}

// GetFallback retrieves a stored value from this connection instance.
// It is thread safe.
func (c *Connection) GetFallback(key byte, fallback interface{}) interface{} {
	if v, f := c.Get(key); f {
		return v
	}
	return fallback
}

// Del deletes a stored value from this connection instance.
// It is thread safe.
func (c *Connection) Del(key byte) {
	c.valuesMutex.Lock()
	defer c.valuesMutex.Unlock()
	delete(c.values, key)
}

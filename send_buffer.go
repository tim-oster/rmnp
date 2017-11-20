// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "sync"

type sendBufferOP byte

const (
	sendBufferDelete   sendBufferOP = iota
	sendBufferCancel
	sendBufferContinue
)

type sendPacket struct {
	packet   *packet
	sendTime int64
	noRTT    bool
}

type sendBuffer struct {
	head  *sendBufferElement
	tail  *sendBufferElement
	mutex sync.Mutex
}

type sendBufferElement struct {
	next *sendBufferElement
	prev *sendBufferElement
	data sendPacket
}

func newSendBuffer() *sendBuffer {
	return new(sendBuffer)
}

func (buffer *sendBuffer) reset() {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	buffer.head = nil
	buffer.tail = nil
}

func (buffer *sendBuffer) add(packet *packet, noRTT bool) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	e := &sendBufferElement{data: sendPacket{
		packet:   packet,
		sendTime: currentTime(),
		noRTT:    noRTT,
	}}

	if buffer.head == nil {
		buffer.head = e
		buffer.tail = e
	} else {
		e.prev = buffer.tail
		buffer.tail.next = e
		buffer.tail = e
	}
}

func (buffer *sendBuffer) remove(e *sendBufferElement) {
	if e.prev == nil {
		buffer.head = e.next
	} else {
		e.prev.next = e.next
	}

	if e.next == nil {
		buffer.tail = e.prev
	} else {
		e.next.prev = e.prev
	}
}

func (buffer *sendBuffer) retrieve(sequence sequenceNumber) (sendPacket, bool) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	for e := buffer.head; e != nil; e = e.next {
		if e.data.packet.sequence == sequence {
			buffer.remove(e)
			return e.data, true
		}
	}

	var packet sendPacket
	return packet, false
}

func (buffer *sendBuffer) iterate(iterator func(int, *sendPacket) sendBufferOP) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	for i, e := 0, buffer.head; e != nil; i, e = i+1, e.next {
		switch iterator(i, &e.data) {
		case sendBufferDelete:
			buffer.remove(e)
		case sendBufferCancel:
			return
		case sendBufferContinue:
		}
	}
}

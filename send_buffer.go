package rmnp

import "sync"

type SendBufferOP byte

const (
	SendBufferDelete   SendBufferOP = iota
	SendBufferCancel
	SendBufferContinue
)

type sendPacket struct {
	packet   *Packet
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

func NewSendBuffer() *sendBuffer {
	return new(sendBuffer)
}

func (buffer *sendBuffer) Reset() {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	buffer.head = nil
	buffer.tail = nil
}

func (buffer *sendBuffer) Add(packet *Packet, noRTT bool) {
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

func (buffer *sendBuffer) Retrieve(sequence sequenceNumber) (sendPacket, bool) {
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

func (buffer *sendBuffer) Iterate(iterator func(int, *sendPacket) SendBufferOP) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	for i, e := 0, buffer.head; e != nil; i, e = i+1, e.next {
		switch iterator(i, &e.data) {
		case SendBufferDelete:
			buffer.remove(e)
		case SendBufferCancel:
			return
		case SendBufferContinue:
		}
	}
}

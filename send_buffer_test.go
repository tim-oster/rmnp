package rmnp

import (
	"testing"
)

func TestSendBufferAddRemove(t *testing.T) {
	b := newSendBuffer()

	b.add(&packet{sequence: 0}, false)

	if b.head == nil || b.head != b.tail {
		t.Error("Expected head to be the same as tail")
	}

	if s := b.head.data.packet.sequence; s != 0 {
		t.Errorf("Expected first element to contain sequence number 0 not %v", s)
	}

	b.add(&packet{sequence: 1}, false)

	if s := b.tail.data.packet.sequence; s != 1 {
		t.Errorf("Expected last element to contain sequence number 1 not %v", s)
	}

	b.add(&packet{sequence: 2}, false)

	if s := b.tail.data.packet.sequence; s != 2 {
		t.Errorf("Expected last element to contain sequence number 2 not %v", s)
	}

	if b.head.next != b.tail.prev || b.head.next.prev != b.head || b.tail.prev.next != b.tail {
		t.Error("Expected middle element to be interlinked with adjacent elements")
	}

	b.remove(b.head.next)

	if b.head.next != b.tail || b.tail.prev != b.head {
		t.Error("Expected both remaining elements to be interlinked")
	}

	b.remove(b.tail)

	if b.tail != b.head {
		t.Error("Expected tail to be the same as head after removing second last element")
	}

	b.remove(b.head)

	if b.head != nil || b.tail != nil {
		t.Error("Expected head and tail to be nil in an empty buffer")
	}
}

func TestSendBufferRetrieve(t *testing.T) {
	b := newSendBuffer()

	for i := 0; i < 10; i++ {
		b.add(&packet{sequence: sequenceNumber(i)}, false)
	}

	for i := 9; i >= 0; i-- {
		p, f := b.retrieve(sequenceNumber(i))

		if !f {
			t.Errorf("Expected an element for sequence %v", i)
			break
		}

		if p.packet.sequence != sequenceNumber(i) {
			t.Errorf("Expected retrieved packet's sequence number to be %v not %v", i, p.packet.sequence)
			break
		}

		if (i > 0 && b.tail.data.packet.sequence != sequenceNumber(i-1)) || (i == 0 && b.head != nil && b.tail != nil) {
			t.Error("Expected tail to be the next element to be retrieved")
			break
		}
	}
}

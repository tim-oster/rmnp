package rmnp

import "testing"

func TestDropChannel(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Error(err)
		}
	}()

	c := newDropChannel(make(chan interface{}, 2))

	c.push(10)
	if i := <-c.channel; i != 10 {
		t.Error("Channel wrapping does not work")
		return
	}

	values := []byte{2, 4, 8, 16}
	for _, b := range values {
		c.push(b)
	}
	for i := 2; i <= 3; i++ {
		if v, r := <-c.channel, values[i]; v != r {
			t.Errorf("Expected first value to be %v not %v", r, i)
		}
	}
}

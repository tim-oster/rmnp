package rmnp

import "fmt"

type dropChannel struct {
	channel chan interface{}
}

func newDropChannel(channel chan interface{}) *dropChannel {
	c := new(dropChannel)
	c.channel = channel
	return c
}

func (c *dropChannel) push(i interface{}) {
	if l := len(c.channel); l > 0 && l == cap(c.channel) {
		fmt.Println("dropping packets from dropChannel")
		<-c.channel
	}

	select {
	case c.channel <- i:
	default:
		panic("drop channel is blocking")
	}
}

func (c *dropChannel) clear() {
loop:
	for {
		select {
		case <-c.channel:
		default:
			break loop
		}
	}
}

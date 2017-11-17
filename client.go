// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"fmt"
	"time"
)

type Client struct {
	protocolImpl

	server *Connection

	// TODO tmp
	stop bool
}

func NewClient(server string) *Client {
	c := new(Client)

	c.readFunc = func(conn *net.UDPConn, buffer []byte) (int, *net.UDPAddr, bool) {
		length, err := conn.Read(buffer)

		if err != nil {
			return 0, nil, false
		}

		return length, c.server.addr, true
	}

	c.writeFunc = func(c *Connection, buffer []byte) {
		c.conn.Write(buffer)
	}

	c.onConnect = func(connection *Connection) {
		fmt.Println("connected to server")
	}

	c.onDisconnect = func(connection *Connection) {
		fmt.Println("disconnected from server")
		c.stop = true
		c.destroy()
	}

	c.onTimeout = func(connection *Connection) {
		fmt.Println("timeout")
	}

	c.onValidation = func(connection *Connection, addr *net.UDPAddr, packet []byte) bool {
		return false
	}

	c.onPacket = func(connection *Connection, packet *Packet) {
		fmt.Println(string(packet.data))
	}

	c.init(server)
	return c
}

func (c *Client) Connect() {
	c.setSocket(net.DialUDP("udp", nil, c.address))
	c.listen()
	c.server = c.connectClient(c.socket.RemoteAddr().(*net.UDPAddr))
}

func (c *Client) Disconnect() {
	c.destroy()
	c.server = nil
}

// TODO tmp
func (c *Client) Send() {
	/*
	for {
		if c.stop {
			break
		}

		c.server.sendPacket(&Packet{
			descriptor: Reliable,
			data:       []byte("hi"),
		}, false)
		time.Sleep(500 * time.Millisecond)
	}*/

	time.Sleep(500 * time.Millisecond)

	c.testSend(1)
	c.testSend(4)
	c.testSend(3)
	c.testSend(2)
	c.testSend(0)
}

func (c *Client) testSend(id byte) {
	c.server.sendPacket(&Packet{
		descriptor: Reliable | Ordered,
		order:      orderNumber(id),
		data:       []byte{id},
	}, false)
}

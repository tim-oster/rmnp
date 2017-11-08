// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"time"
)

type Client struct {
	protocolImpl

	server *Connection
}

func NewClient(server string) *Client {
	c := new(Client)
	c.protocolImpl.init(server, func(c *Connection, buffer []byte) {
		c.conn.Write(buffer)
	})
	return c
}

func (c *Client) Connect() {
	socket, err := net.DialUDP("udp", nil, c.address)
	checkError("Cannot connect to server", err)
	c.socket = socket
	c.server = c.protocolImpl.retrieveConnection(socket.RemoteAddr().(*net.UDPAddr))
	go c.protocolImpl.listen()
}

func (c *Client) Disconnect() {
	c.socket.Close()
}

func (c *Client) Send() {
	for {
		c.server.sendPacket(&Packet{descriptor: Reliable}, false)
		time.Sleep(600 * time.Millisecond)
	}
}

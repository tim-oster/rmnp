// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
)

type Client struct {
	protocolImpl

	server *Connection
}

func NewClient(server string) *Client {
	c := new(Client)

	c.protocolImpl.readFunc = func(conn *net.UDPConn, buffer []byte) (int, *net.UDPAddr, bool) {
		length, err := conn.Read(buffer)

		if err != nil {
			return 0, nil, false
		}

		return length, c.server.addr, true
	}

	c.protocolImpl.writeFunc = func(c *Connection, buffer []byte) {
		c.conn.Write(buffer)
	}

	c.protocolImpl.init(server)
	return c
}

func (c *Client) Connect() {
	socket, err := net.DialUDP("udp", nil, c.address)
	checkError("Cannot connect to server", err)

	c.socket = socket
	c.server = c.protocolImpl.connectClient(socket.RemoteAddr().(*net.UDPAddr))

	c.protocolImpl.listen()
	c.server.sendLowLevelPacket(Reliable | Connect)
}

func (c *Client) Disconnect() {
	c.protocolImpl.destroy()
	c.server = nil
}

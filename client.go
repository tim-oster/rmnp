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

	AddConnectionCallback(&c.onConnect, func(connection *Connection) {
		fmt.Println("connected to server")
	})

	AddConnectionCallback(&c.onDisconnect, func(connection *Connection) {
		fmt.Println("disconnected from server")
	})

	AddConnectionCallback(&c.onTimeout, func(connection *Connection) {
		fmt.Println("timeout")
	})

	c.init(server)
	return c
}

func (c *Client) Connect() {
	socket, err := net.DialUDP("udp", nil, c.address)
	checkError("Cannot connect to server", err)
	c.socket = socket
	c.listen()
	c.server = c.connectClient(socket.RemoteAddr().(*net.UDPAddr))
}

func (c *Client) Disconnect() {
	c.destroy()
	c.server = nil
}

func (c *Client) Send() {
	for {
		c.server.sendLowLevelPacket(Reliable)
		time.Sleep(500 * time.Millisecond)
	}
}

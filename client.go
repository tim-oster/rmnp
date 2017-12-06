// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "net"

type Client struct {
	protocolImpl

	// Server is the Connection to the server (nil if not connected).
	Server *Connection

	// ServerConnect is called when a connection to the server was established.
	ServerConnect ConnectionCallback

	// ServerDisconnect is called when the server disconnected the client.
	ServerDisconnect ConnectionCallback

	// ServerTimeout is called when the connection to the server timed out.
	ServerTimeout ConnectionCallback

	// PacketHandler is called when packets arrive to handle the received data.
	PacketHandler PacketCallback
}

// NewClient creates and returns a new Client instance that will try to connect
// to the given server address. It does not connect automatically.
func NewClient(server string) *Client {
	c := new(Client)

	c.readFunc = func(conn *net.UDPConn, buffer []byte) (int, *net.UDPAddr, bool) {
		length, err := conn.Read(buffer)

		if err != nil {
			return 0, nil, false
		}

		return length, c.Server.Addr, true
	}

	c.writeFunc = func(c *Connection, buffer []byte) {
		c.Conn.Write(buffer)
	}

	c.onConnect = func(connection *Connection, packet []byte) {
		if c.ServerConnect != nil {
			c.ServerConnect(connection, packet)
		}
	}

	c.onDisconnect = func(connection *Connection, packet []byte) {
		if c.ServerDisconnect != nil {
			c.ServerDisconnect(connection, packet)
		}

		c.destroy()
	}

	c.onTimeout = func(connection *Connection, packet []byte) {
		if c.ServerTimeout != nil {
			c.ServerTimeout(connection, packet)
		}
	}

	c.onValidation = func(addr *net.UDPAddr, packet []byte) bool {
		return false
	}

	c.onPacket = func(connection *Connection, packet []byte, channel Channel) {
		if c.PacketHandler != nil {
			c.PacketHandler(connection, packet, channel)
		}
	}

	c.init(server)
	return c
}

// Connect tries to connect to the server specified in the NewClient call. This call is async.
// On successful connection the Client.ServerConnect callback is invoked.
// If no connection can be established after CfgTimeoutThreshold milliseconds
// Client.ServerTimeout is called.
func (c *Client) Connect() {
	c.setSocket(net.DialUDP("udp", nil, c.address))
	c.listen()
	c.Server = c.connectClient(c.socket.RemoteAddr().(*net.UDPAddr))
}

// Disconnect immediately disconnects from the server. It invokes no callbacks.
// This call could take some time because it waits for goroutines to exit.
func (c *Client) Disconnect() {
	c.destroy()
	c.Server = nil
}

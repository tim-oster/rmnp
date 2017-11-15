// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"fmt"
)

type Server struct {
	protocolImpl
}

func NewServer(address string) *Server {
	s := new(Server)

	s.readFunc = func(conn *net.UDPConn, buffer []byte) (int, *net.UDPAddr, bool) {
		length, addr, err := conn.ReadFromUDP(buffer)

		if err != nil {
			return 0, nil, false
		}

		return length, addr, true
	}

	s.writeFunc = func(c *Connection, buffer []byte) {
		c.conn.WriteToUDP(buffer, c.addr)
	}

	s.onConnect = func(connection *Connection) {
		fmt.Println("client connected:", connection.addr)
	}

	s.onDisconnect = func(connection *Connection) {
		fmt.Println("client disconnect:", connection.addr)
	}

	s.onTimeout = func(connection *Connection) {
		fmt.Println("timeout:", connection.addr)
	}

	s.onValidation = func(connection *Connection, addr *net.UDPAddr, packet []byte) bool {
		return true
	}

	s.init(address)
	return s
}

func (s *Server) Start() {
	socket, err := net.ListenUDP("udp", s.address)
	checkError("Cannot listen on udp", err)
	socket.SetReadBuffer(MTU)
	socket.SetWriteBuffer(MTU)
	s.socket = socket
	s.listen()
}

func (s *Server) Stop() {
	s.destroy()
}

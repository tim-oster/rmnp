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

	ClientConnect    ConnectionCallback
	ClientDisconnect ConnectionCallback
	ClientTimeout    ConnectionCallback
	ClientValidation ValidationCallback
	PacketHandler    PacketCallback
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
		c.Conn.WriteToUDP(buffer, c.Addr)
	}

	s.onConnect = func(connection *Connection) {
		if s.ClientConnect != nil {
			s.ClientConnect(connection)
		}
	}

	s.onDisconnect = func(connection *Connection) {
		if s.ClientDisconnect != nil {
			s.ClientDisconnect(connection)
		}
	}

	s.onTimeout = func(connection *Connection) {
		if s.ClientTimeout != nil {
			s.ClientTimeout(connection)
		}
	}

	s.onValidation = func(connection *Connection, addr *net.UDPAddr, packet []byte) bool {
		if s.ClientValidation != nil {
			return s.ClientValidation(connection, addr, packet)
		}

		return true
	}

	s.onPacket = func(connection *Connection, packet []byte) {
		if s.PacketHandler != nil {
			s.PacketHandler(connection, packet)
		}
	}

	s.init(address)
	return s
}

func (s *Server) Start() {
	s.setSocket(net.ListenUDP("udp", s.address))
	s.listen()
}

func (s *Server) Stop() {
	s.destroy()
}

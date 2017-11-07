// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "net"

type Server struct {
	protocolImpl
}

func NewServer(address string) *Server {
	s := new(Server)
	s.protocolImpl.init(address, func(c *Connection, buffer []byte) {
		c.conn.WriteToUDP(buffer, c.addr)
	})
	return s
}

func (s *Server) Start() {
	socket, err := net.ListenUDP("udp", s.address)
	checkError("Cannot listen on udp", err)
	s.socket = socket
	s.protocolImpl.listen()
}

func (s *Server) Stop() {
	s.socket.Close()
}

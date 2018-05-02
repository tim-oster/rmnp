// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "net"

// Server listens for incoming rmnp packets and manages client connections
type Server struct {
	protocolImpl

	// ClientConnect is invoked when a new client connects.
	ClientConnect ConnectionCallback

	// ClientDisconnect is invoked when a client disconnects.
	ClientDisconnect ConnectionCallback

	// ClientTimeout is called when a client timed out. After that ClientDisconnect will be called.
	ClientTimeout ConnectionCallback

	// ClientValidation is called when a new client connects to either accept or deny the connection attempt.
	ClientValidation ValidationCallback

	// PacketHandler is called when packets arrive to handle the received data.
	PacketHandler PacketCallback
}

// NewServer creates and returns a new Server instance that will listen on the
// specified address and port. It does not start automatically.
func NewServer(address string) *Server {
	s := new(Server)

	s.readFunc = func(conn *net.UDPConn, buffer []byte) (int, *net.UDPAddr, bool) {
		length, addr, err := conn.ReadFromUDP(buffer)

		if err != nil {
			return 0, nil, false
		}

		return length, addr, true
	}

	s.writeFunc = func(conn *net.UDPConn, addr *net.UDPAddr, buffer []byte) {
		conn.WriteToUDP(buffer, addr)
	}

	s.onConnect = func(connection *Connection, packet []byte) {
		if s.ClientConnect != nil {
			s.ClientConnect(connection, packet)
		}
	}

	s.onDisconnect = func(connection *Connection, packet []byte) {
		if s.ClientDisconnect != nil {
			s.ClientDisconnect(connection, packet)
		}
	}

	s.onTimeout = func(connection *Connection, packet []byte) {
		if s.ClientTimeout != nil {
			s.ClientTimeout(connection, packet)
		}
	}

	s.onValidation = func(addr *net.UDPAddr, packet []byte) bool {
		if s.ClientValidation != nil {
			return s.ClientValidation(addr, packet)
		}

		return true
	}

	s.onPacket = func(connection *Connection, packet []byte, channel Channel) {
		if s.PacketHandler != nil {
			s.PacketHandler(connection, packet, channel)
		}
	}

	s.init(address)
	return s
}

// Start starts the server asynchronously. It invokes no callbacks but
// the server is guaranteed to be running after this call.
func (s *Server) Start() {
	s.setSocket(net.ListenUDP("udp", s.address))
	s.listen()
}

// Stop stops the server and disconnects all clients. It invokes no callbacks.
// This call could take some time because it waits for goroutines to exit.
func (s *Server) Stop() {
	s.destroy()
}

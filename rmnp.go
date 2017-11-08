// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"fmt"
	"github.com/joaojeronimo/go-crc16"
)

type protocolImpl struct {
	address *net.UDPAddr
	socket  *net.UDPConn

	connections map[uint16]*Connection
	writerFunc  func(*Connection, []byte)
}

func (impl *protocolImpl) init(address string, writerFunc func(*Connection, []byte)) {
	addr, err := net.ResolveUDPAddr("udp", address)
	checkError("Failed to resolve udp address", err)
	impl.address = addr

	impl.connections = make(map[uint16]*Connection)
	impl.writerFunc = writerFunc
}

func (impl *protocolImpl) listen() {
	for {
		// TODO pool?
		buffer := make([]byte, MTU)
		length, addr, _ := impl.socket.ReadFromUDP(buffer)

		// TODO handle in go-routine?
		packet := buffer[:length]

		if !validateHeader(packet) {
			fmt.Println("error during sending")
			//return
			continue
		}

		connection := impl.retrieveConnection(addr)
		go connection.handlePacket(packet)
	}
}

func (impl *protocolImpl) retrieveConnection(addr *net.UDPAddr) *Connection {
	port := cnvUint32(uint32(addr.Port))
	hash := crc16.Crc16(append(addr.IP, port...))
	connection, exists := impl.connections[hash]

	if !exists {
		fmt.Println("connection from:", addr)

		// TODO pool?
		connection = &Connection{
			protocol:       impl,
			conn:           impl.socket,
			addr:           addr,
			localSequence:  1,
			remoteSequence: 0,
			sendMap:        make(map[sequenceNumber]*sendPacket),
			recvBuffer:     NewSequenceBuffer(SequenceBufferSize),
		}
		impl.connections[hash] = connection

		go connection.update()
	}

	return connection
}

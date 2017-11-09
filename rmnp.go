// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"fmt"
	"github.com/joaojeronimo/go-crc16"
	"time"
)

type ReadFunc func(*net.UDPConn, []byte) (int, *net.UDPAddr, bool)
type WriteFunc func(*Connection, []byte)

type protocolImpl struct {
	address *net.UDPAddr
	socket  *net.UDPConn

	connections map[uint16]*Connection
	readFunc    ReadFunc
	writeFunc   WriteFunc
}

func (impl *protocolImpl) init(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	checkError("Failed to resolve udp address", err)
	impl.address = addr

	impl.connections = make(map[uint16]*Connection)
}

func (impl *protocolImpl) listen() {
	for {
		// TODO pool?
		buffer := make([]byte, MTU)

		length, addr, next := impl.readFunc(impl.socket, buffer)

		if !next {
			continue
		}

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
			protocol:     impl,
			conn:         impl.socket,
			addr:         addr,
			orderedChain: NewPacketChain(),
			sendMap:      make(map[sequenceNumber]*sendPacket),
			recvBuffer:   NewSequenceBuffer(SequenceBufferSize),
		}
		impl.connections[hash] = connection

		go connection.update()

		if addr.Port != 10001 && false {
			go func() {
				for {
					connection.sendPacket(&Packet{descriptor: Reliable | Ordered}, false)
					time.Sleep(100 * time.Millisecond)
				}
			}()
		}
	}

	return connection
}

// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"fmt"
	"context"
	"time"
)

type ReadFunc func(*net.UDPConn, []byte) (int, *net.UDPAddr, bool)
type WriteFunc func(*Connection, []byte)

type protocolImpl struct {
	address *net.UDPAddr
	socket  *net.UDPConn

	ctx    context.Context
	cancel context.CancelFunc

	// TODO thread-safe?
	connections map[uint16]*Connection
	readFunc    ReadFunc
	writeFunc   WriteFunc

	// callbacks
	// for clients: only executed if client is still connected. if client disconnects callback will not be executed
	onConnect    ConnectionCallbacks
	onDisconnect ConnectionCallbacks
}

func (impl *protocolImpl) init(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	checkError("Failed to resolve udp address", err)
	impl.address = addr
	impl.connections = make(map[uint16]*Connection)
}

func (impl *protocolImpl) destroy() {
	for _, conn := range impl.connections {
		impl.disconnectClient(conn)
	}

	impl.cancel()
	impl.ctx = nil
	impl.cancel = nil

	impl.socket.Close()
	impl.address = nil
	impl.connections = nil
	impl.readFunc = nil
	impl.writeFunc = nil
}

func (impl *protocolImpl) listen() {
	impl.ctx, impl.cancel = context.WithCancel(context.Background())

	go func(ctx context.Context) {
		for {
			// TODO pool?
			buffer := make([]byte, MTU)

			// TODO execute on multiple go-routines
			impl.socket.SetDeadline(time.Now().Add(time.Second))
			length, addr, next := impl.readFunc(impl.socket, buffer)

			select {
			case <-ctx.Done():
				return
			default:
			}

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

			impl.handlePacket(addr, packet)
		}
	}(impl.ctx)
}

func (impl *protocolImpl) handlePacket(addr *net.UDPAddr, packet []byte) {
	hash := addrHash(addr)
	connection, exists := impl.connections[hash]

	if !exists {
		// check if connection packet
		if descriptor(packet[5])&Connect == 0 {
			fmt.Println("no connect packet send")
			return
		}

		connection = impl.connectClient(addr)
	}

	if descriptor(packet[5])&Connect != 0 {
		invokeConnectionCallbacks(impl.onConnect, connection)
	}

	if descriptor(packet[5])&Disconnect != 0 {
		invokeConnectionCallbacks(impl.onDisconnect, connection)
		impl.disconnectClient(connection)
		return
	}

	go connection.handlePacket(packet)
}

func (impl *protocolImpl) connectClient(addr *net.UDPAddr) *Connection {
	hash := addrHash(addr)
	fmt.Println("connection from:", addr)

	// TODO pool?
	connection := &Connection{
		protocol:     impl,
		conn:         impl.socket,
		addr:         addr,
		orderedChain: NewPacketChain(),
		sendMap:      make(map[sequenceNumber]*sendPacket),
		recvBuffer:   NewSequenceBuffer(SequenceBufferSize),
	}
	connection.ctx, connection.stopRoutines = context.WithCancel(context.Background())
	impl.connections[hash] = connection

	connection.sendLowLevelPacket(Reliable | Connect)

	go connection.update(connection.ctx)
	return connection
}

func (impl *protocolImpl) disconnectClient(connection *Connection) {
	fmt.Println("disconnection of:", connection.addr)

	connection.sendLowLevelPacket(Reliable | Disconnect)
	connection.stopRoutines()

	delete(impl.connections, addrHash(connection.addr))
}

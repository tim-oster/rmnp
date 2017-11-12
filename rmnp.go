// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"net"
	"fmt"
	"context"
	"time"
	"sync"
)

type ReadFunc func(*net.UDPConn, []byte) (int, *net.UDPAddr, bool)
type WriteFunc func(*Connection, []byte)

type protocolImpl struct {
	address *net.UDPAddr
	socket  *net.UDPConn

	ctx       context.Context
	cancel    context.CancelFunc
	waitGroup sync.WaitGroup

	// TODO thread-safe?
	connections map[uint16]*Connection
	readFunc    ReadFunc
	writeFunc   WriteFunc

	// callbacks
	// for clients: only executed if client is still connected. if client disconnects callback will not be executed
	onConnect    ConnectionCallbacks
	onDisconnect ConnectionCallbacks
	onTimeout    ConnectionCallbacks
}

func (impl *protocolImpl) init(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	checkError("Failed to resolve udp address", err)
	impl.address = addr
	impl.connections = make(map[uint16]*Connection)
}

// is blocking call!
func (impl *protocolImpl) destroy() {
	for _, conn := range impl.connections {
		impl.disconnectClient(conn, true)
	}

	impl.cancel()
	impl.waitGroup.Wait()
	impl.socket.Close()
	impl.ctx = nil
	impl.cancel = nil

	impl.address = nil
	impl.connections = nil
	impl.readFunc = nil
	impl.writeFunc = nil

	impl.onConnect = nil
	impl.onDisconnect = nil
	impl.onTimeout = nil
}

func (impl *protocolImpl) listen() {
	impl.ctx, impl.cancel = context.WithCancel(context.Background())

	go func(ctx context.Context) {
		for {
			// TODO pool?
			buffer := make([]byte, MTU)

			// TODO execute on multiple go-routines
			impl.waitGroup.Add(1)
			impl.socket.SetDeadline(time.Now().Add(time.Second))
			length, addr, next := impl.readFunc(impl.socket, buffer)
			impl.waitGroup.Done()

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
		invokeConnectionCallbacks(impl.onConnect, connection)
	}

	if descriptor(packet[5])&Disconnect != 0 {
		impl.disconnectClient(connection, false)
		return
	}

	go func() {
		impl.waitGroup.Add(1)
		defer impl.waitGroup.Done()
		connection.handlePacket(packet)
	}()
}

func (impl *protocolImpl) connectClient(addr *net.UDPAddr) *Connection {
	hash := addrHash(addr)

	// TODO pool?
	connection := newConnection(impl, addr)
	impl.connections[hash] = connection

	connection.state = Connected
	connection.sendLowLevelPacket(Reliable | Connect)
	connection.startRoutines()

	return connection
}

func (impl *protocolImpl) disconnectClient(connection *Connection, shutdown bool) {
	if connection.state == Disconnected {
		return
	}

	connection.state = Disconnected
	connection.sendLowLevelPacket(Reliable | Disconnect)
	connection.stopRoutines()

	delete(impl.connections, addrHash(connection.addr))

	if !shutdown {
		invokeConnectionCallbacks(impl.onDisconnect, connection)
	}

	connection.destroy()
}

func (impl *protocolImpl) timeoutClient(connection *Connection) {
	invokeConnectionCallbacks(impl.onTimeout, connection)
	impl.disconnectClient(connection, false)
}

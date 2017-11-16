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

type ConnectionCallback func(*Connection)
type PacketCallback func(*Connection, *net.UDPAddr, []byte) bool

func invokeConnectionCallback(callback ConnectionCallback, connection *Connection) {
	if callback != nil {
		callback(connection)
	}
}

func invokePacketCallback(callback PacketCallback, connection *Connection, addr *net.UDPAddr, packet []byte) bool {
	if callback != nil {
		return callback(connection, addr, packet)
	}

	return true
}

type ReadFunc func(*net.UDPConn, []byte) (int, *net.UDPAddr, bool)
type WriteFunc func(*Connection, []byte)

type protocolImpl struct {
	address *net.UDPAddr
	socket  *net.UDPConn

	ctx       context.Context
	cancel    context.CancelFunc
	waitGroup sync.WaitGroup

	connectionsMutex sync.RWMutex
	connections      map[uint16]*Connection
	readFunc         ReadFunc
	writeFunc        WriteFunc

	packetPool     sync.Pool
	connectionPool sync.Pool

	// callbacks
	// for clients: only executed if client is still connected. if client disconnects callback will not be executed
	onConnect    ConnectionCallback
	onDisconnect ConnectionCallback
	onTimeout    ConnectionCallback
	onValidation PacketCallback
}

func (impl *protocolImpl) init(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	checkError("Failed to resolve udp address", err)

	impl.address = addr
	impl.connections = make(map[uint16]*Connection)

	impl.packetPool = sync.Pool{
		New: func() interface{} { return make([]byte, MTU) },
	}

	impl.connectionPool = sync.Pool{
		New: func() interface{} { return newConnection() },
	}
}

// is blocking call!
func (impl *protocolImpl) destroy() {
	for _, conn := range impl.connections {
		impl.disconnectClient(conn, true)
	}

	impl.cancel()
	impl.waitGroup.Wait()
	impl.socket.Close()

	impl.address = nil
	impl.socket = nil
	impl.ctx = nil
	impl.cancel = nil

	impl.connections = nil
	impl.readFunc = nil
	impl.writeFunc = nil

	impl.onConnect = nil
	impl.onDisconnect = nil
	impl.onTimeout = nil
	impl.onValidation = nil
}

func (impl *protocolImpl) setSocket(socket *net.UDPConn, err error) {
	checkError("Error creating socket", err)
	impl.socket = socket
	impl.socket.SetReadBuffer(MTU)
	impl.socket.SetWriteBuffer(MTU)
}

func (impl *protocolImpl) listen() {
	impl.ctx, impl.cancel = context.WithCancel(context.Background())

	go func(ctx context.Context) {
		for {
			buffer := impl.packetPool.Get().([]byte)

			impl.waitGroup.Add(1)
			impl.socket.SetDeadline(time.Now().Add(time.Second))
			length, addr, next := impl.readFunc(impl.socket, buffer)
			impl.waitGroup.Done()

			select {
			case <-ctx.Done():
				impl.packetPool.Put(buffer)
				return
			default:
			}

			if !next {
				continue
			}

			go func(addr *net.UDPAddr, buffer []byte, length int) {
				impl.waitGroup.Add(1)
				defer impl.waitGroup.Done()

				defer impl.packetPool.Put(buffer)
				packet := buffer[:length]

				if !validateHeader(packet) {
					fmt.Println("error during sending")
					return
				}

				impl.handlePacket(addr, packet)
			}(addr, buffer, length)
		}
	}(impl.ctx)
}

func (impl *protocolImpl) handlePacket(addr *net.UDPAddr, packet []byte) {
	hash := addrHash(addr)

	impl.connectionsMutex.RLock()
	connection, exists := impl.connections[hash]
	impl.connectionsMutex.RUnlock()

	if !exists {
		if descriptor(packet[5])&Connect == 0 {
			fmt.Println("no connect data data")
			return
		}

		header := headerSize(packet)
		if !invokePacketCallback(impl.onValidation, nil, addr, packet[header:]) {
			fmt.Println("connection rejected")
			return
		}

		connection = impl.connectClient(addr)
	}

	// done this way to ensure that connect callback is executed on client-side
	if connection.state == Disconnected && descriptor(packet[5])&Connect != 0 {
		invokeConnectionCallback(impl.onConnect, connection)
		connection.state = Connected
	}

	if descriptor(packet[5])&Disconnect != 0 {
		impl.disconnectClient(connection, false)
		return
	}

	connection.handlePacket(packet)
}

func (impl *protocolImpl) connectClient(addr *net.UDPAddr) *Connection {
	hash := addrHash(addr)

	connection := impl.connectionPool.Get().(*Connection)
	connection.init(impl, addr)

	impl.connectionsMutex.Lock()
	impl.connections[hash] = connection
	impl.connectionsMutex.Unlock()

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

	hash := addrHash(connection.addr)

	impl.connectionsMutex.Lock()
	delete(impl.connections, hash)
	impl.connectionsMutex.Unlock()

	if !shutdown {
		invokeConnectionCallback(impl.onDisconnect, connection)
	}

	connection.reset()
	impl.connectionPool.Put(connection)
}

func (impl *protocolImpl) timeoutClient(connection *Connection) {
	invokeConnectionCallback(impl.onTimeout, connection)
	impl.disconnectClient(connection, false)
}

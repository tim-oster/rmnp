// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionCallback is the function called when connections change
type ConnectionCallback func(*Connection, []byte)

// ValidationCallback is the function called to validate a connnection
type ValidationCallback func(*net.UDPAddr, []byte) bool

// PacketCallback is the function called when a packet is received
type PacketCallback func(*Connection, []byte, Channel)

func invokeConnectionCallback(callback ConnectionCallback, connection *Connection, packet []byte) {
	if callback != nil {
		callback(connection, packet)
	}
}

func invokeValidationCallback(callback ValidationCallback, addr *net.UDPAddr, packet []byte) bool {
	if callback != nil {
		return callback(addr, packet)
	}

	return true
}

func invokePacketCallback(callback PacketCallback, connection *Connection, packet []byte, channel Channel) {
	if callback != nil {
		callback(connection, packet, channel)
	}
}

// ReadFunc is the function called to write information to a udp connection
type ReadFunc func(*net.UDPConn, []byte) (int, *net.UDPAddr, bool)

// WriteFunc is the function called to read information from a udp connection
type WriteFunc func(*net.UDPConn, *net.UDPAddr, []byte)

type disconnectType byte

const (
	disconnectTypeDefault  disconnectType = iota
	disconnectTypeShutdown
	disconnectTypeTimeout
)

type protocolImpl struct {
	address *net.UDPAddr
	socket  *net.UDPConn

	ctx       context.Context
	cancel    context.CancelFunc
	waitGroup sync.WaitGroup

	connectGuard     *execGuard
	connectionsMutex sync.RWMutex
	connections      map[uint32]*Connection
	readFunc         ReadFunc
	writeFunc        WriteFunc

	bufferPool     sync.Pool
	connectionPool sync.Pool

	// callbacks
	// for clients: only executed if client is still connected. if client disconnects callback will not be executed.
	onConnect    ConnectionCallback
	onDisconnect ConnectionCallback
	onTimeout    ConnectionCallback
	onValidation ValidationCallback
	onPacket     PacketCallback
}

func (impl *protocolImpl) init(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	checkError("Failed to resolve udp address", err)

	impl.address = addr
	impl.connectGuard = newExecGuard()
	impl.connections = make(map[uint32]*Connection)

	impl.bufferPool = sync.Pool{
		New: func() interface{} { return make([]byte, CfgMTU) },
	}

	impl.connectionPool = sync.Pool{
		New: func() interface{} { return newConnection() },
	}
}

// is blocking call!
func (impl *protocolImpl) destroy() {
	if impl.address == nil {
		return
	}

	impl.connectionsMutex.Lock()
	for _, conn := range impl.connections {
		impl.disconnectClient(conn, disconnectTypeShutdown, nil)
	}
	impl.connectionsMutex.Unlock()

	impl.cancel()
	impl.waitGroup.Wait()
	impl.socket.Close()

	impl.address = nil
	impl.socket = nil
	impl.ctx = nil
	impl.cancel = nil

	impl.connectGuard = nil
	impl.connections = nil
}

func (impl *protocolImpl) setSocket(socket *net.UDPConn, err error) {
	checkError("Error creating socket", err)
	impl.socket = socket
	impl.socket.SetReadBuffer(CfgMTU)
	impl.socket.SetWriteBuffer(CfgMTU)
}

func (impl *protocolImpl) listen() {
	impl.ctx, impl.cancel = context.WithCancel(context.Background())

	for i := 0; i < CfgParallelListenerCount; i++ {
		go impl.listeningWorker()
	}
}

func (impl *protocolImpl) listeningWorker() {
	defer antiPanic(impl.listeningWorker)

	impl.waitGroup.Add(1)
	defer impl.waitGroup.Done()

	atomic.AddUint64(&StatRunningGoRoutines, 1)
	defer atomic.AddUint64(&StatRunningGoRoutines, ^uint64(0))

	for {
		select {
		case <-impl.ctx.Done():
			return
		default:
		}

		func() {
			defer antiPanic(nil)

			buffer := impl.bufferPool.Get().([]byte)
			defer impl.bufferPool.Put(buffer)

			impl.socket.SetDeadline(time.Now().Add(time.Second))
			length, addr, next := impl.readFunc(impl.socket, buffer)

			if !next {
				return
			}

			sizedBuffer := buffer[:length]

			if !validateHeader(sizedBuffer) {
				return
			}

			packet := make([]byte, length)
			copy(packet, sizedBuffer)
			atomic.AddUint64(&StatReceivedBytes, uint64(length))

			impl.handlePacket(addr, packet)
		}()
	}
}

func (impl *protocolImpl) handlePacket(addr *net.UDPAddr, packet []byte) {
	hash := addrHash(addr)

	impl.connectionsMutex.RLock()
	connection, exists := impl.connections[hash]
	impl.connectionsMutex.RUnlock()

	if !exists {
		if descriptor(packet[5])&descConnect == 0 {
			return
		}

		if !impl.connectGuard.tryExecute(hash) {
			return
		}

		header := headerSize(packet)
		if !invokeValidationCallback(impl.onValidation, addr, packet[header:]) {
			atomic.AddUint64(&StatDeniedConnects, 1)
			return
		}

		connection = impl.connectClient(addr, nil)
	}

	// done this way to ensure that connect callback is executed on client-side
	if descriptor(packet[5])&descConnect != 0 {
		if connection.updateState(stateConnected) {
			header := headerSize(packet)

			// clear buffers so all remaining connection packets are deleted
			if connection.IsServer {
				connection.sendBuffer.reset()
				connection.sendQueue.clear()
			}

			invokeConnectionCallback(impl.onConnect, connection, packet[header:])
			impl.connectGuard.finish(hash)
		}

		return
	}

	if descriptor(packet[5])&descDisconnect != 0 {
		header := headerSize(packet)
		impl.disconnectClient(connection, disconnectTypeDefault, packet[header:])
		return
	}

	atomic.AddUint64(&StatProcessedBytes, uint64(len(packet)))
	connection.receiveQueue.push(packet)
}

func (impl *protocolImpl) connectClient(addr *net.UDPAddr, data []byte) *Connection {
	atomic.AddUint64(&StatConnects, 1)

	hash := addrHash(addr)

	connection := impl.connectionPool.Get().(*Connection)
	connection.init(impl, addr)

	impl.connectionsMutex.Lock()
	impl.connections[hash] = connection
	impl.connectionsMutex.Unlock()

	if data != nil {
		connection.sendHighLevelPacket(descReliable|descConnect, data)
	} else {
		connection.sendLowLevelPacket(descReliable | descConnect)
	}

	connection.startRoutines()

	return connection
}

func (impl *protocolImpl) disconnectClient(connection *Connection, disconnectType disconnectType, packet []byte) {
	if !connection.updateState(stateDisconnected) {
		return
	}

	if disconnectType == disconnectTypeTimeout {
		atomic.AddUint64(&StatTimeouts, 1)
		invokeConnectionCallback(impl.onTimeout, connection, nil)
	}

	atomic.AddUint64(&StatDisconnects, 1)

	// send more than necessary so that the packet hopefully arrives
	for i := 0; i < 10; i++ {
		connection.sendHighLevelPacket(descDisconnect, packet)
	}

	// give the channel some time to process the packets
	time.Sleep(20 * time.Millisecond)

	connection.stopRoutines()
	connection.waitGroup.Wait()

	if disconnectType != disconnectTypeShutdown {
		if connection.Addr != nil {
			hash := addrHash(connection.Addr)

			impl.connectionsMutex.Lock()
			delete(impl.connections, hash)
			impl.connectionsMutex.Unlock()
		}

		invokeConnectionCallback(impl.onDisconnect, connection, packet)
	}

	connection.reset()
	impl.connectionPool.Put(connection)
}

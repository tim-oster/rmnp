package main

import (
	"github.com/obsilp/rmnp"
	"net"
	"fmt"
)

func main() {
	server := rmnp.NewServer(":10001")

	server.ClientConnect = clientConnect
	server.ClientDisconnect = clientDisconnect
	server.ClientTimeout = clientTimeout
	server.ClientValidation = validateClient
	server.PacketHandler = handleServerPacket

	server.Start()
	fmt.Println("server started")

	select {}
}

func clientConnect(conn *rmnp.Connection, data []byte) {
	fmt.Println("client connection with:", data)

	if data[0] != 0 {
		conn.Disconnect([]byte("not allowed"))
	}
}

func clientDisconnect(conn *rmnp.Connection, data []byte) {
	fmt.Println("client disconnect")
}

func clientTimeout(conn *rmnp.Connection, data []byte) {
	fmt.Println("client timeout")
}

func validateClient(addr *net.UDPAddr, data []byte) bool {
	return len(data) == 3
}

func handleServerPacket(conn *rmnp.Connection, data []byte, channel rmnp.Channel) {
	str := string(data)
	fmt.Println("'"+str+"'", "from", conn.Addr.String(), "on channel", channel)

	if str == "ping" {
		conn.SendReliableOrdered([]byte("pong"))
		conn.Disconnect([]byte("session end"))
	}
}

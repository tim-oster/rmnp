package main

import (
	"fmt"
	"rmnp"
)

func main() {
	client := rmnp.NewClient("127.0.0.1:10001")

	client.ServerConnect = serverConnect
	client.ServerDisconnect = serverDisconnect
	client.ServerTimeout = serverTimeout
	client.PacketHandler = handleClientPacket

	client.ConnectWithData([]byte{0, 1, 2})

	select {}
}

func serverConnect(conn *rmnp.Connection, data []byte) {
	fmt.Println("connected to server")
	conn.SendReliableOrdered([]byte("ping"))
}

func serverDisconnect(conn *rmnp.Connection, data []byte) {
	fmt.Println("disconnected from server:", string(data))
}

func serverTimeout(conn *rmnp.Connection, data []byte) {
	fmt.Println("server timeout")
}

func handleClientPacket(conn *rmnp.Connection, data []byte, channel rmnp.Channel) {
	fmt.Println("'"+string(data)+"'", "on channel", channel)
}

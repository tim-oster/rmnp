package main

import (
	"github.com/obsilp/rmnp"
	"fmt"
	"time"
)

func main() {
	fmt.Println("starting client")
	client := rmnp.NewClient("127.0.0.1:10001")
	client.Connect()

	time.Sleep(2 * time.Second)
	client.Disconnect()

	for {
		time.Sleep(1 * time.Second)
	}
}

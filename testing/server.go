package main

import (
	"github.com/obsilp/rmnp"
	"fmt"
	"time"
)

func main() {
	fmt.Println("starting server")
	server := rmnp.NewServer(":10001")
	server.Start()

	go func() {
		time.Sleep(5 * time.Second)
		server.Stop()
	}()

	for {
		time.Sleep(1 * time.Second)
	}
}

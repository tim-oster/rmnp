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

	for {
		time.Sleep(1 * time.Second)
	}
}

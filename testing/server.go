package main

import (
	"github.com/obsilp/rmnp"
	"fmt"
)

func main() {
	fmt.Println("starting server")
	server := rmnp.NewServer(":10001")
	server.Start()
}

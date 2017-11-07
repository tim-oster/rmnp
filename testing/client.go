package main

import (
	"github.com/obsilp/rmnp"
	"fmt"
)

func main() {
	fmt.Println("starting client")
	client := rmnp.NewClient("127.0.0.1:10001")
	client.Connect()

	client.Send()

	for {
		i := 0
		_ = i
	}
}

# RMNP - Realtime Multiplayer Networking Protocol

RMNP aims to combine all the advantages of TCP and the speed of UDP in order to be fast enough to support
modern realtime games like first person shooters. It is basically an extension for UDP.

## Features

- Connections (with timeouts and ping calculation)
- Error detection
- Small overhead (max 15 bytes for header)
- Simple congestion control (avoids flooding nodes between sender/receiver)
- Optional reliable and ordered packet delivery

## How it works

The bad thing about TCP is that once a packet is dropped it stops sending all other packets until the missing one is delivered.
This can be a huge problem for games that are time sensitive because it is not uncommon for devices to encounter
packet-loss. Therefore RMNP facilitates UDP to guarantee fast delivery without any restrictions. Because UDP is stateless
RMNP implements an easy way to handle connection and to distinguish between "connected" clients. Every packet contains a small
header mainly containing a CRC32 hash to ensure that all received packets were transmitted correctly.

To guarantee reliability the receiver sends acknowledgment packets back to tell the sender which packets it received. The sender
resends each packet until it received an acknowledgment or the maximum timeout is reached. Because of that RMNP is not 100% reliable
but it can be assumed that a packet will be delivered unless a client has a packet-loss of about 100% for a couple seconds.

## Getting started

### Installation

```
go get github.com/obsilp/rmnp
```

### Basic Server

[Example Pong Server](example/server.go)

```golang
package main

import "github.com/obsilp/rmnp"

func main() {
	server := rmnp.NewServer(":10001")
	server.Start() // non-blocking

	// other code ...
}
```

### Basic Client

[Example Ping Client](example/client.go)

```golang
package main

import "github.com/obsilp/rmnp"

func main() {
	client := rmnp.NewClient("127.0.0.1:10001")
	client.Connect() // non-blocking

	// other code ...
}
```

### Callbacks

Events and received packets can be received by setting callbacks. Look at the respective classes for more
information.

[Client callbacks](client.go) | [Server callbacks](server.go)

### Send types

- **Unreliable** - Fast delivery without any guarantee on arrival or order
- **Unreliable Ordered** - Same as unreliable but only the most recent packet is accepted
- **Reliable** - Packets are guaranteed to arrive but not in order
- **Reliable Ordered** - Packets are guaranteed to arrive in order

## Ports
- [C# Version](https://github.com/obsilp/rmnp-csharp)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

## Acknowledgments

- Thanks to [@gafferongames](https://github.com/gafferongames) whose [articles](https://gafferongames.com/tags/networking)
were a huge help for this project

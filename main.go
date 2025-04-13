package main

import (
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
)

func main() {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress: ":3000",
		Decoder:       p2p.DefaultDecoder{},
		HandshakeFunc: p2p.NOPHandshakeFunc,
	}

	tcp := p2p.NewTCPTransport(tcpOpts)

	if err := tcp.ListenAndAccept(); err != nil {
		fmt.Println(err)
	}

	select {}
}

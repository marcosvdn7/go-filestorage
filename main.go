package main

import (
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
	"log"
)

func main() {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress: ":3000",
		Decoder:       p2p.DefaultDecoder{},
		HandshakeFunc: p2p.NOPHandshakeFunc,
		OnPeer:        func(p2p.Peer) error { return fmt.Errorf("failed on peer func") },
	}

	tcp := p2p.NewTCPTransport(tcpOpts)

	go func() {
		for {
			msg := <-tcp.Consume()
			fmt.Printf("Receveid msg %+v\n", msg)
		}
	}()

	if err := tcp.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}

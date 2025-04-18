package main

import (
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
	"log"
)

func onPeer(p2p.Peer) error {
	fmt.Println("Doing some logic with the peer outside of transporter")
	return nil
}

func main() {
	// Initialize the tcp transport configurations: Listen address, decoder, handshake and on peer function
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress: ":3000",
		Decoder:       p2p.DefaultDecoder{},
		HandshakeFunc: p2p.NOPHandshakeFunc,
		OnPeer:        onPeer,
	}

	tcp := p2p.NewTCPTransport(tcpOpts)

	// Create a go routine do consume the data being received in the TCPTransporter
	go func() {
		for {
			msg := <-tcp.Consume()
			fmt.Printf("Receveid msg: %s.\n", string(msg.Payload))
		}
	}()

	if err := tcp.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}

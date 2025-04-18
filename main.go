package main

import (
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
	"log"
	"time"
)

func onPeer(p2p.Peer) error {
	fmt.Println("Doing some logic with the peer outside of transporter")
	return nil
}

func main() {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress: ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO: onPeer func
	}
	tcpTransport := p2p.NewTCPTransport(tcpOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:         "3000_network",
		PathTransformerFunc: CASPathTransformerFunc,
		Transport:           tcpTransport,
	}

	s := NewFileServer(fileServerOpts)

	go func() {
		time.Sleep(time.Second * 2)
		s.Stop()
	}()

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}

//func main() {
//	// Initialize the tcp transport configurations: Listen address, decoder, handshake and on peer function
//	tcpOpts := p2p.TCPTransportOpts{
//		ListenAddress: ":3000",
//		Decoder:       p2p.DefaultDecoder{},
//		HandshakeFunc: p2p.NOPHandshakeFunc,
//		OnPeer:        onPeer,
//	}
//
//	tcp := p2p.NewTCPTransport(tcpOpts)
//
//	// Create a go routine do consume the data being received in the TCPTransporter
//	go func() {
//		for {
//			msg := <-tcp.Consume()
//			fmt.Printf("Receveid msg: %s.\n", string(msg.Payload))
//		}
//	}()
//
//	if err := tcp.ListenAndAccept(); err != nil {
//		log.Fatal(err)
//	}
//
//	select {}
//}

package main

import (
	"bytes"
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
	"log"
	"strings"
	"time"
)

func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()

	time.Sleep(1 * time.Second)

	go func() {
		err := s2.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()
	time.Sleep(1 * time.Second)

	for i := 0; i < 10; i++ {
		data := bytes.NewReader([]byte("my big data file here!"))
		if err := s2.Store(fmt.Sprintf("myprivatedata_%d", i), data); err != nil {
			log.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	select {}
}

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress: listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpOpts)

	fListenAddr, _ := strings.CutPrefix(listenAddr, ":")

	fileServerOpts := FileServerOpts{
		StorageRoot:         fListenAddr + "_network",
		PathTransformerFunc: CASPathTransformerFunc,
		Transport:           tcpTransport,
		BootstrapNodes:      nodes,
	}

	s := NewFileServer(fileServerOpts)
	tcpTransport.OnPeer = s.OnPeer

	return s
}

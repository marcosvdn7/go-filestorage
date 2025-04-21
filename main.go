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

	data := bytes.NewReader([]byte("my big data file here!"))
	fmt.Printf("Data sended to store %+v\n", data)

	time.Sleep(1 * time.Second)

	go func() {
		err := s2.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()
	time.Sleep(1 * time.Second)
	if err := s2.StoreData("myprivatedata", data); err != nil {
		fmt.Printf("Erro storing data %s\n", err.Error())
		log.Fatal(err)
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

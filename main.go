package main

import (
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
	"io"
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
	time.Sleep(2 * time.Second)

	go func() {
		log.Fatal(s2.Start())
	}()
	time.Sleep(2 * time.Second)
	//data := bytes.NewReader([]byte("my big data file here!"))
	//s2.Store("coolpicture.jpg", data)

	r, err := s2.Get("coolpicture.jpg")
	if err != nil {
		log.Fatal(err)
	}
	b, err := io.ReadAll(r)

	fmt.Println(string(b))
	//select {}
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

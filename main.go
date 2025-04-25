package main

import (
	"bytes"
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
	s3 := makeServer(":5000", ":3000", ":4000")

	go func() {
		log.Fatal(s1.Start())
	}()
	time.Sleep(2 * time.Second)

	go func() {
		log.Fatal(s2.Start())
	}()
	time.Sleep(2 * time.Second)

	go func() {
		log.Fatal(s3.Start())
	}()
	time.Sleep(2 * time.Second)

	for i := 0; i <= 20; i++ {
		key := fmt.Sprintf("picture_%d.jpg", i)
		data := bytes.NewReader([]byte("my big data file here!"))
		s2.Store(key, data)

		if err := s2.store.Delete(key); err != nil {
			log.Fatal(err)
		}

		r, err := s2.Get(key)
		if err != nil {
			log.Fatal(err)
		}
		b, err := io.ReadAll(r)

		fmt.Println(string(b))
	}
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
		EncryptionKey:       newEncryptionKey(),
		StorageRoot:         fListenAddr + "_network",
		PathTransformerFunc: CASPathTransformerFunc,
		Transport:           tcpTransport,
		BootstrapNodes:      nodes,
	}

	s := NewFileServer(fileServerOpts)
	tcpTransport.OnPeer = s.OnPeer

	return s
}

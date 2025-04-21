package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
	"io"
	"log"
	"sync"
)

type FileServerOpts struct {
	StorageRoot         string              // Root folder where the store is going to save the files
	PathTransformerFunc PathTransformerFunc // Transformer func to implement how the folders are going to be organized
	Transport           p2p.Transport
	BootstrapNodes      []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitCh chan struct{} // Empty struct channel to close the server
}

type Payload struct {
	Key  string
	Data []byte
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:                opts.StorageRoot,
		PathTransformerFunc: opts.PathTransformerFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitCh:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

// Start calls the giving transporter listen and accept function to start listening to a server
func (fs *FileServer) Start() error {
	if err := fs.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if len(fs.BootstrapNodes) != 0 {
		if err := fs.bootstrapNetwork(); err != nil {
			return err
		}
	}

	fs.loop()

	return nil
}

// Stop close the quit channel, shutting down the connection
func (fs *FileServer) Stop() {
	close(fs.quitCh)
}

func (fs *FileServer) OnPeer(p p2p.Peer) error {
	fs.peerLock.Lock()
	defer func() {
		fs.peerLock.Unlock()
		log.Printf("established connection with remote %s", p.RemoteAddr().String())
	}()

	fs.peers[p.RemoteAddr().String()] = p

	return nil
}

func (fs *FileServer) StoreData(key string, r io.Reader) error {
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	if err := fs.store.Write(key, tee); err != nil {
		return err
	}

	p := &Payload{Key: key, Data: buf.Bytes()}

	return fs.broadcast(p)
}

func (fs *FileServer) broadcast(p *Payload) error {
	var peers []io.Writer
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	encoder := gob.NewEncoder(mw)
	encoded := encoder.Encode(p)

	fmt.Println(string(p.Data))
	fmt.Println(mw)

	return encoded
}

// loop Creates the for/select responsible to handle the receiving messages
func (fs *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user action")
		fs.Transport.Close()
	}()
	for {
		select {
		case msg := <-fs.Transport.Consume():
			var p Payload
			fmt.Printf("Received msg: %q\n", msg)
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&p); err != nil {
				log.Fatal(err)
			}
		case <-fs.quitCh:
			return
		}
	}
}

// bootstrapNetwork dials and establish a connection with every node in the network
func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			if err := fs.Transport.Dial(addr); err != nil {
				log.Printf("dial error: %s\n", err)
			}
		}(addr)
	}

	return nil
}

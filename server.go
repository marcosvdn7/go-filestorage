package main

import (
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
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

// loop Creates the for/select responsible to handle the receiving messages
func (fs *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user action")
		fs.Transport.Close()
	}()
	for {
		select {
		case msg := <-fs.Transport.Consume():
			fmt.Printf("Received message: %s\n", msg)
		case <-fs.quitCh:
			return
		}
	}
}

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

package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
	"io"
	"log"
	"sync"
	"time"
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

type Message struct {
	//From    string
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
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
	size, err := fs.store.Write(key, tee)
	if err != nil {
		return err
	}
	msgBuf := new(bytes.Buffer)
	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}
	gob.Register(MessageStoreFile{})
	if err := gob.NewEncoder(msgBuf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range fs.peers {
		if err := peer.Send(msgBuf.Bytes()); err != nil {
			return err
		}
	}

	time.Sleep(time.Second * 3)
	//payload := []byte("THIS LARGE FILE")
	for _, peer := range fs.peers {
		n, err := io.Copy(peer, buf)
		if err != nil {
			return err
		}

		fmt.Printf("%d bytes copied to peer %s\n", n, peer.RemoteAddr().String())
	}

	return nil
}

func (fs *FileServer) broadcast(msg *Message) error {
	//var peers []io.Writer
	//for _, peer := range fs.peers {
	//	peers = append(peers, peer)
	//}
	//
	//mw := io.MultiWriter(peers...)
	//gob.Register(DataMessage{})
	//return gob.NewEncoder(mw).Encode(msg)
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
		case rpc := <-fs.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				fmt.Printf("Error decoding received message %s\n", err)
			}

			if err := fs.handleMessage(rpc.From, &msg); err != nil {
				fmt.Printf("Error handling message %s\n", err)
				return
			}
		case <-fs.quitCh:
			return
		}
	}
}

func (fs *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return fs.handleMessageStoreFile(from, &v)
	}
	return nil
}

func (fs *FileServer) handleMessageStoreFile(from string, msg *MessageStoreFile) error {
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not found in the peer map", from)
	}

	if _, err := fs.store.Write(msg.Key, io.LimitReader(peer, msg.Size)); err != nil {
		return err
	}
	peer.(*p2p.TCPPeer).Wg.Done()

	return nil
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

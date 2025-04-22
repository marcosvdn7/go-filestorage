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

type MessageGetFile struct {
	Key string
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
	fs.init()
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

func (fs *FileServer) Store(key string, r io.Reader) error {
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := fs.store.Write(key, tee)
	if err != nil {
		return err
	}
	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 3)
	// TODO: (@marcosvd7) use a multi writer here
	for _, peer := range fs.peers {
		if err := peer.Send([]byte{p2p.IncomingStream}); err != nil {
			return err
		}
		n, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}

		fmt.Printf("%d bytes copied to peer %s\n", n, peer.RemoteAddr().String())
	}

	return nil
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.Has(key) {
		return fs.store.Read(key)
	}

	fmt.Printf("don have file (%s) locally, fetching from network\n", key)

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	for _, peer := range fs.peers {
		fileBuffer := new(bytes.Buffer)
		n, err := io.CopyN(fileBuffer, peer, 10)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Received %d bytes from peer (%s)\n", n, peer.RemoteAddr().String())
		fmt.Printf(fileBuffer.String())
	}

	select {}

	return nil, nil
}

func (fs *FileServer) stream(msg *Message) error {
	var peers []io.Writer
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (fs *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range fs.peers {
		if err := peer.Send([]byte{p2p.IncomingMessage}); err != nil {
			return err
		}
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

// loop Creates the for/select responsible to handle the receiving messages
func (fs *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to error or user action")
		fs.Transport.Close()
	}()
	for {
		select {
		case rpc := <-fs.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				fmt.Printf("Error decoding received message: %s\n", err)
			}

			if err := fs.handleMessage(rpc.From, &msg); err != nil {
				fmt.Printf("Error handling message: %s\n", err)
			}
		case <-fs.quitCh:
			return
		}
	}
}

func (fs *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return fs.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return fs.handleMessageGetFile(from, &v)
	}
	return nil
}

func (fs *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not found in the peer map", from)
	}

	n, err := fs.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}
	fmt.Printf("[%s] writen %d bytes to disk\n", fs.Transport.Addr(), n)

	peer.CloseStream()

	return nil
}

func (fs *FileServer) handleMessageGetFile(from string, msg *MessageGetFile) error {
	if !fs.store.Has(msg.Key) {
		return fmt.Errorf("need to serve file %s but it does not exist on disk", msg.Key)
	}
	fmt.Printf("serving file (%s) over the network\n", msg.Key)
	r, err := fs.store.Read(msg.Key)
	if err != nil {
		return err
	}

	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not found in the peer map", from)
	}

	n, err := io.Copy(peer, r)
	if err != nil {
		fmt.Printf("%d bytes written over the network to %s\n", n, from)
	}
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

func (fs *FileServer) init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}

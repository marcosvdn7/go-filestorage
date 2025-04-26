package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/marcosvdn7/go-filestorage/p2p"
	"io"
	"log"
	"sync"
	"time"
)

type FileServerOpts struct {
	ID                  string
	EncryptionKey       []byte
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
	ID   string
	Key  string
	Size int64
}

type MessageGetFile struct {
	ID  string
	Key string
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:                opts.StorageRoot,
		PathTransformerFunc: opts.PathTransformerFunc,
	}

	if len(opts.ID) == 0 {
		opts.ID, _ = generateID()
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

	size, err := fs.store.Write(fs.ID, key, tee)
	if err != nil {
		return err
	}
	msg := Message{
		Payload: MessageStoreFile{
			ID:   fs.ID,
			Key:  hashKey(key),
			Size: size + 16,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	var peers []io.Writer
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	if _, err = mw.Write([]byte{p2p.IncomingStream}); err != nil {
		return err
	}
	n, err := copyEncrypt(fs.EncryptionKey, fileBuffer, mw)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] received and writter (%d) bytes to disk\n", fs.Transport.Addr(), n)

	return nil
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.Has(fs.ID, key) {
		fmt.Printf("[%s] serving file (%s) from local disk\n", fs.Transport.Addr(), key)
		_, r, err := fs.store.Read(fs.ID, key)
		return r, err
	}

	fmt.Printf("[%s] don't have file (%s) locally, fetching from network\n", fs.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			ID:  fs.ID,
			Key: hashKey(key),
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range fs.peers {
		// First read the file size so we can limit the amount of bytes that we read from the connection
		// so it will not keep hanging
		var fileSize int64
		if err := binary.Read(peer, binary.LittleEndian, &fileSize); err != nil {
			return nil, err
		}
		n, err := fs.store.WriteDecrypt(fs.ID, key, fs.EncryptionKey, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}
		fmt.Printf("[%s] received %d bytes from peer (%s)\n", fs.Transport.Addr(), n, peer.RemoteAddr().String())

		peer.CloseStream()
	}
	_, r, err := fs.store.Read(fs.ID, key)

	return r, err
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

	n, err := fs.store.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}
	fmt.Printf("[%s] writen %d bytes to disk\n", fs.Transport.Addr(), n)

	peer.CloseStream()
	fmt.Printf("[%s] stream close, resuming read loop\n", peer.RemoteAddr().String())

	return nil
}

func (fs *FileServer) handleMessageGetFile(from string, msg *MessageGetFile) error {
	if !fs.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf("[%s] need to serve file %s but it does not exist on disk", fs.Transport.Addr(), msg.Key)
	}

	fmt.Printf("[%s] serving file (%s) over the network\n", fs.Transport.Addr(), msg.Key)

	fileSize, r, err := fs.store.Read(msg.ID, msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		defer rc.Close()
	}

	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("[%s] peer %s not found in the peer map", fs.Transport.Addr(), from)
	}

	// First send the "incomingStream" byte to the peer, and then we can send
	// the file fileSize as an int64
	if err := peer.Send([]byte{p2p.IncomingStream}); err != nil {
		return err
	}
	if err := binary.Write(peer, binary.LittleEndian, &fileSize); err != nil {
		return err
	}

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes over the network to %s\n", fs.Transport.Addr(), n, from)

	return nil
}

// bootstrapNetwork dials and establish a connection with every node in the network
func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			fmt.Printf("[%s] atempting to connect with remote %s\n", fs.Transport.Addr(), addr)
			if err := fs.Transport.Dial(addr); err != nil {
				log.Printf("[%s] dial error: %s\n", fs.Transport.Addr(), err)
			}
		}(addr)
	}

	return nil
}

func (fs *FileServer) init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}

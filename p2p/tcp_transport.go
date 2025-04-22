package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP established connection
type TCPPeer struct {
	// Underlying connection of the peer. In this case, a TCP connectoin
	net.Conn

	// If we dial and retrieve a connection => outbound == true
	// If we accept and retrieve a connection => outbound == false
	outbound bool

	wg *sync.WaitGroup
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// NewTCPPeer initialize Peer with connection and outbound
func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{conn, outbound, &sync.WaitGroup{}}
}

//// Close implements the Peer interface method Close()
//func (tp *TCPPeer) Close() error {
//	return tp.conn.Close()
//}
//
//// RemoteAddr implements the Peer interface method RemoteAddr()
//// it will return the remote address of its underlying connection
//func (tp *TCPPeer) RemoteAddr() net.Addr {
//	return tp.conn.RemoteAddr()
//}

func (tp *TCPPeer) Send(data []byte) error {
	_, err := tp.Conn.Write(data)
	return err
}

// TCPTransportOpts holds the options to initialize the transporter
type TCPTransportOpts struct {
	// Address which the transporter is going to listen from
	ListenAddress string
	// Func responsible to check if everything is fine with the connection
	HandshakeFunc HandshakeFunc
	// Responsible to decode the data we receive through the connection
	Decoder Decoder
	OnPeer  func(peer Peer) error
}

// TCPTransport contains info and functions to handle the listening
// and processing of tcp connections
type TCPTransport struct {
	TCPTransportOpts
	// Listener who will be responsible to accept the connection
	listener net.Listener
	rpcChan  chan RPC
}

// NewTCPTransport initializes the tcp transporter with the handshake function
// and the address to listen from
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcChan:          make(chan RPC, 1024),
	}
}

// Addr implements the Transport interface return the address the transport
// is accepting connections
func (t *TCPTransport) Addr() string {
	return t.ListenAddress
}

// Consume implements the Transport interface, which will return a read-only channel
// for reading the incoming messages received from another peer.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcChan
}

// Close implements the Transport interface
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial implements the Transport interface
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)

	return nil
}

// ListenAndAccept with listen to the address given on the initialization of
// the transport and start the accept loop
func (t *TCPTransport) ListenAndAccept() (err error) {
	t.listener, err = net.Listen("tcp", t.ListenAddress)
	if err != nil {
		return
	}
	go t.starAcceptLoop()

	log.Printf("TCP transport listening on port: %s\n", t.ListenAddress)

	return
}

// starAcceptLoop accept and establish connection with listener
func (t *TCPTransport) starAcceptLoop() {
	for {
		// Start the loop and accept the connection with the listener
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			fmt.Printf("TCP accept loop error: %s\n", err)
		}

		go t.handleConn(conn, false)
	}
}

// handleConn defer the closing of the connection, create a new peer, calls the
// handshake to see if connection is ok and calls the onPeer function.
func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error
	defer func() {
		fmt.Printf("Dropping peer connection: %+v\n", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, outbound)

	// Does a handshake with the peer to check if everything is ok with the connection
	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// Create the struct containing the decoded payload and the sender address
	for {
		rpc := RPC{}
		// Decode de data received from the connection
		if err := t.Decoder.Decode(conn, &rpc); err != nil {
			return
		}
		// Takes the remote address from the sender
		rpc.From = conn.RemoteAddr().String()

		if rpc.Stream {
			peer.wg.Add(1)
			fmt.Printf("Incoming stream from %s. Wating...\n", rpc.From)
			peer.wg.Wait()
			fmt.Printf("Stream from %s closed. Resuming read loop\n", rpc.From)
			continue
		}

		// Put the data into the channel to be consumed
		t.rpcChan <- rpc
	}
}

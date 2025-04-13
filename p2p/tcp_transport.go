package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP established connection
type TCPPeer struct {
	// Underlying connection of the peer
	conn net.Conn

	// If we dial and retrieve a connection => outbound == true
	// If we and accept and retrieve a connection => outbound == false
	outbound bool
}

// NewTCPPeer initialize peer with connection and outbound
func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{conn, outbound}
}

type TCPTransportOpts struct {
	// Address which the transporter is going to listen from
	ListenAddress string
	// Func responsible to check if everything is fine with the connection
	HandshakeFunc HandshakeFunc
	// Responsible to decode the data we receive through the connection
	Decoder Decoder
}

// TCPTransport contains info and functions to handle the listening
// and processing of tcp connections
type TCPTransport struct {
	TCPTransportOpts
	// Listener who will be responsible to accept the connection
	listener net.Listener

	mu sync.RWMutex
	// List of nodes connected to the tcp transport
	peers map[net.Addr]Peer
}

// NewTCPTransport initializes the tcp transporter with the handshake function
// and the address to listen from
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
	}
}

// ListenAndAccept with listen to the address given on the initialization of
// the transport and start the accept loop
func (t *TCPTransport) ListenAndAccept() (err error) {
	t.listener, err = net.Listen("tcp", t.ListenAddress)
	if err != nil {
		return
	}
	go t.starAcceptLoop()

	return
}

func (t *TCPTransport) starAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("TCP accept loop error: %s\n", err)
		}

		fmt.Printf("TCP new incoming connection: %s\n", conn)

		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	peer := NewTCPPeer(conn, true)

	if err := t.HandshakeFunc(peer); err != nil {
		err := conn.Close()
		if err != nil {
			fmt.Printf("TCP close conn error: %s\n", err)
			return
		}
		fmt.Printf("TCP handshake error: %s\n", err)
		return
	}

	msg := &Message{}
	for {
		if err := t.Decoder.Decode(conn, msg); err != nil {
			fmt.Printf("TCP decode error: %s\n", err)
			continue
		}

		msg.From = conn.RemoteAddr()

		fmt.Printf("TCP data: %v\n", msg)
	}
}

package p2p

import (
	"fmt"
	"net"
	"sync"
)

type TCPTransport struct {
	listenAddress string
	listener      net.Listener

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(listenAddress string) *TCPTransport {
	return &TCPTransport{
		listenAddress: listenAddress,
	}
}

func (t *TCPTransport) ListenAndAccept() (err error) {
	t.listener, err = net.Listen("tcp", t.listenAddress)
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

		if err = conn.Close(); err != nil {
			fmt.Printf("TCP conn close error: %s\n", err)
		}

	}
}

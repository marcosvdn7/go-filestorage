package p2p

import "net"

// Peer is an interface that represents the remote node
type Peer interface {
	net.Conn
	Send([]byte) error
}

// Transport is an interface that handle the communication
// between the nodes in the network. This can be any protocol
// communication (TCP, UDP, Websockets, etc...)
type Transport interface {
	Dial(addr string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}

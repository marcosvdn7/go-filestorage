package p2p

// Peer is an interface that represents the remote node
type Peer interface {
	Close() error
}

// Transport is an interface that handle the communication
// between the nodes in the network. This can be of the
// following (TCP, UDP, websockets...)
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}

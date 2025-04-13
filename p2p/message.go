package p2p

import "net"

// Message holds any data that is being transported between two
// nodes in the network
type Message struct {
	From    net.Addr
	Payload []byte
}

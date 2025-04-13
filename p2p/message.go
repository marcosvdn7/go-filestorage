package p2p

import "net"

// RPC holds any data that is being transported between two
// nodes in the network
type RPC struct {
	From    net.Addr
	Payload []byte
}

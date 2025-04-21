package p2p

// RPC holds any data that is being transported between two
// nodes in the network
type RPC struct {
	From    string
	Payload []byte
}

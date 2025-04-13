package p2p

// Message holds any data that is being transported between two
// nodes in the network
type Message struct {
	Payload []byte
}

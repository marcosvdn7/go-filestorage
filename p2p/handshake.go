package p2p

//// ErrInvalidHandshake is returned if the handshake between
//// the remote and local node could not be established
//var ErrInvalidHandshake = errors.New("invalid handshake")

type HandshakeFunc func(peer Peer) error

func NOPHandshakeFunc(peer Peer) error {
	return nil
}

package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTCPTransport(t *testing.T) {
	listenAddr := "127.0.0.1:8080"
	tpcOpts := TCPTransportOpts{
		ListenAddress: listenAddr,
		HandshakeFunc: NOPHandshakeFunc,
		Decoder:       DefaultDecoder{},
	}
	tr := NewTCPTransport(tpcOpts)

	assert.Equal(t, tr.ListenAddress, listenAddr)

	assert.Nil(t, tr.ListenAndAccept())
}

package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTCPTransport(t *testing.T) {
	listenAddr := "127.0.0.1:8080"
	tr := NewTCPTransport(listenAddr)

	assert.Equal(t, tr.listenAddress, listenAddr)
}

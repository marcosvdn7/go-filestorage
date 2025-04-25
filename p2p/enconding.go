package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(reader io.Reader, m *RPC) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(reader io.Reader, msg *RPC) error {
	return gob.NewDecoder(reader).Decode(msg)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, m *RPC) error {
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return err
	}

	// In case of a stream we are not decoding what is being sent over the network
	// We are just setting Stream true so we can handle it in the server
	stream := peekBuf[0] == IncomingStream
	if stream {
		m.Stream = true
		return nil
	}

	buff := make([]byte, 1028)
	n, err := r.Read(buff)
	if err != nil {
		return err
	}

	m.Payload = buff[:n]

	return nil
}

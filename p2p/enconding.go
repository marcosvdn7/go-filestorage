package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(reader io.Reader, m *Message) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(reader io.Reader, msg *Message) error {
	return gob.NewDecoder(reader).Decode(msg)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, m *Message) error {
	buff := make([]byte, 1028)
	n, err := r.Read(buff)
	if err != nil {
		return err
	}

	m.Payload = buff[:n]
	return nil
}

package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCASPathTransformerFunc(t *testing.T) {
	key := "maysecretkey"
	path := CASPathTransformerFunc(key)
	expectedOriginalKey := "3b2b11b7a4e96a07a1d668c44e3fd30e96a49764"
	expectedPath := "3b2b1/1b7a4/e96a0/7a1d6/68c44/e3fd3/0e96a/49764"

	assert.Equal(t, expectedPath, path.PathName)
	assert.Equal(t, expectedOriginalKey, path.OriginalKey)
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformerFunc: DefaultPathTransformFunc,
	}
	s := NewStore(opts)

	data := bytes.NewReader([]byte("some jpg file"))
	err := s.writeStream("myspecialpicture", data)

	assert.Nil(t, err)
}

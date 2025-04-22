package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestCASPathTransformerFunc(t *testing.T) {
	key := "maysecretkey"
	path := CASPathTransformerFunc(key)
	expectedFileName := "3b2b11b7a4e96a07a1d668c44e3fd30e96a49764"
	expectedPath := "3b2b1/1b7a4/e96a0/7a1d6/68c44/e3fd3/0e96a/49764"

	assert.Equal(t, expectedPath, path.PathName)
	assert.Equal(t, expectedFileName, path.FileName)
}

func TestDelete(t *testing.T) {
	s := newStore()
	defer tearDown(t, s)

	key := "somefile"
	data := []byte("some jpg file")
	_, err := s.writeStream(key, bytes.NewReader(data))
	assert.Nil(t, err)

	err = s.Delete(key)
	assert.Nil(t, err)
}

func TestStore(t *testing.T) {
	s := newStore()
	defer tearDown(t, s)

	key := "somefile"
	data := []byte("some jpg file")
	_, err := s.writeStream(key, bytes.NewReader(data))

	assert.Nil(t, err)

	ok := s.Has(key)

	assert.Nil(t, err)
	assert.True(t, ok)

	_, r, err := s.Read(key)
	assert.Nil(t, err)

	b, _ := io.ReadAll(r)
	assert.Equal(t, string(data), string(b))
}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformerFunc: CASPathTransformerFunc,
	}
	return NewStore(opts)
}

func tearDown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}

package main

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCopyEncrypt(t *testing.T) {
	var (
		payload = "Foo not Bar"
		src     = bytes.NewReader([]byte(payload))
		dst     = new(bytes.Buffer)
		key     = newEncryptionKey()
	)

	n, err := CopyEncrypt(key, src, dst)
	assert.Nil(t, err)
	assert.Zero(t, n)

	fmt.Println(len(payload))
	fmt.Println(len(dst.String()))

	out := new(bytes.Buffer)
	nw, err := copyDecrypt(key, dst, out)
	assert.Nil(t, err)
	assert.Equal(t, payload, out.String())
	assert.Equal(t, nw, 16+len(payload))

}

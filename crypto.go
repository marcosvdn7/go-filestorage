package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"log"
)

func newEncryptionKey() []byte {
	keyBuf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, keyBuf); err != nil {
		log.Fatal(err)
	}

	return keyBuf
}

func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// Read the IV from the given io.Reader which, in this case, should be the block.BlockSize() bytes
	// we read
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}
	var (
		buf          = make([]byte, 32*1024)
		stream       = cipher.NewCTR(block, iv)
		bytesWritten = block.BlockSize()
	)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			bytesWritten += nn
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return bytesWritten, nil
}

func CopyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}
	iv := make([]byte, block.BlockSize()) // 16 bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	var (
		buf          = make([]byte, 32*1024)
		stream       = cipher.NewCTR(block, iv)
		bytesWritten = block.BlockSize()
	)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			bytesWritten += nn
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return bytesWritten, nil
}

package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"os"
	"strings"
)

type PathTransformerFunc = func(key string) (path PathKey)

var DefaultPathTransformFunc = func(key string) (path PathKey) {
	return PathKey{PathName: key, OriginalKey: key}
}

func CASPathTransformerFunc(key string) (pathKey PathKey) {
	hash := sha1.Sum([]byte(key))
	stringHash := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(stringHash) / blockSize
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = stringHash[from:to]
	}

	return PathKey{PathName: strings.Join(paths, "/"), OriginalKey: stringHash}
}

type PathKey struct {
	PathName    string
	OriginalKey string
}

type StoreOpts struct {
	PathTransformerFunc PathTransformerFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	return &Store{StoreOpts: opts}
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathName := s.PathTransformerFunc(key)
	if err := os.MkdirAll(pathName.PathName, os.ModePerm); err != nil {
		return err
	}

	pathAndFileName := pathName.PathName + "/somefile"

	f, err := os.Create(pathAndFileName)
	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)

	log.Printf("%d bytes writen to %s", n, pathName)

	return nil
}

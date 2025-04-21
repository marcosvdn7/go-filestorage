package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "system-files"

type PathTransformerFunc = func(key string) (path PathKey)

var DefaultPathTransformFunc = func(key string) (path PathKey) {
	return PathKey{PathName: key, FileName: key}
}

// CASPathTransformerFunc creates a hash from the received key, encode this hash
// to string and creates the pathFile. The path is created following the logic
//
//   - blockSize defines how many characters from the given hash will the name of
//     each folder contain.
//
//   - sliceLen will define how many depths the pathFile will have (len of string hash
//     will always be 40).
//
//   - paths will be the slice to contain the name of each folder.
//
// returns a path key struct containing the new path and the original string hash
// transformed from the giving key.
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

	return PathKey{PathName: strings.Join(paths, "/"), FileName: stringHash}
}

type PathKey struct {
	PathName string
	FileName string
}

// fullPath returns the formatted path with the filename.
func (p PathKey) fullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.FileName)
}

func (p PathKey) firstPathName() string {
	paths := strings.Split(p.PathName, "/")
	if len(paths) <= 1 {
		fmt.Println("Folder not found")
		return ""
	}
	return paths[0]
}

type StoreOpts struct {
	// Root is the folder name of the root path that contains all the folders/files of the system
	Root                string
	PathTransformerFunc PathTransformerFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformerFunc == nil {
		opts.PathTransformerFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}
	return &Store{StoreOpts: opts}
}

func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

// Read returns a buffer with the data read from the received key
func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer func(f io.ReadCloser) {
		err := f.Close()
		if err != nil {
			log.Fatalf("Read error: error closing file: %s", err)
		}
	}(f)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)

	return buf, err
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformerFunc(key)

	defer fmt.Printf("%s deleted from disk", pathKey.FileName)

	firstPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.firstPathName())

	return os.RemoveAll(firstPathWithRoot)
}

// Has - verify if the path for the giving key exists
func (s *Store) Has(key string) (ok bool) {
	pathKey := s.PathTransformerFunc(key)
	pathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.fullPath())

	_, err := os.Stat(pathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

// writeStream receives the key, transforms into a pathName using the received
// path transformer function, create the folders following the transformed path
// and save the file (r Reader)
func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformerFunc(key)                              // Transform the path with the provided key and function
	pathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.PathName) // Adds the root path

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil { // Creates all the folders using the giving path
		return err
	}
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.fullPath())

	// Crate the file in the transformed path
	f, err := os.Create(fullPathWithRoot)
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatalf("Write error: error closing file: %s", err)
		}
	}(f)
	if err != nil {
		return err
	}

	// Copy the data received in r to the created file
	n, err := io.Copy(f, r)
	if err != nil {
		fmt.Println(err)
	}

	log.Printf("%d bytes writen to %s", n, fullPathWithRoot)

	return nil
}

// readStream returns the file saved on the transformed path from the receiving key
func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformerFunc(key)
	pathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.fullPath())
	return os.Open(pathWithRoot)
}

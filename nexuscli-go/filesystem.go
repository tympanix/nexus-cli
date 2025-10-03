package main

import (
	"os"
	"path/filepath"
)

// fileSystem is an interface for file system operations
type fileSystem interface {
	Open(name string) (*os.File, error)
	Stat(name string) (os.FileInfo, error)
	Walk(root string, fn filepath.WalkFunc) error
}

// osFS implements fileSystem using the real os package
type osFS struct{}

func (osFS) Open(name string) (*os.File, error) {
	return os.Open(name)
}

func (osFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (osFS) Walk(root string, fn filepath.WalkFunc) error {
	return filepath.Walk(root, fn)
}

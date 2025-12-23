package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FileSystem abstracts file system operations for testability
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	ReadDir(path string) ([]fs.DirEntry, error)
	Remove(path string) error
	Walk(root string, walkFn filepath.WalkFunc) error
}

// OSFileSystem implements FileSystem using the real OS file system
type OSFileSystem struct{}

// NewOSFileSystem creates a new OSFileSystem instance
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

func (fs *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs *OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (fs *OSFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (fs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *OSFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

func (fs *OSFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (fs *OSFileSystem) Walk(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}





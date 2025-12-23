package filesystem

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MockFileSystem is a mock implementation of FileSystem for testing
type MockFileSystem struct {
	files      map[string][]byte
	dirs       map[string]bool
	filePerms  map[string]os.FileMode
	dirPerms   map[string]os.FileMode
	mu         sync.RWMutex
	readErrors map[string]error
	writeErrors map[string]error
	statErrors map[string]error
}

// NewMockFileSystem creates a new MockFileSystem instance
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:       make(map[string][]byte),
		dirs:        make(map[string]bool),
		filePerms:   make(map[string]os.FileMode),
		dirPerms:    make(map[string]os.FileMode),
		readErrors:  make(map[string]error),
		writeErrors: make(map[string]error),
		statErrors:  make(map[string]error),
	}
}

// SetReadError sets an error to return when reading a specific file
func (m *MockFileSystem) SetReadError(path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readErrors[path] = err
}

// SetWriteError sets an error to return when writing a specific file
func (m *MockFileSystem) SetWriteError(path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeErrors[path] = err
}

// SetStatError sets an error to return when stating a specific path
func (m *MockFileSystem) SetStatError(path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statErrors[path] = err
}

// AddFile adds a file to the mock filesystem
func (m *MockFileSystem) AddFile(path string, data []byte, perm os.FileMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[path] = data
	m.filePerms[path] = perm
}

// AddDir adds a directory to the mock filesystem
func (m *MockFileSystem) AddDir(path string, perm os.FileMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirs[path] = true
	m.dirPerms[path] = perm
}

// GetFile returns the content of a file
func (m *MockFileSystem) GetFile(path string) []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.files[path]
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.readErrors[path]; ok {
		return nil, err
	}

	if data, ok := m.files[path]; ok {
		return data, nil
	}

	return nil, os.ErrNotExist
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, ok := m.writeErrors[path]; ok {
		return err
	}

	m.files[path] = data
	m.filePerms[path] = perm
	return nil
}

func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.statErrors[path]; ok {
		return nil, err
	}

	if _, ok := m.files[path]; ok {
		return &mockFileInfo{
			name: filepath.Base(path),
			size: int64(len(m.files[path])),
			mode: m.filePerms[path],
			isDir: false,
		}, nil
	}

	if _, ok := m.dirs[path]; ok {
		return &mockFileInfo{
			name: filepath.Base(path),
			size: 0,
			mode: m.dirPerms[path],
			isDir: true,
		}, nil
	}

	return nil, os.ErrNotExist
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirs[path] = true
	m.dirPerms[path] = perm
	return nil
}

func (m *MockFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entries []fs.DirEntry
	seen := make(map[string]bool)

	// Normalize path
	path = filepath.Clean(path)
	if path == "." {
		path = ""
	}

	for filePath := range m.files {
		dir := filepath.Dir(filepath.Clean(filePath))
		if dir == path {
			name := filepath.Base(filePath)
			if !seen[name] {
				entries = append(entries, &mockDirEntry{
					name:  name,
					isDir: false,
				})
				seen[name] = true
			}
		}
	}

	for dirPath := range m.dirs {
		dir := filepath.Dir(filepath.Clean(dirPath))
		if dir == path && dirPath != path {
			name := filepath.Base(dirPath)
			if !seen[name] {
				entries = append(entries, &mockDirEntry{
					name:  name,
					isDir: true,
				})
				seen[name] = true
			}
		}
	}

	return entries, nil
}

func (m *MockFileSystem) Remove(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, path)
	delete(m.dirs, path)
	delete(m.filePerms, path)
	delete(m.dirPerms, path)
	return nil
}

func (m *MockFileSystem) Walk(root string, walkFn filepath.WalkFunc) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	visited := make(map[string]bool)

	var walk func(string) error
	walk = func(path string) error {
		if visited[path] {
			return nil
		}
		visited[path] = true

		info, err := m.Stat(path)
		if err != nil {
			return walkFn(path, nil, err)
		}

		err = walkFn(path, info, nil)
		if err != nil {
			return err
		}

		if info.IsDir() {
			entries, err := m.ReadDir(path)
			if err != nil {
				return err
			}

			for _, entry := range entries {
				subPath := filepath.Join(path, entry.Name())
				if err := walk(subPath); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return walk(root)
}

// mockFileInfo implements os.FileInfo
type mockFileInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// mockDirEntry implements fs.DirEntry
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() fs.FileMode          { return 0 }
func (m *mockDirEntry) Info() (fs.FileInfo, error) { return nil, errors.New("not implemented") }


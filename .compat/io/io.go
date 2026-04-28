package io

import (
	"bytes"
	"errors"
	goio "io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Medium interface {
	Read(path string) (string, error)
	Write(path, content string) error
	WriteMode(path, content string, mode fs.FileMode) error
	EnsureDir(path string) error
	IsFile(path string) bool
	Delete(path string) error
	DeleteAll(path string) error
	Rename(oldPath, newPath string) error
	List(path string) ([]fs.DirEntry, error)
	Stat(path string) (fs.FileInfo, error)
	Open(path string) (fs.File, error)
	Create(path string) (goio.WriteCloser, error)
	Append(path string) (goio.WriteCloser, error)
	ReadStream(path string) (goio.ReadCloser, error)
	WriteStream(path string) (goio.WriteCloser, error)
	Exists(path string) bool
	IsDir(path string) bool
}

var Local Medium = localMedium{}

type localMedium struct{}

func (localMedium) Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

func (m localMedium) Write(path, content string) error {
	return m.WriteMode(path, content, 0o644)
}

func (localMedium) WriteMode(path, content string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), mode)
}

func (localMedium) EnsureDir(path string) error { return os.MkdirAll(path, 0o755) }
func (localMedium) IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
func (localMedium) Delete(path string) error    { return os.Remove(path) }
func (localMedium) DeleteAll(path string) error { return os.RemoveAll(path) }
func (localMedium) Rename(oldPath, newPath string) error {
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}
func (localMedium) List(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
func (localMedium) Stat(path string) (fs.FileInfo, error)   { return os.Stat(path) }
func (localMedium) Open(path string) (fs.File, error)       { return os.Open(path) }
func (localMedium) Create(path string) (goio.WriteCloser, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.Create(path)
}
func (localMedium) Append(path string) (goio.WriteCloser, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}
func (m localMedium) ReadStream(path string) (goio.ReadCloser, error)   { return os.Open(path) }
func (m localMedium) WriteStream(path string) (goio.WriteCloser, error) { return m.Create(path) }
func (localMedium) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
func (localMedium) IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func Copy(source Medium, sourcePath string, destination Medium, destinationPath string) error {
	content, err := source.Read(sourcePath)
	if err != nil {
		return err
	}
	mode := fs.FileMode(0o644)
	if info, err := source.Stat(sourcePath); err == nil {
		mode = info.Mode()
	}
	return destination.WriteMode(destinationPath, content, mode)
}

type MemoryMedium struct {
	files map[string]memoryFile
	dirs  map[string]bool
}

type memoryFile struct {
	content string
	mode    fs.FileMode
	modTime time.Time
}

func NewMemoryMedium() *MemoryMedium {
	return &MemoryMedium{
		files: make(map[string]memoryFile),
		dirs:  make(map[string]bool),
	}
}

func (m *MemoryMedium) normal(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	path = strings.TrimPrefix(path, "/")
	if path == "." {
		return ""
	}
	return path
}

func (m *MemoryMedium) Read(path string) (string, error) {
	file, ok := m.files[m.normal(path)]
	if !ok {
		return "", fs.ErrNotExist
	}
	return file.content, nil
}

func (m *MemoryMedium) Write(path, content string) error {
	return m.WriteMode(path, content, 0o644)
}

func (m *MemoryMedium) WriteMode(path, content string, mode fs.FileMode) error {
	path = m.normal(path)
	if path == "" {
		return fs.ErrInvalid
	}
	m.files[path] = memoryFile{content: content, mode: mode, modTime: time.Now()}
	m.ensureParents(filepath.ToSlash(filepath.Dir(path)))
	return nil
}

func (m *MemoryMedium) EnsureDir(path string) error {
	path = m.normal(path)
	if path == "" {
		return nil
	}
	m.dirs[path] = true
	m.ensureParents(filepath.ToSlash(filepath.Dir(path)))
	return nil
}

func (m *MemoryMedium) ensureParents(path string) {
	path = m.normal(path)
	for path != "" && path != "." {
		m.dirs[path] = true
		path = m.normal(filepath.ToSlash(filepath.Dir(path)))
	}
}

func (m *MemoryMedium) IsFile(path string) bool {
	_, ok := m.files[m.normal(path)]
	return ok
}

func (m *MemoryMedium) Delete(path string) error {
	path = m.normal(path)
	if _, ok := m.files[path]; ok {
		delete(m.files, path)
		return nil
	}
	if m.IsDir(path) {
		prefix := path + "/"
		for name := range m.files {
			if strings.HasPrefix(name, prefix) {
				return errors.New("directory not empty")
			}
		}
		delete(m.dirs, path)
		return nil
	}
	return fs.ErrNotExist
}

func (m *MemoryMedium) DeleteAll(path string) error {
	path = m.normal(path)
	prefix := path + "/"
	for name := range m.files {
		if name == path || strings.HasPrefix(name, prefix) {
			delete(m.files, name)
		}
	}
	for name := range m.dirs {
		if name == path || strings.HasPrefix(name, prefix) {
			delete(m.dirs, name)
		}
	}
	return nil
}

func (m *MemoryMedium) Rename(oldPath, newPath string) error {
	oldPath = m.normal(oldPath)
	newPath = m.normal(newPath)
	if file, ok := m.files[oldPath]; ok {
		delete(m.files, oldPath)
		m.files[newPath] = file
		m.ensureParents(filepath.ToSlash(filepath.Dir(newPath)))
		return nil
	}
	if !m.IsDir(oldPath) {
		return fs.ErrNotExist
	}
	oldPrefix := oldPath + "/"
	for name, file := range m.files {
		if strings.HasPrefix(name, oldPrefix) {
			delete(m.files, name)
			m.files[newPath+strings.TrimPrefix(name, oldPath)] = file
		}
	}
	for name := range m.dirs {
		if name == oldPath || strings.HasPrefix(name, oldPrefix) {
			delete(m.dirs, name)
			m.dirs[newPath+strings.TrimPrefix(name, oldPath)] = true
		}
	}
	m.ensureParents(filepath.ToSlash(filepath.Dir(newPath)))
	return nil
}

func (m *MemoryMedium) List(path string) ([]fs.DirEntry, error) {
	path = m.normal(path)
	if path != "" && !m.IsDir(path) {
		return nil, fs.ErrNotExist
	}
	prefix := ""
	if path != "" {
		prefix = path + "/"
	}
	seen := map[string]fs.DirEntry{}
	for name, file := range m.files {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := strings.TrimPrefix(name, prefix)
		part := strings.Split(rest, "/")[0]
		if part == rest {
			seen[part] = dirEntry{info: fileInfo{name: part, size: int64(len(file.content)), mode: file.mode, modTime: file.modTime}}
		} else if _, ok := seen[part]; !ok {
			seen[part] = dirEntry{info: fileInfo{name: part, mode: fs.ModeDir | 0o755, isDir: true, modTime: time.Now()}}
		}
	}
	for name := range m.dirs {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := strings.TrimPrefix(name, prefix)
		part := strings.Split(rest, "/")[0]
		if part != "" {
			seen[part] = dirEntry{info: fileInfo{name: part, mode: fs.ModeDir | 0o755, isDir: true, modTime: time.Now()}}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, seen[name])
	}
	return entries, nil
}

func (m *MemoryMedium) Stat(path string) (fs.FileInfo, error) {
	path = m.normal(path)
	if file, ok := m.files[path]; ok {
		return fileInfo{name: filepath.Base(path), size: int64(len(file.content)), mode: file.mode, modTime: file.modTime}, nil
	}
	if m.IsDir(path) {
		return fileInfo{name: filepath.Base(path), mode: fs.ModeDir | 0o755, isDir: true, modTime: time.Now()}, nil
	}
	return nil, fs.ErrNotExist
}

func (m *MemoryMedium) Open(path string) (fs.File, error) {
	info, err := m.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fs.ErrInvalid
	}
	content, _ := m.Read(path)
	return &memoryOpenFile{Reader: bytes.NewReader([]byte(content)), info: info}, nil
}

func (m *MemoryMedium) Create(path string) (goio.WriteCloser, error) {
	return &memoryWriter{close: func(content string) error { return m.Write(path, content) }}, nil
}

func (m *MemoryMedium) Append(path string) (goio.WriteCloser, error) {
	existing, _ := m.Read(path)
	return &memoryWriter{buf: bytes.NewBufferString(existing), close: func(content string) error { return m.Write(path, content) }}, nil
}

func (m *MemoryMedium) ReadStream(path string) (goio.ReadCloser, error) {
	content, err := m.Read(path)
	if err != nil {
		return nil, err
	}
	return goio.NopCloser(strings.NewReader(content)), nil
}

func (m *MemoryMedium) WriteStream(path string) (goio.WriteCloser, error) { return m.Create(path) }
func (m *MemoryMedium) Exists(path string) bool {
	path = m.normal(path)
	return m.IsFile(path) || m.IsDir(path)
}
func (m *MemoryMedium) IsDir(path string) bool {
	path = m.normal(path)
	if path == "" {
		return true
	}
	if m.dirs[path] {
		return true
	}
	prefix := path + "/"
	for name := range m.files {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (i fileInfo) Name() string       { return i.name }
func (i fileInfo) Size() int64        { return i.size }
func (i fileInfo) Mode() fs.FileMode  { return i.mode }
func (i fileInfo) ModTime() time.Time { return i.modTime }
func (i fileInfo) IsDir() bool        { return i.isDir }
func (i fileInfo) Sys() any           { return nil }

type dirEntry struct{ info fs.FileInfo }

func (e dirEntry) Name() string               { return e.info.Name() }
func (e dirEntry) IsDir() bool                { return e.info.IsDir() }
func (e dirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e dirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

type memoryOpenFile struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *memoryOpenFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memoryOpenFile) Close() error               { return nil }

type memoryWriter struct {
	buf   *bytes.Buffer
	close func(string) error
}

func (w *memoryWriter) Write(p []byte) (int, error) {
	if w.buf == nil {
		w.buf = &bytes.Buffer{}
	}
	return w.buf.Write(p)
}

func (w *memoryWriter) Close() error {
	if w.buf == nil {
		w.buf = &bytes.Buffer{}
	}
	return w.close(w.buf.String())
}

package io

import (
	goio "io"
	"io/fs"
	"sort"
	"time"

	core "dappco.re/go"
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
	data := core.ReadFile(path)
	if !data.OK {
		return "", resultError(data)
	}
	return string(data.Value.([]byte)), nil
}

func (m localMedium) Write(path, content string) error {
	return m.WriteMode(path, content, 0o644)
}

func (localMedium) WriteMode(path, content string, mode fs.FileMode) error {
	created := core.MkdirAll(core.PathDir(path), 0o755)
	if !created.OK {
		return resultError(created)
	}
	written := core.WriteFile(path, []byte(content), mode)
	if !written.OK {
		return resultError(written)
	}
	return nil
}

func (localMedium) EnsureDir(path string) error {
	created := core.MkdirAll(path, 0o755)
	if !created.OK {
		return resultError(created)
	}
	return nil
}
func (localMedium) IsFile(path string) bool {
	info := core.Stat(path)
	return info.OK && !info.Value.(fs.FileInfo).IsDir()
}
func (localMedium) Delete(path string) error {
	removed := core.Remove(path)
	if !removed.OK {
		return resultError(removed)
	}
	return nil
}
func (localMedium) DeleteAll(path string) error {
	removed := core.RemoveAll(path)
	if !removed.OK {
		return resultError(removed)
	}
	return nil
}
func (localMedium) Rename(oldPath, newPath string) error {
	created := core.MkdirAll(core.PathDir(newPath), 0o755)
	if !created.OK {
		return resultError(created)
	}
	renamed := core.Rename(oldPath, newPath)
	if !renamed.OK {
		return resultError(renamed)
	}
	return nil
}
func (localMedium) List(path string) ([]fs.DirEntry, error) {
	read := core.ReadDir(core.DirFS(path), ".")
	if !read.OK {
		return nil, resultError(read)
	}
	return read.Value.([]core.FsDirEntry), nil
}
func (localMedium) Stat(path string) (fs.FileInfo, error) {
	info := core.Stat(path)
	if !info.OK {
		return nil, resultError(info)
	}
	return info.Value.(fs.FileInfo), nil
}
func (localMedium) Open(path string) (fs.File, error) {
	file := core.Open(path)
	if !file.OK {
		return nil, resultError(file)
	}
	return file.Value.(*core.OSFile), nil
}
func (localMedium) Create(path string) (goio.WriteCloser, error) {
	created := core.MkdirAll(core.PathDir(path), 0o755)
	if !created.OK {
		return nil, resultError(created)
	}
	file := core.Create(path)
	if !file.OK {
		return nil, resultError(file)
	}
	return file.Value.(*core.OSFile), nil
}
func (localMedium) Append(path string) (goio.WriteCloser, error) {
	created := core.MkdirAll(core.PathDir(path), 0o755)
	if !created.OK {
		return nil, resultError(created)
	}
	file := core.OpenFile(path, core.O_CREATE|core.O_WRONLY|core.O_APPEND, 0o644)
	if !file.OK {
		return nil, resultError(file)
	}
	return file.Value.(*core.OSFile), nil
}
func (m localMedium) ReadStream(path string) (goio.ReadCloser, error) {
	file := core.Open(path)
	if !file.OK {
		return nil, resultError(file)
	}
	return file.Value.(*core.OSFile), nil
}
func (m localMedium) WriteStream(path string) (goio.WriteCloser, error) { return m.Create(path) }
func (localMedium) Exists(path string) bool {
	return core.Stat(path).OK
}
func (localMedium) IsDir(path string) bool {
	info := core.Stat(path)
	return info.OK && info.Value.(fs.FileInfo).IsDir()
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
	path = core.PathToSlash(core.PathJoin(path))
	path = core.TrimPrefix(path, "/")
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
	m.ensureParents(parentSlash(path))
	return nil
}

func (m *MemoryMedium) EnsureDir(path string) error {
	path = m.normal(path)
	if path == "" {
		return nil
	}
	m.dirs[path] = true
	m.ensureParents(parentSlash(path))
	return nil
}

func (m *MemoryMedium) ensureParents(path string) {
	path = m.normal(path)
	for path != "" && path != "." {
		m.dirs[path] = true
		path = m.normal(parentSlash(path))
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
			if core.HasPrefix(name, prefix) {
				return core.NewError("directory not empty")
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
		if name == path || core.HasPrefix(name, prefix) {
			delete(m.files, name)
		}
	}
	for name := range m.dirs {
		if name == path || core.HasPrefix(name, prefix) {
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
		m.ensureParents(parentSlash(newPath))
		return nil
	}
	if !m.IsDir(oldPath) {
		return fs.ErrNotExist
	}
	oldPrefix := oldPath + "/"
	for name, file := range m.files {
		if core.HasPrefix(name, oldPrefix) {
			delete(m.files, name)
			m.files[newPath+core.TrimPrefix(name, oldPath)] = file
		}
	}
	for name := range m.dirs {
		if name == oldPath || core.HasPrefix(name, oldPrefix) {
			delete(m.dirs, name)
			m.dirs[newPath+core.TrimPrefix(name, oldPath)] = true
		}
	}
	m.ensureParents(parentSlash(newPath))
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
		if !core.HasPrefix(name, prefix) {
			continue
		}
		rest := core.TrimPrefix(name, prefix)
		part := core.Split(rest, "/")[0]
		if part == rest {
			seen[part] = dirEntry{info: fileInfo{name: part, size: int64(len(file.content)), mode: file.mode, modTime: file.modTime}}
		} else if _, ok := seen[part]; !ok {
			seen[part] = dirEntry{info: fileInfo{name: part, mode: fs.ModeDir | 0o755, isDir: true, modTime: time.Now()}}
		}
	}
	for name := range m.dirs {
		if !core.HasPrefix(name, prefix) {
			continue
		}
		rest := core.TrimPrefix(name, prefix)
		part := core.Split(rest, "/")[0]
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
		return fileInfo{name: core.PathBase(path), size: int64(len(file.content)), mode: file.mode, modTime: file.modTime}, nil
	}
	if m.IsDir(path) {
		return fileInfo{name: core.PathBase(path), mode: fs.ModeDir | 0o755, isDir: true, modTime: time.Now()}, nil
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
	return &memoryOpenFile{reader: core.NewReader(content), info: info}, nil
}

func (m *MemoryMedium) Create(path string) (goio.WriteCloser, error) {
	return &memoryWriter{close: func(content string) error { return m.Write(path, content) }}, nil
}

func (m *MemoryMedium) Append(path string) (goio.WriteCloser, error) {
	existing, _ := m.Read(path)
	return &memoryWriter{buf: core.NewBufferString(existing), close: func(content string) error { return m.Write(path, content) }}, nil
}

func (m *MemoryMedium) ReadStream(path string) (goio.ReadCloser, error) {
	content, err := m.Read(path)
	if err != nil {
		return nil, err
	}
	return goio.NopCloser(core.NewReader(content)), nil
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
		if core.HasPrefix(name, prefix) {
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
	reader goio.Reader
	info   fs.FileInfo
}

func (f *memoryOpenFile) Read(p []byte) (int, error) { return f.reader.Read(p) }
func (f *memoryOpenFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memoryOpenFile) Close() error               { return nil }

type stringBuffer interface {
	Write([]byte) (int, error)
	String() string
}

type memoryWriter struct {
	buf   stringBuffer
	close func(string) error
}

func (w *memoryWriter) Write(p []byte) (int, error) {
	if w.buf == nil {
		w.buf = core.NewBuffer()
	}
	return w.buf.Write(p)
}

func (w *memoryWriter) Close() error {
	if w.buf == nil {
		w.buf = core.NewBuffer()
	}
	return w.close(w.buf.String())
}

func parentSlash(path string) string {
	return core.PathToSlash(core.PathDir(path))
}

func resultError(result core.Result) error {
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.NewError(result.Error())
}

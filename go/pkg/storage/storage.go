package storage

import (
	goio "io"
	"io/fs"
	"testing/fstest"
	"time"

	core "dappco.re/go"
)

type Medium interface {
	Read(path string) core.Result
	Write(path, content string) core.Result
	WriteMode(path, content string, mode fs.FileMode) core.Result
	EnsureDir(path string) core.Result
	IsFile(path string) bool
	Delete(path string) core.Result
	DeleteAll(path string) core.Result
	Rename(oldPath, newPath string) core.Result
	List(path string) core.Result
	Stat(path string) core.Result
	Open(path string) core.Result
	Create(path string) core.Result
	Append(path string) core.Result
	ReadStream(path string) core.Result
	WriteStream(path string) core.Result
	Exists(path string) bool
	IsDir(path string) bool
}

var Local Medium = localstore{}

type localstore struct{}

func (localstore) Read(path string) core.Result {
	data := core.ReadFile(path)
	if !data.OK {
		return data
	}
	return core.Ok(string(data.Value.([]byte)))
}

func (m localstore) Write(path, content string) core.Result {
	return m.WriteMode(path, content, 0o644)
}

func (localstore) WriteMode(path, content string, mode fs.FileMode) core.Result {
	created := core.MkdirAll(core.PathDir(path), 0o755)
	if !created.OK {
		return created
	}
	written := core.WriteFile(path, []byte(content), mode)
	if !written.OK {
		return written
	}
	return core.Ok(nil)
}

func (localstore) EnsureDir(path string) core.Result {
	created := core.MkdirAll(path, 0o755)
	if !created.OK {
		return created
	}
	return core.Ok(nil)
}

func (localstore) IsFile(path string) bool {
	info := core.Stat(path)
	return info.OK && !info.Value.(fs.FileInfo).IsDir()
}

func (localstore) Delete(path string) core.Result {
	removed := core.Remove(path)
	if !removed.OK {
		return removed
	}
	return core.Ok(nil)
}

func (localstore) DeleteAll(path string) core.Result {
	removed := core.RemoveAll(path)
	if !removed.OK {
		return removed
	}
	return core.Ok(nil)
}

func (localstore) Rename(oldPath, newPath string) core.Result {
	created := core.MkdirAll(core.PathDir(newPath), 0o755)
	if !created.OK {
		return created
	}
	renamed := core.Rename(oldPath, newPath)
	if !renamed.OK {
		return renamed
	}
	return core.Ok(nil)
}

func (localstore) List(path string) core.Result {
	read := core.ReadDir(core.DirFS(path), ".")
	if !read.OK {
		return read
	}
	return read
}

func (localstore) Stat(path string) core.Result {
	info := core.Stat(path)
	if !info.OK {
		return info
	}
	return info
}

func (localstore) Open(path string) core.Result {
	file := core.Open(path)
	if !file.OK {
		return file
	}
	return core.Ok(file.Value.(*core.OSFile))
}

func (localstore) Create(path string) core.Result {
	created := core.MkdirAll(core.PathDir(path), 0o755)
	if !created.OK {
		return created
	}
	file := core.Create(path)
	if !file.OK {
		return file
	}
	return core.Ok(file.Value.(*core.OSFile))
}

func (localstore) Append(path string) core.Result {
	created := core.MkdirAll(core.PathDir(path), 0o755)
	if !created.OK {
		return created
	}
	file := core.OpenFile(path, core.O_CREATE|core.O_WRONLY|core.O_APPEND, 0o644)
	if !file.OK {
		return file
	}
	return core.Ok(file.Value.(*core.OSFile))
}

func (localstore) ReadStream(path string) core.Result {
	file := core.Open(path)
	if !file.OK {
		return file
	}
	return core.Ok(file.Value.(*core.OSFile))
}

func (m localstore) WriteStream(path string) core.Result { return m.Create(path) }

func (localstore) Exists(path string) bool {
	return core.Stat(path).OK
}

func (localstore) IsDir(path string) bool {
	info := core.Stat(path)
	return info.OK && info.Value.(fs.FileInfo).IsDir()
}

func Copy(source Medium, sourcePath string, destination Medium, destinationPath string) core.Result {
	content := source.Read(sourcePath)
	if !content.OK {
		return content
	}
	mode := fs.FileMode(0o644)
	if info := source.Stat(sourcePath); info.OK {
		mode = info.Value.(fs.FileInfo).Mode()
	}
	return destination.WriteMode(destinationPath, content.Value.(string), mode)
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

func (m *MemoryMedium) Read(path string) core.Result {
	file, ok := m.files[m.normal(path)]
	if !ok {
		return core.Fail(fs.ErrNotExist)
	}
	return core.Ok(file.content)
}

func (m *MemoryMedium) Write(path, content string) core.Result {
	return m.WriteMode(path, content, 0o644)
}

func (m *MemoryMedium) WriteMode(path string, content string, mode fs.FileMode) core.Result {
	path = m.normal(path)
	if path == "" {
		return core.Fail(fs.ErrInvalid)
	}
	m.files[path] = memoryFile{content: content, mode: mode, modTime: time.Now()}
	m.ensureParents(parentSlash(path))
	return core.Ok(nil)
}

func (m *MemoryMedium) EnsureDir(path string) core.Result {
	path = m.normal(path)
	if path == "" {
		return core.Ok(nil)
	}
	m.dirs[path] = true
	m.ensureParents(parentSlash(path))
	return core.Ok(nil)
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

func (m *MemoryMedium) Delete(path string) core.Result {
	path = m.normal(path)
	if _, ok := m.files[path]; ok {
		delete(m.files, path)
		return core.Ok(nil)
	}
	if m.IsDir(path) {
		prefix := path + "/"
		for name := range m.files {
			if core.HasPrefix(name, prefix) {
				return core.Fail(core.NewError("directory not empty"))
			}
		}
		delete(m.dirs, path)
		return core.Ok(nil)
	}
	return core.Fail(fs.ErrNotExist)
}

func (m *MemoryMedium) DeleteAll(path string) core.Result {
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
	return core.Ok(nil)
}

func (m *MemoryMedium) Rename(oldPath, newPath string) core.Result {
	oldPath = m.normal(oldPath)
	newPath = m.normal(newPath)
	if file, ok := m.files[oldPath]; ok {
		delete(m.files, oldPath)
		m.files[newPath] = file
		m.ensureParents(parentSlash(newPath))
		return core.Ok(nil)
	}
	if !m.IsDir(oldPath) {
		return core.Fail(fs.ErrNotExist)
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
	return core.Ok(nil)
}

func (m *MemoryMedium) List(path string) core.Result {
	path = m.normal(path)
	if path != "" && !m.IsDir(path) {
		return core.Fail(fs.ErrNotExist)
	}
	entries, err := fs.ReadDir(m.mapFS(), pathOrDot(path))
	return core.ResultOf(entries, err)
}

func (m *MemoryMedium) Stat(path string) core.Result {
	path = m.normal(path)
	if file, ok := m.files[path]; ok {
		return core.Ok(fileinfo{name: core.PathBase(path), size: int64(len(file.content)), mode: file.mode, modTime: file.modTime})
	}
	if m.IsDir(path) {
		return core.Ok(fileinfo{name: core.PathBase(path), mode: fs.ModeDir | 0o755, isDir: true, modTime: time.Now()})
	}
	return core.Fail(fs.ErrNotExist)
}

func (m *MemoryMedium) Open(path string) core.Result {
	info := m.Stat(path)
	if !info.OK {
		return info
	}
	if info.Value.(fs.FileInfo).IsDir() {
		return core.Fail(fs.ErrInvalid)
	}
	file, err := m.mapFS().Open(m.normal(path))
	return core.ResultOf(file, err)
}

func (m *MemoryMedium) Create(path string) core.Result {
	return core.Fail(core.NewError("memory stream create is not supported"))
}

func (m *MemoryMedium) Append(path string) core.Result {
	return core.Fail(core.NewError("memory stream append is not supported"))
}

func (m *MemoryMedium) ReadStream(path string) core.Result {
	content := m.Read(path)
	if !content.OK {
		return content
	}
	return core.Ok(goio.NopCloser(core.NewReader(content.Value.(string))))
}

func (m *MemoryMedium) WriteStream(path string) core.Result { return m.Create(path) }

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

func (m *MemoryMedium) mapFS() fstest.MapFS {
	mapped := fstest.MapFS{}
	for name, file := range m.files {
		mapped[name] = &fstest.MapFile{
			Data:    []byte(file.content),
			Mode:    file.mode,
			ModTime: file.modTime,
		}
	}
	for name := range m.dirs {
		if _, ok := mapped[name]; !ok {
			mapped[name] = &fstest.MapFile{Mode: fs.ModeDir | 0o755, ModTime: time.Now()}
		}
	}
	return mapped
}

func pathOrDot(path string) string {
	if path == "" {
		return "."
	}
	return path
}

type fileinfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (i fileinfo) Name() string       { return i.name }
func (i fileinfo) Size() int64        { return i.size }
func (i fileinfo) Mode() fs.FileMode  { return i.mode }
func (i fileinfo) ModTime() time.Time { return i.modTime }
func (i fileinfo) IsDir() bool        { return i.isDir }
func (i fileinfo) Sys() any           { return nil }

func parentSlash(path string) string {
	return core.PathToSlash(core.PathDir(path))
}

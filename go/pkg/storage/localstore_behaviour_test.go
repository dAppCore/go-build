package storage

import (
	goio "io"
	"io/fs"

	core "dappco.re/go"
)

// Behaviour tests exercise the real filesystem-backed Local medium, which the
// existing suite left entirely uncovered (it only drove MemoryMedium). Each
// test uses t.TempDir() so nothing escapes the sandbox.

func localPath(t *core.T, name string) string {
	t.Helper()
	return core.PathJoin(t.TempDir(), name)
}

func TestStorage_Local_WriteRead_Good(t *core.T) {
	path := localPath(t, "out.txt")
	core.AssertTrue(t, Local.Write(path, "hello").OK)
	read := Local.Read(path)
	core.AssertTrue(t, read.OK)
	core.AssertEqual(t, "hello", read.Value.(string))
}

func TestStorage_Local_Read_Bad(t *core.T) {
	core.AssertFalse(t, Local.Read(localPath(t, "missing.txt")).OK)
}

func TestStorage_Local_WriteMode_Good(t *core.T) {
	path := localPath(t, "nested/deep/file.txt")
	core.AssertTrue(t, Local.WriteMode(path, "x", 0o600).OK)
	info := Local.Stat(path)
	core.AssertTrue(t, info.OK)
	core.AssertEqual(t, fs.FileMode(0o600), info.Value.(fs.FileInfo).Mode().Perm())
}

func TestStorage_Local_EnsureDir_Good(t *core.T) {
	dir := localPath(t, "made/here")
	core.AssertTrue(t, Local.EnsureDir(dir).OK)
	core.AssertTrue(t, Local.IsDir(dir))
}

func TestStorage_Local_IsFile_Good(t *core.T) {
	path := localPath(t, "f.txt")
	core.AssertTrue(t, Local.Write(path, "a").OK)
	core.AssertTrue(t, Local.IsFile(path))
}

func TestStorage_Local_IsFile_Bad(t *core.T) {
	// A directory is not a file, and a missing path is not a file.
	dir := localPath(t, "d")
	core.AssertTrue(t, Local.EnsureDir(dir).OK)
	core.AssertFalse(t, Local.IsFile(dir))
	core.AssertFalse(t, Local.IsFile(localPath(t, "nope")))
}

func TestStorage_Local_Delete_Good(t *core.T) {
	path := localPath(t, "del.txt")
	core.AssertTrue(t, Local.Write(path, "a").OK)
	core.AssertTrue(t, Local.Delete(path).OK)
	core.AssertFalse(t, Local.Exists(path))
}

func TestStorage_Local_Delete_Bad(t *core.T) {
	core.AssertFalse(t, Local.Delete(localPath(t, "missing.txt")).OK)
}

func TestStorage_Local_DeleteAll_Good(t *core.T) {
	dir := localPath(t, "tree")
	core.AssertTrue(t, Local.Write(core.PathJoin(dir, "a.txt"), "a").OK)
	core.AssertTrue(t, Local.Write(core.PathJoin(dir, "b.txt"), "b").OK)
	core.AssertTrue(t, Local.DeleteAll(dir).OK)
	core.AssertFalse(t, Local.Exists(dir))
}

func TestStorage_Local_Rename_Good(t *core.T) {
	src := localPath(t, "src.txt")
	dst := localPath(t, "moved/dst.txt")
	core.AssertTrue(t, Local.Write(src, "payload").OK)
	core.AssertTrue(t, Local.Rename(src, dst).OK)
	core.AssertFalse(t, Local.Exists(src))
	core.AssertEqual(t, "payload", Local.Read(dst).Value.(string))
}

func TestStorage_Local_Rename_Bad(t *core.T) {
	core.AssertFalse(t, Local.Rename(localPath(t, "absent.txt"), localPath(t, "dst.txt")).OK)
}

func TestStorage_Local_List_Good(t *core.T) {
	dir := localPath(t, "listme")
	core.AssertTrue(t, Local.Write(core.PathJoin(dir, "one.txt"), "1").OK)
	core.AssertTrue(t, Local.Write(core.PathJoin(dir, "two.txt"), "2").OK)
	listed := Local.List(dir)
	core.AssertTrue(t, listed.OK)
	core.AssertEqual(t, 2, len(listed.Value.([]fs.DirEntry)))
}

func TestStorage_Local_List_Bad(t *core.T) {
	core.AssertFalse(t, Local.List(localPath(t, "no-such-dir")).OK)
}

func TestStorage_Local_Stat_Bad(t *core.T) {
	core.AssertFalse(t, Local.Stat(localPath(t, "absent")).OK)
}

func TestStorage_Local_Open_Good(t *core.T) {
	path := localPath(t, "open.txt")
	core.AssertTrue(t, Local.Write(path, "data").OK)
	opened := Local.Open(path)
	core.AssertTrue(t, opened.OK)
	file := opened.Value.(*core.OSFile)
	core.AssertEqual(t, nil, file.Close())
}

func TestStorage_Local_Open_Bad(t *core.T) {
	core.AssertFalse(t, Local.Open(localPath(t, "missing.txt")).OK)
}

func TestStorage_Local_Create_Good(t *core.T) {
	path := localPath(t, "created/file.txt")
	created := Local.Create(path)
	core.AssertTrue(t, created.OK)
	file := created.Value.(*core.OSFile)
	core.AssertEqual(t, nil, file.Close())
	core.AssertTrue(t, Local.Exists(path))
}

func TestStorage_Local_Append_Good(t *core.T) {
	path := localPath(t, "appendable/log.txt")
	appended := Local.Append(path)
	core.AssertTrue(t, appended.OK)
	file := appended.Value.(*core.OSFile)
	core.AssertEqual(t, nil, file.Close())
	core.AssertTrue(t, Local.Exists(path))
}

func TestStorage_Local_ReadStream_Good(t *core.T) {
	path := localPath(t, "stream.txt")
	core.AssertTrue(t, Local.Write(path, "streamed").OK)
	stream := Local.ReadStream(path)
	core.AssertTrue(t, stream.OK)
	reader := stream.Value.(*core.OSFile)
	data, err := goio.ReadAll(reader)
	core.AssertEqual(t, nil, err)
	core.AssertEqual(t, "streamed", string(data))
	core.AssertEqual(t, nil, reader.Close())
}

func TestStorage_Local_ReadStream_Bad(t *core.T) {
	core.AssertFalse(t, Local.ReadStream(localPath(t, "missing.txt")).OK)
}

func TestStorage_Local_WriteStream_Good(t *core.T) {
	path := localPath(t, "ws/out.txt")
	stream := Local.WriteStream(path)
	core.AssertTrue(t, stream.OK)
	file := stream.Value.(*core.OSFile)
	core.AssertEqual(t, nil, file.Close())
}

func TestStorage_Local_Exists_Ugly(t *core.T) {
	// Exists is true for a written file and false for an absent path.
	path := localPath(t, "exists.txt")
	core.AssertFalse(t, Local.Exists(path))
	core.AssertTrue(t, Local.Write(path, "y").OK)
	core.AssertTrue(t, Local.Exists(path))
}

func TestStorage_Local_WriteMode_Ugly(t *core.T) {
	// A file standing where a parent directory is expected forces the MkdirAll
	// step to fail, exercising the error branch of WriteMode/Create/Append.
	blocker := localPath(t, "blocker")
	core.AssertTrue(t, Local.Write(blocker, "i am a file").OK)
	under := core.PathJoin(blocker, "child.txt")
	core.AssertFalse(t, Local.WriteMode(under, "x", 0o644).OK)
	core.AssertFalse(t, Local.Create(under).OK)
	core.AssertFalse(t, Local.Append(under).OK)
}

func TestStorage_Local_Copy_Good(t *core.T) {
	src := localPath(t, "copy-src.txt")
	dst := localPath(t, "copy-dst.txt")
	core.AssertTrue(t, Local.Write(src, "copied").OK)
	core.AssertTrue(t, Copy(Local, src, Local, dst).OK)
	core.AssertEqual(t, "copied", Local.Read(dst).Value.(string))
}

func TestStorage_FileInfo_Accessors_Good(t *core.T) {
	// MemoryMedium.Stat returns the package fileinfo; drive its Size/ModTime/Sys
	// accessors which the existing suite never read.
	mem := NewMemoryMedium()
	core.AssertTrue(t, mem.Write("a.txt", "12345").OK)
	stat := mem.Stat("a.txt")
	core.AssertTrue(t, stat.OK)
	info := stat.Value.(fs.FileInfo)
	core.AssertEqual(t, int64(5), info.Size())
	core.AssertFalse(t, info.ModTime().IsZero())
	core.AssertEqual(t, nil, info.Sys())
}

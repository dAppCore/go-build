package storage

import (
	"io/fs"

	core "dappco.re/go"
)

func TestStorage_Copy_Good(t *core.T) {
	source := NewMemoryMedium()
	core.AssertTrue(t, source.Write("in.txt", "copy").OK)
	destination := NewMemoryMedium()
	result := Copy(source, "in.txt", destination, "out.txt")
	core.AssertTrue(t, result.OK)
}

func TestStorage_Copy_Bad(t *core.T) {
	source := NewMemoryMedium()
	destination := NewMemoryMedium()
	result := Copy(source, "missing.txt", destination, "out.txt")
	core.AssertFalse(t, result.OK)
}

func TestStorage_Copy_Ugly(t *core.T) {
	source := NewMemoryMedium()
	core.AssertTrue(t, source.WriteMode("in.txt", "copy", 0o600).OK)
	destination := NewMemoryMedium()
	core.AssertTrue(t, Copy(source, "in.txt", destination, "out.txt").OK)
	core.AssertEqual(t, fs.FileMode(0o600), storageInfo(t, destination, "out.txt").Mode().Perm())
}

func TestStorage_NewMemoryMedium_Good(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Write("file.txt", "value")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, medium.Exists("file.txt"))
}

func TestStorage_NewMemoryMedium_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Read("missing.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestStorage_NewMemoryMedium_Ugly(t *core.T) {
	first := NewMemoryMedium()
	second := NewMemoryMedium()
	core.AssertTrue(t, first.Write("file.txt", "one").OK)
	core.AssertFalse(t, second.Exists("file.txt"))
}

func TestStorage_MemoryMedium_Read_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("read.txt", "value").OK)
	result := medium.Read("read.txt")
	core.AssertEqual(t, "value", result.Value.(string))
}

func TestStorage_MemoryMedium_Read_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Read("missing.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestStorage_MemoryMedium_Read_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("read.txt", "").OK)
	result := medium.Read("read.txt")
	core.AssertEqual(t, "", result.Value.(string))
}

func TestStorage_MemoryMedium_Write_Good(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Write("write.txt", "value")
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "value", storageRead(t, medium, "write.txt"))
}

func TestStorage_MemoryMedium_Write_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Write("", "value")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestStorage_MemoryMedium_Write_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("write.txt", "first").OK)
	core.AssertTrue(t, medium.Write("write.txt", "second").OK)
	core.AssertEqual(t, "second", storageRead(t, medium, "write.txt"))
}

func TestStorage_MemoryMedium_WriteMode_Good(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.WriteMode("mode.txt", "value", 0o600)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, fs.FileMode(0o600), storageInfo(t, medium, "mode.txt").Mode().Perm())
}

func TestStorage_MemoryMedium_WriteMode_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.WriteMode("", "value", 0o600)
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestStorage_MemoryMedium_WriteMode_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.WriteMode("nested/mode.txt", "value", 0o640)
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, medium.IsDir("nested"))
}

func TestStorage_MemoryMedium_EnsureDir_Good(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.EnsureDir("dir/sub")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestStorage_MemoryMedium_EnsureDir_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.EnsureDir("")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestStorage_MemoryMedium_EnsureDir_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.EnsureDir("dir").OK)
	result := medium.EnsureDir("dir")
	core.AssertTrue(t, result.OK)
}

func TestStorage_MemoryMedium_IsFile_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("file.txt", "value").OK)
	ok := medium.IsFile("file.txt")
	core.AssertTrue(t, ok)
}

func TestStorage_MemoryMedium_IsFile_Bad(t *core.T) {
	medium := NewMemoryMedium()
	ok := medium.IsFile("missing.txt")
	core.AssertFalse(t, ok)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestStorage_MemoryMedium_IsFile_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.EnsureDir("dir").OK)
	ok := medium.IsFile("dir")
	core.AssertFalse(t, ok)
}

func TestStorage_MemoryMedium_Delete_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("file.txt", "value").OK)
	result := medium.Delete("file.txt")
	core.AssertTrue(t, result.OK)
	core.AssertFalse(t, medium.Exists("file.txt"))
}

func TestStorage_MemoryMedium_Delete_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Delete("missing.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestStorage_MemoryMedium_Delete_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("dir/file.txt", "value").OK)
	result := medium.Delete("dir")
	core.AssertFalse(t, result.OK)
}

func TestStorage_MemoryMedium_DeleteAll_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("dir/file.txt", "value").OK)
	result := medium.DeleteAll("dir")
	core.AssertTrue(t, result.OK)
	core.AssertFalse(t, medium.Exists("dir/file.txt"))
}

func TestStorage_MemoryMedium_DeleteAll_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.DeleteAll("missing")
	core.AssertTrue(t, result.OK)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestStorage_MemoryMedium_DeleteAll_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("dir/nested/file.txt", "value").OK)
	core.AssertTrue(t, medium.DeleteAll("dir/nested").OK)
	core.AssertFalse(t, medium.Exists("dir/nested/file.txt"))
}

func TestStorage_MemoryMedium_Rename_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("old.txt", "value").OK)
	result := medium.Rename("old.txt", "new.txt")
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "value", storageRead(t, medium, "new.txt"))
}

func TestStorage_MemoryMedium_Rename_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Rename("missing.txt", "new.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestStorage_MemoryMedium_Rename_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("old/file.txt", "value").OK)
	core.AssertTrue(t, medium.Rename("old", "new").OK)
	core.AssertTrue(t, medium.Exists("new/file.txt"))
}

func TestStorage_MemoryMedium_List_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("dir/file.txt", "value").OK)
	result := medium.List("dir")
	core.AssertTrue(t, result.OK)
}

func TestStorage_MemoryMedium_List_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.List("missing")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.IsDir("missing"))
}

func TestStorage_MemoryMedium_List_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("file.txt", "value").OK)
	result := medium.List("")
	core.AssertTrue(t, result.OK)
}

func TestStorage_MemoryMedium_Stat_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("file.txt", "value").OK)
	result := medium.Stat("file.txt")
	core.AssertEqual(t, "file.txt", result.Value.(fs.FileInfo).Name())
}

func TestStorage_MemoryMedium_Stat_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Stat("missing.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestStorage_MemoryMedium_Stat_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.EnsureDir("dir").OK)
	result := medium.Stat("dir")
	core.AssertTrue(t, result.Value.(fs.FileInfo).IsDir())
}

func TestStorage_MemoryMedium_Open_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("file.txt", "value").OK)
	result := medium.Open("file.txt")
	core.AssertTrue(t, result.OK)
}

func TestStorage_MemoryMedium_Open_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Open("missing.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestStorage_MemoryMedium_Open_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.EnsureDir("dir").OK)
	result := medium.Open("dir")
	core.AssertFalse(t, result.OK)
}

func TestStorage_MemoryMedium_Create_Good(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Create("file.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("file.txt"))
}

func TestStorage_MemoryMedium_Create_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Create("")
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestStorage_MemoryMedium_Create_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Create("nested/file.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("nested/file.txt"))
}

func TestStorage_MemoryMedium_Append_Good(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Append("file.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("file.txt"))
}

func TestStorage_MemoryMedium_Append_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.Append("")
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestStorage_MemoryMedium_Append_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("file.txt", "value").OK)
	result := medium.Append("file.txt")
	core.AssertFalse(t, result.OK)
}

func TestStorage_MemoryMedium_ReadStream_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("file.txt", "value").OK)
	result := medium.ReadStream("file.txt")
	core.AssertEqual(t, "value", storageReadAll(t, result.Value))
}

func TestStorage_MemoryMedium_ReadStream_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.ReadStream("missing.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestStorage_MemoryMedium_ReadStream_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("file.txt", "").OK)
	result := medium.ReadStream("file.txt")
	core.AssertEqual(t, "", storageReadAll(t, result.Value))
}

func TestStorage_MemoryMedium_WriteStream_Good(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.WriteStream("file.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("file.txt"))
}

func TestStorage_MemoryMedium_WriteStream_Bad(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.WriteStream("")
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestStorage_MemoryMedium_WriteStream_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	result := medium.WriteStream("nested/file.txt")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.Exists("nested/file.txt"))
}

func TestStorage_MemoryMedium_Exists_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("file.txt", "value").OK)
	ok := medium.Exists("file.txt")
	core.AssertTrue(t, ok)
}

func TestStorage_MemoryMedium_Exists_Bad(t *core.T) {
	medium := NewMemoryMedium()
	ok := medium.Exists("missing.txt")
	core.AssertFalse(t, ok)
	core.AssertFalse(t, medium.IsFile("missing.txt"))
}

func TestStorage_MemoryMedium_Exists_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.EnsureDir("dir").OK)
	ok := medium.Exists("dir")
	core.AssertTrue(t, ok)
}

func TestStorage_MemoryMedium_IsDir_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.EnsureDir("dir").OK)
	ok := medium.IsDir("dir")
	core.AssertTrue(t, ok)
}

func TestStorage_MemoryMedium_IsDir_Bad(t *core.T) {
	medium := NewMemoryMedium()
	ok := medium.IsDir("missing")
	core.AssertFalse(t, ok)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestStorage_MemoryMedium_IsDir_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertTrue(t, medium.Write("dir/file.txt", "value").OK)
	ok := medium.IsDir("dir")
	core.AssertTrue(t, ok)
}

func storageRead(t *core.T, medium *MemoryMedium, path string) string {
	t.Helper()
	result := medium.Read(path)
	if !result.OK {
		t.Fatalf("unexpected read error: %v", result.Error())
	}
	return result.Value.(string)
}

func storageInfo(t *core.T, medium *MemoryMedium, path string) fs.FileInfo {
	t.Helper()
	result := medium.Stat(path)
	if !result.OK {
		t.Fatalf("unexpected stat error: %v", result.Error())
	}
	return result.Value.(fs.FileInfo)
}

func storageReadAll(t *core.T, stream any) string {
	t.Helper()
	result := core.ReadAll(stream)
	if !result.OK {
		t.Fatalf("unexpected stream read error: %v", result.Error())
	}
	switch value := result.Value.(type) {
	case []byte:
		return string(value)
	case string:
		return value
	default:
		t.Fatalf("unexpected stream read type: %T", result.Value)
		return ""
	}
}

package storage

func ExampleCopy() {
	source := NewMemoryMedium()
	_ = source.Write("in.txt", "value")
	destination := NewMemoryMedium()
	_ = Copy(source, "in.txt", destination, "out.txt")
}

func ExampleNewMemoryMedium() {
	medium := NewMemoryMedium()
	_ = medium.Write("file.txt", "value")
	_ = medium.Exists("file.txt")
}

func ExampleMemoryMedium_Read() {
	medium := NewMemoryMedium()
	_ = medium.Write("file.txt", "value")
	_ = medium.Read("file.txt")
}

func ExampleMemoryMedium_Write() {
	medium := NewMemoryMedium()
	_ = medium.Write("file.txt", "value")
	_ = medium.Read("file.txt")
}

func ExampleMemoryMedium_WriteMode() {
	medium := NewMemoryMedium()
	_ = medium.WriteMode("file.txt", "value", 0o600)
	_ = medium.Stat("file.txt")
}

func ExampleMemoryMedium_EnsureDir() {
	medium := NewMemoryMedium()
	_ = medium.EnsureDir("dir")
	_ = medium.IsDir("dir")
}

func ExampleMemoryMedium_IsFile() {
	medium := NewMemoryMedium()
	_ = medium.Write("file.txt", "value")
	_ = medium.IsFile("file.txt")
}

func ExampleMemoryMedium_Delete() {
	medium := NewMemoryMedium()
	_ = medium.Write("file.txt", "value")
	_ = medium.Delete("file.txt")
}

func ExampleMemoryMedium_DeleteAll() {
	medium := NewMemoryMedium()
	_ = medium.Write("dir/file.txt", "value")
	_ = medium.DeleteAll("dir")
}

func ExampleMemoryMedium_Rename() {
	medium := NewMemoryMedium()
	_ = medium.Write("old.txt", "value")
	_ = medium.Rename("old.txt", "new.txt")
}

func ExampleMemoryMedium_List() {
	medium := NewMemoryMedium()
	_ = medium.Write("dir/file.txt", "value")
	_ = medium.List("dir")
}

func ExampleMemoryMedium_Stat() {
	medium := NewMemoryMedium()
	_ = medium.Write("file.txt", "value")
	_ = medium.Stat("file.txt")
}

func ExampleMemoryMedium_Open() {
	medium := NewMemoryMedium()
	_ = medium.Write("file.txt", "value")
	_ = medium.Open("file.txt")
}

func ExampleMemoryMedium_Create() {
	medium := NewMemoryMedium()
	_ = medium.Create("file.txt")
	_ = medium.Exists("file.txt")
}

func ExampleMemoryMedium_Append() {
	medium := NewMemoryMedium()
	_ = medium.Append("file.txt")
	_ = medium.Exists("file.txt")
}

func ExampleMemoryMedium_ReadStream() {
	medium := NewMemoryMedium()
	_ = medium.Write("file.txt", "value")
	_ = medium.ReadStream("file.txt")
}

func ExampleMemoryMedium_WriteStream() {
	medium := NewMemoryMedium()
	_ = medium.WriteStream("file.txt")
	_ = medium.Exists("file.txt")
}

func ExampleMemoryMedium_Exists() {
	medium := NewMemoryMedium()
	_ = medium.Write("file.txt", "value")
	_ = medium.Exists("file.txt")
}

func ExampleMemoryMedium_IsDir() {
	medium := NewMemoryMedium()
	_ = medium.EnsureDir("dir")
	_ = medium.IsDir("dir")
}

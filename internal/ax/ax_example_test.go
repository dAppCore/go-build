package ax

import (
	"syscall"
	"time"

	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleDS() {
	_ = DS()
	core.Println("DS")
	// Output: DS
}

func ExampleClean() {
	_ = Clean(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Clean")
	// Output: Clean
}

func ExampleJoin() {
	_ = Join()
	core.Println("Join")
	// Output: Join
}

func ExampleAbs() {
	_, _ = Abs(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Abs")
	// Output: Abs
}

func ExampleRel() {
	_, _ = Rel("agent", "linux")
	core.Println("Rel")
	// Output: Rel
}

func ExampleBase() {
	_ = Base(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Base")
	// Output: Base
}

func ExampleDir() {
	_ = Dir(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Dir")
	// Output: Dir
}

func ExampleExt() {
	_ = Ext(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Ext")
	// Output: Ext
}

func ExampleIsAbs() {
	_ = IsAbs(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("IsAbs")
	// Output: IsAbs
}

func ExampleFromSlash() {
	_ = FromSlash(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("FromSlash")
	// Output: FromSlash
}

func ExampleGetwd() {
	_, _ = Getwd()
	core.Println("Getwd")
	// Output: Getwd
}

func ExampleTempDir() {
	_, _ = TempDir("agent")
	core.Println("TempDir")
	// Output: TempDir
}

func ExampleMkdirTemp() {
	_, _ = MkdirTemp("agent")
	core.Println("MkdirTemp")
	// Output: MkdirTemp
}

func ExampleReadFile() {
	_, _ = ReadFile(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("ReadFile")
	// Output: ReadFile
}

func ExampleWriteFile() {
	_ = WriteFile(core.Path(core.TempDir(), "go-build-compliance"), []byte("agent"), 0o755)
	core.Println("WriteFile")
	// Output: WriteFile
}

func ExampleWriteString() {
	_ = WriteString(core.Path(core.TempDir(), "go-build-compliance"), "agent", 0o755)
	core.Println("WriteString")
	// Output: WriteString
}

func ExampleMkdirAll() {
	_ = MkdirAll(core.Path(core.TempDir(), "go-build-compliance"), 0o755)
	core.Println("MkdirAll")
	// Output: MkdirAll
}

func ExampleMkdir() {
	_ = Mkdir(core.Path(core.TempDir(), "go-build-compliance"), 0o755)
	core.Println("Mkdir")
	// Output: Mkdir
}

func ExampleRemoveAll() {
	_ = RemoveAll(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("RemoveAll")
	// Output: RemoveAll
}

func ExampleStat() {
	_, _ = Stat(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Stat")
	// Output: Stat
}

func ExampleReadDir() {
	_, _ = ReadDir(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("ReadDir")
	// Output: ReadDir
}

func ExampleOpen() {
	_, _ = Open(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Open")
	// Output: Open
}

func ExampleCreate() {
	_, _ = Create(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Create")
	// Output: Create
}

func ExampleExists() {
	_ = Exists(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Exists")
	// Output: Exists
}

func ExampleIsFile() {
	_ = IsFile(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("IsFile")
	// Output: IsFile
}

func ExampleIsDir() {
	_ = IsDir(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("IsDir")
	// Output: IsDir
}

func ExampleChmod() {
	_ = Chmod(core.Path(core.TempDir(), "go-build-compliance"), 0o755)
	core.Println("Chmod")
	// Output: Chmod
}

func ExampleGetuid() {
	_ = Getuid()
	core.Println("Getuid")
	// Output: Getuid
}

func ExampleGetgid() {
	_ = Getgid()
	core.Println("Getgid")
	// Output: Getgid
}

func ExampleGeteuid() {
	_ = Geteuid()
	core.Println("Geteuid")
	// Output: Geteuid
}

func ExampleJSONMarshal() {
	_, _ = JSONMarshal("agent")
	core.Println("JSONMarshal")
	// Output: JSONMarshal
}

func ExampleJSONUnmarshal() {
	_ = JSONUnmarshal([]byte("agent"), "agent")
	core.Println("JSONUnmarshal")
	// Output: JSONUnmarshal
}

func ExampleLookPath() {
	_, _ = LookPath("agent")
	core.Println("LookPath")
	// Output: LookPath
}

func ExampleResolveCommand() {
	_, _ = ResolveCommand("agent")
	core.Println("ResolveCommand")
	// Output: ResolveCommand
}

func ExampleRun() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_, _ = Run(ctx, "dappcore-command-not-found")
	core.Println("Run")
	// Output: Run
}

func ExampleRunDir() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_, _ = RunDir(ctx, core.Path(core.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
	core.Println("RunDir")
	// Output: RunDir
}

func ExampleExec() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = Exec(ctx, "dappcore-command-not-found")
	core.Println("Exec")
	// Output: Exec
}

func ExampleExecDir() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = ExecDir(ctx, core.Path(core.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
	core.Println("ExecDir")
	// Output: ExecDir
}

func ExampleExecWithEnv() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = ExecWithEnv(ctx, core.Path(core.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
	core.Println("ExecWithEnv")
	// Output: ExecWithEnv
}

func ExampleExecWithWriters() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = ExecWithWriters(ctx, core.Path(core.TempDir(), "go-build-compliance"), []string{"agent"}, core.NewBuffer(), core.NewBuffer(), "dappcore-command-not-found")
	core.Println("ExecWithWriters")
	// Output: ExecWithWriters
}

func ExampleCombinedOutput() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_, _ = CombinedOutput(ctx, core.Path(core.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
	core.Println("CombinedOutput")
	// Output: CombinedOutput
}

func ExampleChtimes() {
	path := Join(core.TempDir(), "go-build-compliance-chtimes")
	_ = WriteString(path, "agent", 0o644)
	_ = Chtimes(path, time.Unix(123, 0), time.Unix(123, 0))
	core.Println("Chtimes")
	// Output: Chtimes
}

func ExampleReadlink() {
	dir := core.TempDir()
	link := Join(dir, "go-build-compliance-link")
	_ = syscall.Symlink("target", link)
	_, _ = Readlink(link)
	core.Println("Readlink")
	// Output: Readlink
}

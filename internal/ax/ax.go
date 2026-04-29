package ax

import (
	"context"
	"io"
	"io/fs"
	"runtime"
	"syscall"
	"time"

	"dappco.re/go"
	coreio "dappco.re/go/io"
	process "dappco.re/go/process"
	processexec "dappco.re/go/process/exec"
)

// DS returns the current platform directory separator.
//
// Usage example: read ax.DS() when building Core-aware filesystem paths.
func DS() string {
	if sep := core.Env("DS"); sep != "" {
		return sep
	}
	if runtime.GOOS == "windows" {
		return "\\"
	}
	return "/"
}

// Clean normalises a filesystem path using Core path primitives.
//
// Usage example: clean := ax.Clean("./dist/../dist/output")
func Clean(path string) string {
	return core.CleanPath(path, DS())
}

// Join combines path segments without relying on path/filepath.
//
// Usage example: path := ax.Join(projectDir, ".core", "build.yaml")
func Join(parts ...string) string {
	return Clean(core.Join(DS(), parts...))
}

// Abs resolves a path against the current working directory.
//
// Usage example: abs, err := ax.Abs("./testdata")
func Abs(path string) core.Result {
	if core.PathIsAbs(path) {
		return core.Ok(Clean(path))
	}

	cwd := Getwd()
	if !cwd.OK {
		return cwd
	}

	return core.Ok(Join(cwd.Value.(string), path))
}

// Rel returns target relative to base when target is inside base.
//
// Usage example: rel, err := ax.Rel(projectDir, artifactPath)
func Rel(base, target string) core.Result {
	base = Clean(base)
	target = Clean(target)

	if base == target {
		return core.Ok(".")
	}

	prefix := base
	if !core.HasSuffix(prefix, DS()) {
		prefix = core.Concat(prefix, DS())
	}

	if core.HasPrefix(target, prefix) {
		return core.Ok(core.TrimPrefix(target, prefix))
	}

	return core.Fail(core.E("ax.Rel", "path is outside base: "+target, nil))
}

// Base returns the last path element.
//
// Usage example: name := ax.Base("/tmp/dist/app.tar.gz")
func Base(path string) string {
	return core.PathBase(path)
}

// Dir returns the parent directory for a path.
//
// Usage example: dir := ax.Dir("/tmp/dist/app.tar.gz")
func Dir(path string) string {
	return core.PathDir(path)
}

// Ext returns the filename extension including the dot.
//
// Usage example: ext := ax.Ext("app.tar.gz")
func Ext(path string) string {
	return core.PathExt(path)
}

// IsAbs reports whether a path is absolute.
//
// Usage example: if ax.IsAbs(outputDir) { ... }
func IsAbs(path string) bool {
	return core.PathIsAbs(path)
}

// FromSlash rewrites slash-separated paths for the current platform.
//
// Usage example: path := ax.FromSlash("ui/dist/index.html")
func FromSlash(path string) string {
	if DS() == "/" {
		return path
	}
	return core.Replace(path, "/", DS())
}

// Getwd returns the current working directory from Core environment metadata.
//
// Usage example: cwd, err := ax.Getwd()
func Getwd() core.Result {
	cwd := core.Env("DIR_CWD")
	if cwd == "" {
		wd, err := syscall.Getwd()
		if err != nil {
			return core.Fail(core.E("ax.Getwd", "failed to get current working directory", err))
		}
		return core.Ok(wd)
	}
	return core.Ok(cwd)
}

// TempDir creates a temporary directory via Core's filesystem primitive.
//
// Usage example: dir, err := ax.TempDir("core-build-*")
func TempDir(prefix string) core.Result {
	dir := (&core.Fs{}).NewUnrestricted().TempDir(prefix)
	if dir == "" {
		return core.Fail(core.E("ax.TempDir", "failed to create temporary directory", nil))
	}
	return core.Ok(dir)
}

// MkdirTemp creates a temporary directory via Core's filesystem primitive.
//
// Usage example: dir, err := ax.MkdirTemp("core-build-*")
func MkdirTemp(prefix string) core.Result {
	return TempDir(prefix)
}

// ReadFile reads a file into bytes via io.Local.
//
// Usage example: data, err := ax.ReadFile("go.mod")
func ReadFile(path string) core.Result {
	content := coreio.Local.Read(path)
	if !content.OK {
		return core.Fail(core.E("ax.ReadFile", "failed to read file "+path, core.NewError(content.Error())))
	}
	return core.Ok([]byte(content.Value.(string)))
}

// WriteFile writes bytes via io.Local with an explicit mode.
//
// Usage example: err := ax.WriteFile("README.md", []byte("hi"), 0o644)
func WriteFile(path string, data []byte, mode fs.FileMode) core.Result {
	written := coreio.Local.WriteMode(path, string(data), mode)
	if !written.OK {
		return core.Fail(core.E("ax.WriteFile", "failed to write file "+path, core.NewError(written.Error())))
	}
	return core.Ok(nil)
}

// WriteString writes text via io.Local with an explicit mode.
//
// Usage example: err := ax.WriteString("README.md", "hi", 0o644)
func WriteString(path, data string, mode fs.FileMode) core.Result {
	written := coreio.Local.WriteMode(path, data, mode)
	if !written.OK {
		return core.Fail(core.E("ax.WriteString", "failed to write file "+path, core.NewError(written.Error())))
	}
	return core.Ok(nil)
}

// MkdirAll ensures a directory exists.
//
// Usage example: err := ax.MkdirAll("dist/linux_arm64", 0o755)
func MkdirAll(path string, _ fs.FileMode) core.Result {
	created := coreio.Local.EnsureDir(path)
	if !created.OK {
		return core.Fail(core.E("ax.MkdirAll", "failed to create directory "+path, core.NewError(created.Error())))
	}
	return core.Ok(nil)
}

// Mkdir ensures a directory exists.
//
// Usage example: err := ax.Mkdir(".core", 0o755)
func Mkdir(path string, mode fs.FileMode) core.Result {
	return MkdirAll(path, mode)
}

// RemoveAll removes a file or directory tree.
//
// Usage example: err := ax.RemoveAll("dist")
func RemoveAll(path string) core.Result {
	removed := coreio.Local.DeleteAll(path)
	if !removed.OK {
		return core.Fail(core.E("ax.RemoveAll", "failed to remove path "+path, core.NewError(removed.Error())))
	}
	return core.Ok(nil)
}

// Stat returns file metadata from io.Local.
//
// Usage example: info, err := ax.Stat("go.mod")
func Stat(path string) core.Result {
	info := coreio.Local.Stat(path)
	if !info.OK {
		return core.Fail(core.E("ax.Stat", "failed to stat path "+path, core.NewError(info.Error())))
	}
	return info
}

// ReadDir lists directory entries via io.Local.
//
// Usage example: entries, err := ax.ReadDir("dist")
func ReadDir(path string) core.Result {
	entries := coreio.Local.List(path)
	if !entries.OK {
		return core.Fail(core.E("ax.ReadDir", "failed to list directory "+path, core.NewError(entries.Error())))
	}
	return entries
}

// Open opens a file for reading via io.Local.
//
// Usage example: file, err := ax.Open("README.md")
func Open(path string) core.Result {
	file := coreio.Local.Open(path)
	if !file.OK {
		return core.Fail(core.E("ax.Open", "failed to open file "+path, core.NewError(file.Error())))
	}
	return file
}

// Create opens a file for writing via io.Local.
//
// Usage example: file, err := ax.Create("dist/output.txt")
func Create(path string) core.Result {
	file := coreio.Local.Create(path)
	if !file.OK {
		return core.Fail(core.E("ax.Create", "failed to create file "+path, core.NewError(file.Error())))
	}
	return file
}

// Exists reports whether a path exists.
//
// Usage example: if ax.Exists("dist") { ... }
func Exists(path string) bool {
	return coreio.Local.Exists(path)
}

// IsFile reports whether a path is a regular file.
//
// Usage example: if ax.IsFile("go.mod") { ... }
func IsFile(path string) bool {
	return coreio.Local.IsFile(path)
}

// IsDir reports whether a path is a directory.
//
// Usage example: if ax.IsDir(".core") { ... }
func IsDir(path string) bool {
	return coreio.Local.IsDir(path)
}

// Chmod updates file permissions without importing os.
//
// Usage example: err := ax.Chmod("dist/app", 0o755)
func Chmod(path string, mode fs.FileMode) core.Result {
	if err := syscall.Chmod(path, uint32(mode)); err != nil {
		return core.Fail(core.E("ax.Chmod", "failed to change permissions on "+path, err))
	}
	return core.Ok(nil)
}

// Chtimes updates access and modification times without importing the OS package.
//
// Usage example: err := ax.Chtimes("dist/app", modTime, modTime)
func Chtimes(path string, atime, mtime time.Time) core.Result {
	times := []syscall.Timespec{
		syscall.NsecToTimespec(atime.UnixNano()),
		syscall.NsecToTimespec(mtime.UnixNano()),
	}
	if err := syscall.UtimesNano(path, times); err != nil {
		return core.Fail(core.E("ax.Chtimes", "failed to change timestamps on "+path, err))
	}
	return core.Ok(nil)
}

// Readlink reads a symbolic link target without importing the OS package.
//
// Usage example: target, err := ax.Readlink("dist/current")
func Readlink(path string) core.Result {
	buffer := make([]byte, 4096)
	n, err := syscall.Readlink(path, buffer)
	if err != nil {
		return core.Fail(core.E("ax.Readlink", "failed to read symlink "+path, err))
	}
	return core.Ok(string(buffer[:n]))
}

// Getuid returns the current process UID.
//
// Usage example: uid := ax.Getuid()
func Getuid() int {
	return syscall.Getuid()
}

// Getgid returns the current process GID.
//
// Usage example: gid := ax.Getgid()
func Getgid() int {
	return syscall.Getgid()
}

// Geteuid returns the effective UID.
//
// Usage example: if ax.Geteuid() == 0 { ... }
func Geteuid() int {
	return syscall.Geteuid()
}

// JSONMarshal returns a JSON string using Core's JSON wrapper.
//
// Usage example: data, err := ax.JSONMarshal(cfg)
func JSONMarshal(value any) core.Result {
	result := core.JSONMarshal(value)
	if !result.OK {
		return core.Fail(core.E("ax.JSONMarshal", "failed to marshal JSON", core.NewError(result.Error())))
	}
	encoded, ok := result.Value.([]byte)
	if !ok {
		return core.Fail(core.E("ax.JSONMarshal", "failed to marshal JSON", nil))
	}
	return core.Ok(string(encoded))
}

// JSONUnmarshal decodes JSON into target using Core's JSON wrapper.
//
// Usage example: err := ax.JSONUnmarshal(data, &cfg)
func JSONUnmarshal(data []byte, target any) core.Result {
	result := core.JSONUnmarshal(data, target)
	if !result.OK {
		return core.Fail(core.E("ax.JSONUnmarshal", "failed to unmarshal JSON", core.NewError(result.Error())))
	}
	return core.Ok(nil)
}

// LookPath resolves a program on PATH via the Core process package.
//
// Usage example: path, err := ax.LookPath("git")
func LookPath(name string) core.Result {
	program := process.Program{Name: name}
	found := program.Find()
	if !found.OK {
		return core.Fail(core.E("ax.LookPath", "failed to locate command "+name, core.NewError(found.Error())))
	}
	return core.Ok(program.Path)
}

// ResolveCommand resolves a program from PATH or a list of fallback paths.
//
// Usage example: path, err := ax.ResolveCommand("task", "/opt/homebrew/bin/task")
func ResolveCommand(name string, fallbackPaths ...string) core.Result {
	path := LookPath(name)
	if path.OK {
		return path
	}

	for _, fallbackPath := range fallbackPaths {
		if IsFile(fallbackPath) {
			return core.Ok(fallbackPath)
		}
	}

	return core.Fail(core.E("ax.ResolveCommand", "failed to locate command "+name, core.NewError(path.Error())))
}

// Run executes a command and returns trimmed combined output.
//
// Usage example: output, err := ax.Run(ctx, "git", "status", "--short")
func Run(ctx context.Context, command string, args ...string) core.Result {
	program := process.Program{Name: command}
	return program.Run(ctx, args...)
}

// RunDir executes a command in the provided directory and returns combined output.
//
// Usage example: output, err := ax.RunDir(ctx, repoDir, "git", "show", "--stat")
func RunDir(ctx context.Context, dir, command string, args ...string) core.Result {
	program := process.Program{Name: command}
	return program.RunDir(ctx, dir, args...)
}

// Exec executes a command without capturing output.
//
// Usage example: err := ax.Exec(ctx, "go", "test", "./...")
func Exec(ctx context.Context, command string, args ...string) core.Result {
	return processexec.Command(ctx, command, args...).Run()
}

// ExecDir executes a command in a specific directory without capturing output.
//
// Usage example: err := ax.ExecDir(ctx, repoDir, "go", "test", "./...")
func ExecDir(ctx context.Context, dir, command string, args ...string) core.Result {
	return processexec.Command(ctx, command, args...).WithDir(dir).Run()
}

// ExecWithEnv executes a command with additional environment variables.
//
// Usage example: err := ax.ExecWithEnv(ctx, repoDir, []string{"GOOS=linux"}, "go", "build")
func ExecWithEnv(ctx context.Context, dir string, env []string, command string, args ...string) core.Result {
	return processexec.Command(ctx, command, args...).WithDir(dir).WithEnv(env).Run()
}

// ExecWithWriters executes a command and streams output to the provided writers.
//
// Usage example: err := ax.ExecWithWriters(ctx, repoDir, nil, w, w, "docker", "build", ".")
func ExecWithWriters(ctx context.Context, dir string, env []string, stdout, stderr io.Writer, command string, args ...string) core.Result {
	cmd := processexec.Command(ctx, command, args...).WithDir(dir).WithEnv(env)
	if stdout != nil {
		cmd = cmd.WithStdout(stdout)
	}
	if stderr != nil {
		cmd = cmd.WithStderr(stderr)
	}
	return cmd.Run()
}

// CombinedOutput executes a command and returns combined output.
//
// Usage example: output, err := ax.CombinedOutput(ctx, repoDir, nil, "go", "test", "./...")
func CombinedOutput(ctx context.Context, dir string, env []string, command string, args ...string) core.Result {
	cmd := processexec.Command(ctx, command, args...).WithDir(dir).WithEnv(env)
	output := cmd.CombinedOutput()
	if !output.OK {
		return output
	}
	return core.Ok(core.Trim(string(output.Value.([]byte))))
}

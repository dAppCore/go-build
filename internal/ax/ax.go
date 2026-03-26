package ax

import (
	"context"
	"io"
	"io/fs"
	"syscall"

	"dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	process "dappco.re/go/core/process"
	processexec "dappco.re/go/core/process/exec"
)

// DS returns the current platform directory separator.
//
// Usage example: read ax.DS() when building Core-aware filesystem paths.
func DS() string {
	return core.Env("DS")
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
func Abs(path string) (string, error) {
	if core.PathIsAbs(path) {
		return Clean(path), nil
	}

	cwd, err := Getwd()
	if err != nil {
		return "", err
	}

	return Join(cwd, path), nil
}

// Rel returns target relative to base when target is inside base.
//
// Usage example: rel, err := ax.Rel(projectDir, artifactPath)
func Rel(base, target string) (string, error) {
	base = Clean(base)
	target = Clean(target)

	if base == target {
		return ".", nil
	}

	prefix := base
	if !core.HasSuffix(prefix, DS()) {
		prefix = core.Concat(prefix, DS())
	}

	if core.HasPrefix(target, prefix) {
		return core.TrimPrefix(target, prefix), nil
	}

	return "", coreerr.E("ax.Rel", "path is outside base: "+target, nil)
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
func Getwd() (string, error) {
	cwd := core.Env("DIR_CWD")
	if cwd == "" {
		return "", coreerr.E("ax.Getwd", "current working directory is unavailable", nil)
	}
	return cwd, nil
}

// TempDir creates a temporary directory via Core's filesystem primitive.
//
// Usage example: dir, err := ax.TempDir("core-build-*")
func TempDir(prefix string) (string, error) {
	dir := (&core.Fs{}).NewUnrestricted().TempDir(prefix)
	if dir == "" {
		return "", coreerr.E("ax.TempDir", "failed to create temporary directory", nil)
	}
	return dir, nil
}

// ReadFile reads a file into bytes via io.Local.
//
// Usage example: data, err := ax.ReadFile("go.mod")
func ReadFile(path string) ([]byte, error) {
	content, err := coreio.Local.Read(path)
	if err != nil {
		return nil, err
	}
	return []byte(content), nil
}

// WriteFile writes bytes via io.Local with an explicit mode.
//
// Usage example: err := ax.WriteFile("README.md", []byte("hi"), 0o644)
func WriteFile(path string, data []byte, mode fs.FileMode) error {
	return coreio.Local.WriteMode(path, string(data), mode)
}

// WriteString writes text via io.Local with an explicit mode.
//
// Usage example: err := ax.WriteString("README.md", "hi", 0o644)
func WriteString(path, data string, mode fs.FileMode) error {
	return coreio.Local.WriteMode(path, data, mode)
}

// MkdirAll ensures a directory exists.
//
// Usage example: err := ax.MkdirAll("dist/linux_arm64", 0o755)
func MkdirAll(path string, _ fs.FileMode) error {
	return coreio.Local.EnsureDir(path)
}

// Mkdir ensures a directory exists.
//
// Usage example: err := ax.Mkdir(".core", 0o755)
func Mkdir(path string, _ fs.FileMode) error {
	return coreio.Local.EnsureDir(path)
}

// RemoveAll removes a file or directory tree.
//
// Usage example: err := ax.RemoveAll("dist")
func RemoveAll(path string) error {
	return coreio.Local.DeleteAll(path)
}

// Stat returns file metadata from io.Local.
//
// Usage example: info, err := ax.Stat("go.mod")
func Stat(path string) (fs.FileInfo, error) {
	return coreio.Local.Stat(path)
}

// ReadDir lists directory entries via io.Local.
//
// Usage example: entries, err := ax.ReadDir("dist")
func ReadDir(path string) ([]fs.DirEntry, error) {
	return coreio.Local.List(path)
}

// Open opens a file for reading via io.Local.
//
// Usage example: file, err := ax.Open("README.md")
func Open(path string) (fs.File, error) {
	return coreio.Local.Open(path)
}

// Create opens a file for writing via io.Local.
//
// Usage example: file, err := ax.Create("dist/output.txt")
func Create(path string) (io.WriteCloser, error) {
	return coreio.Local.Create(path)
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
func Chmod(path string, mode fs.FileMode) error {
	return syscall.Chmod(path, uint32(mode))
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
func JSONMarshal(value any) (string, error) {
	result := core.JSONMarshal(value)
	if !result.OK {
		return "", result.Value.(error)
	}
	return string(result.Value.([]byte)), nil
}

// JSONUnmarshal decodes JSON into target using Core's JSON wrapper.
//
// Usage example: err := ax.JSONUnmarshal(data, &cfg)
func JSONUnmarshal(data []byte, target any) error {
	result := core.JSONUnmarshal(data, target)
	if !result.OK {
		return result.Value.(error)
	}
	return nil
}

// LookPath resolves a program on PATH via the Core process package.
//
// Usage example: path, err := ax.LookPath("git")
func LookPath(name string) (string, error) {
	program := process.Program{Name: name}
	if err := program.Find(); err != nil {
		return "", err
	}
	return program.Path, nil
}

// Run executes a command and returns trimmed combined output.
//
// Usage example: output, err := ax.Run(ctx, "git", "status", "--short")
func Run(ctx context.Context, command string, args ...string) (string, error) {
	program := process.Program{Name: command}
	return program.Run(ctx, args...)
}

// RunDir executes a command in the provided directory and returns combined output.
//
// Usage example: output, err := ax.RunDir(ctx, repoDir, "git", "log", "--oneline")
func RunDir(ctx context.Context, dir, command string, args ...string) (string, error) {
	program := process.Program{Name: command}
	return program.RunDir(ctx, dir, args...)
}

// Exec executes a command without capturing output.
//
// Usage example: err := ax.Exec(ctx, "go", "test", "./...")
func Exec(ctx context.Context, command string, args ...string) error {
	return processexec.Command(ctx, command, args...).Run()
}

// ExecDir executes a command in a specific directory without capturing output.
//
// Usage example: err := ax.ExecDir(ctx, repoDir, "go", "test", "./...")
func ExecDir(ctx context.Context, dir, command string, args ...string) error {
	return processexec.Command(ctx, command, args...).WithDir(dir).Run()
}

// ExecWithEnv executes a command with additional environment variables.
//
// Usage example: err := ax.ExecWithEnv(ctx, repoDir, []string{"GOOS=linux"}, "go", "build")
func ExecWithEnv(ctx context.Context, dir string, env []string, command string, args ...string) error {
	return processexec.Command(ctx, command, args...).WithDir(dir).WithEnv(env).Run()
}

// ExecWithWriters executes a command and streams output to the provided writers.
//
// Usage example: err := ax.ExecWithWriters(ctx, repoDir, nil, w, w, "docker", "build", ".")
func ExecWithWriters(ctx context.Context, dir string, env []string, stdout, stderr io.Writer, command string, args ...string) error {
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
func CombinedOutput(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
	cmd := processexec.Command(ctx, command, args...).WithDir(dir).WithEnv(env)
	output, err := cmd.CombinedOutput()
	return core.Trim(string(output)), err
}

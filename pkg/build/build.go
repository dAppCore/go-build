// Package build provides project type detection and cross-compilation for the Core build system.
// It supports Go, Wails, Node.js, PHP, and Rust projects with automatic detection based on
// marker files (go.mod, wails.json, package.json, composer.json, Cargo.toml).
package build

import (
	"context"

	"dappco.re/go/core/io"
)

// ProjectType represents a detected project type.
//
// var t build.ProjectType = build.ProjectTypeGo
type ProjectType string

// Project type constants for build detection.
const (
	// ProjectTypeGo indicates a standard Go project with go.mod.
	ProjectTypeGo ProjectType = "go"
	// ProjectTypeWails indicates a Wails desktop application.
	ProjectTypeWails ProjectType = "wails"
	// ProjectTypeNode indicates a Node.js project with package.json.
	ProjectTypeNode ProjectType = "node"
	// ProjectTypePHP indicates a PHP/Laravel project with composer.json.
	ProjectTypePHP ProjectType = "php"
	// ProjectTypeCPP indicates a C++ project with CMakeLists.txt.
	ProjectTypeCPP ProjectType = "cpp"
	// ProjectTypeDocker indicates a Docker-based project with Dockerfile.
	ProjectTypeDocker ProjectType = "docker"
	// ProjectTypeLinuxKit indicates a LinuxKit VM configuration.
	ProjectTypeLinuxKit ProjectType = "linuxkit"
	// ProjectTypeTaskfile indicates a project using Taskfile automation.
	ProjectTypeTaskfile ProjectType = "taskfile"
	// ProjectTypeDocs indicates a documentation project with mkdocs.yml.
	ProjectTypeDocs ProjectType = "docs"
	// ProjectTypePython indicates a Python project with pyproject.toml or requirements.txt.
	ProjectTypePython ProjectType = "python"
	// ProjectTypeRust indicates a Rust project with Cargo.toml.
	ProjectTypeRust ProjectType = "rust"
)

// Target represents a build target platform.
//
// t := build.Target{OS: "linux", Arch: "amd64"}
type Target struct {
	OS   string
	Arch string
}

// String returns the target in GOOS/GOARCH format.
//
// s := t.String() // → "linux/amd64"
func (t Target) String() string {
	return t.OS + "/" + t.Arch
}

// Artifact represents a build output file.
//
// a := build.Artifact{Path: "dist/linux_amd64/myapp", OS: "linux", Arch: "amd64"}
type Artifact struct {
	Path     string
	OS       string
	Arch     string
	Checksum string
}

// Config holds build configuration.
//
// cfg := &build.Config{FS: io.Local, ProjectDir: ".", OutputDir: "dist", Name: "myapp"}
type Config struct {
	// FS is the medium used for file operations.
	FS io.Medium
	// Project holds build-time project metadata.
	Project Project
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// OutputDir is where build artifacts are placed.
	OutputDir string
	// Name is the output binary name.
	Name string
	// Version is the build version string.
	Version string
	// LDFlags are additional linker flags.
	LDFlags []string
	// Flags are additional build flags.
	Flags []string
	// BuildTags are Go build tags passed through to `go build`.
	BuildTags []string
	// Env are additional environment variables.
	Env []string
	// CGO enables CGO for the build (required for Wails, FrankenPHP, etc).
	CGO bool
	// Obfuscate uses garble instead of go build for binary obfuscation.
	Obfuscate bool
	// NSIS enables Windows NSIS installer generation (Wails projects only).
	NSIS bool
	// WebView2 sets the WebView2 delivery method: download|embed|browser|error.
	WebView2 string

	// Docker-specific config
	Dockerfile string            // Path to Dockerfile (default: Dockerfile)
	Registry   string            // Container registry (default: ghcr.io)
	Image      string            // Image name (owner/repo format)
	Tags       []string          // Additional tags to apply
	BuildArgs  map[string]string // Docker build arguments
	Push       bool              // Whether to push after build

	// LinuxKit-specific config
	LinuxKitConfig string   // Path to LinuxKit YAML config
	Formats        []string // Output formats (iso, qcow2, raw, vmdk)
}

// Builder defines the interface for project-specific build implementations.
//
// var b build.Builder = builders.NewGoBuilder()
// artifacts, err := b.Build(ctx, cfg, targets)
type Builder interface {
	// Name returns the builder's identifier.
	Name() string
	// Detect checks if this builder can handle the project in the given directory.
	Detect(fs io.Medium, dir string) (bool, error)
	// Build compiles the project for the specified targets.
	Build(ctx context.Context, cfg *Config, targets []Target) ([]Artifact, error)
}

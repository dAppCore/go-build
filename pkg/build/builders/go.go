// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"os"
	"runtime"
	"strings"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// GoBuilder implements the Builder interface for Go projects.
//
// b := builders.NewGoBuilder()
type GoBuilder struct{}

// NewGoBuilder creates a new GoBuilder instance.
//
// b := builders.NewGoBuilder()
func NewGoBuilder() *GoBuilder {
	return &GoBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "go"
func (b *GoBuilder) Name() string {
	return "go"
}

// Detect checks if this builder can handle the project in the given directory.
// Uses IsGoProject from the build package which checks for go.mod or wails.json.
//
// ok, err := b.Detect(io.Local, ".")
func (b *GoBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsGoProject(fs, dir), nil
}

// Build compiles the Go project for the specified targets.
// If targets is empty, it falls back to the current host platform.
// It sets GOOS, GOARCH, and CGO_ENABLED, applies config-defined build flags
// and ldflags, and uses garble when obfuscation is enabled.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *GoBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("GoBuilder.Build", "config is nil", nil)
	}

	if len(targets) == 0 {
		targets = []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(cfg.ProjectDir, "dist")
	}
	cfg.OutputDir = outputDir

	// Ensure output directory exists
	if err := cfg.FS.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("GoBuilder.Build", "failed to create output directory", err)
	}

	var artifacts []build.Artifact

	for _, target := range targets {
		artifact, err := b.buildTarget(ctx, cfg, target)
		if err != nil {
			return artifacts, coreerr.E("GoBuilder.Build", "failed to build "+target.String(), err)
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// buildTarget compiles for a single target platform.
func (b *GoBuilder) buildTarget(ctx context.Context, cfg *build.Config, target build.Target) (build.Artifact, error) {
	// Determine output binary name
	binaryName := cfg.Name
	if binaryName == "" {
		binaryName = cfg.Project.Binary
	}
	if binaryName == "" {
		binaryName = cfg.Project.Name
	}
	if binaryName == "" {
		binaryName = ax.Base(cfg.ProjectDir)
	}

	// Add .exe extension for Windows
	if target.OS == "windows" && !core.HasSuffix(binaryName, ".exe") {
		binaryName += ".exe"
	}

	// Create platform-specific output path: output/os_arch/binary
	platformDir := ax.Join(cfg.OutputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	if err := cfg.FS.EnsureDir(platformDir); err != nil {
		return build.Artifact{}, coreerr.E("GoBuilder.buildTarget", "failed to create platform directory", err)
	}

	outputPath := ax.Join(platformDir, binaryName)

	// Build the go/garble arguments.
	args := []string{"build"}
	if !containsString(cfg.Flags, "-trimpath") {
		args = append(args, "-trimpath")
	}
	if len(cfg.Flags) > 0 {
		args = append(args, cfg.Flags...)
	}

	if len(cfg.BuildTags) > 0 {
		args = append(args, "-tags", core.Join(",", cfg.BuildTags...))
	}

	// Add ldflags if specified, and inject the build version when needed.
	ldflags := append([]string{}, cfg.LDFlags...)
	if cfg.Version != "" && !hasVersionLDFlag(ldflags) {
		ldflags = append(ldflags, core.Sprintf("-X main.version=%s", cfg.Version))
	}
	if len(ldflags) > 0 {
		args = append(args, "-ldflags", core.Join(" ", ldflags...))
	}

	// Add output path
	args = append(args, "-o", outputPath)

	// Build the configured main package path, defaulting to the project root.
	mainPackage := cfg.Project.Main
	if mainPackage == "" {
		mainPackage = "."
	}
	args = append(args, mainPackage)

	// Set up environment.
	env := append([]string{}, cfg.Env...)
	env = append(env, build.CacheEnvironment(&cfg.Cache)...)
	env = append(env,
		core.Sprintf("TARGET_OS=%s", target.OS),
		core.Sprintf("TARGET_ARCH=%s", target.Arch),
		core.Sprintf("OUTPUT_DIR=%s", cfg.OutputDir),
		core.Sprintf("TARGET_DIR=%s", platformDir),
		core.Sprintf("GOOS=%s", target.OS),
		core.Sprintf("GOARCH=%s", target.Arch),
	)
	if binaryName != "" {
		env = append(env, core.Sprintf("NAME=%s", binaryName))
	}
	if cfg.Version != "" {
		env = append(env, core.Sprintf("VERSION=%s", cfg.Version))
	}
	if cfg.CGO {
		env = append(env, "CGO_ENABLED=1")
	} else {
		env = append(env, "CGO_ENABLED=0")
	}

	command := "go"
	var err error
	if cfg.Obfuscate {
		command, err = b.resolveGarbleCli()
		if err != nil {
			return build.Artifact{}, err
		}
	}

	// Capture output for error messages
	output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, env, command, args...)
	if err != nil {
		return build.Artifact{}, coreerr.E("GoBuilder.buildTarget", command+" build failed: "+output, err)
	}

	return build.Artifact{
		Path: outputPath,
		OS:   target.OS,
		Arch: target.Arch,
	}, nil
}

// resolveGarbleCli returns the executable path for the garble CLI.
//
// command, err := b.resolveGarbleCli()
func (b *GoBuilder) resolveGarbleCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/garble",
			"/opt/homebrew/bin/garble",
		}

		paths = append(paths, garbleInstallPaths()...)

		if home := core.Env("HOME"); home != "" {
			paths = append(paths, ax.Join(home, "go", "bin", "garble"))
		}
	}

	command, err := ax.ResolveCommand("garble", paths...)
	if err != nil {
		return "", coreerr.E("GoBuilder.resolveGarbleCli", "garble CLI not found. Install it with: go install mvdan.cc/garble@latest", err)
	}

	return command, nil
}

// garbleInstallPaths returns the standard Go install locations for garble.
func garbleInstallPaths() []string {
	var paths []string

	if gobin := core.Env("GOBIN"); gobin != "" {
		paths = append(paths, ax.Join(gobin, "garble"))
	}

	if gopath := core.Env("GOPATH"); gopath != "" {
		for _, root := range strings.Split(gopath, string(os.PathListSeparator)) {
			root = strings.TrimSpace(root)
			if root == "" {
				continue
			}
			paths = append(paths, ax.Join(root, "bin", "garble"))
		}
	}

	return paths
}

// hasVersionLDFlag reports whether a version linker flag is already present.
func hasVersionLDFlag(ldflags []string) bool {
	for _, flag := range ldflags {
		if strings.Contains(flag, "main.version=") || strings.Contains(flag, "main.Version=") {
			return true
		}
	}
	return false
}

// containsString reports whether a slice contains the given string.
func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

// Ensure GoBuilder implements the Builder interface.
var _ build.Builder = (*GoBuilder)(nil)

// Package builders provides build implementations for different project types.
package builders

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// GoBuilder implements the Builder interface for Go projects.
// Usage example: declare a value of type builders.GoBuilder in integrating code.
type GoBuilder struct{}

// NewGoBuilder creates a new GoBuilder instance.
// Usage example: call builders.NewGoBuilder(...) from integrating code.
func NewGoBuilder() *GoBuilder {
	return &GoBuilder{}
}

// Name returns the builder's identifier.
// Usage example: call value.Name(...) from integrating code.
func (b *GoBuilder) Name() string {
	return "go"
}

// Detect checks if this builder can handle the project in the given directory.
// Uses IsGoProject from the build package which checks for go.mod or wails.json.
// Usage example: call value.Detect(...) from integrating code.
func (b *GoBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsGoProject(fs, dir), nil
}

// Build compiles the Go project for the specified targets.
// It sets GOOS, GOARCH, and CGO_ENABLED environment variables,
// applies ldflags and trimpath, and runs go build.
// Usage example: call value.Build(...) from integrating code.
func (b *GoBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("GoBuilder.Build", "config is nil", nil)
	}

	if len(targets) == 0 {
		return nil, coreerr.E("GoBuilder.Build", "no targets specified", nil)
	}

	// Ensure output directory exists
	if err := cfg.FS.EnsureDir(cfg.OutputDir); err != nil {
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

	// Build the go build arguments
	args := []string{"build"}

	// Add trimpath flag
	args = append(args, "-trimpath")

	// Add ldflags if specified
	if len(cfg.LDFlags) > 0 {
		ldflags := core.Join(" ", cfg.LDFlags...)
		args = append(args, "-ldflags", ldflags)
	}

	// Add output path
	args = append(args, "-o", outputPath)

	// Add the project directory as the build target (current directory)
	args = append(args, ".")

	// Set up environment
	env := []string{
		core.Sprintf("GOOS=%s", target.OS),
		core.Sprintf("GOARCH=%s", target.Arch),
	}
	if cfg.CGO {
		env = append(env, "CGO_ENABLED=1")
	} else {
		env = append(env, "CGO_ENABLED=0")
	}

	// Capture output for error messages
	output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, env, "go", args...)
	if err != nil {
		return build.Artifact{}, coreerr.E("GoBuilder.buildTarget", "go build failed: "+output, err)
	}

	return build.Artifact{
		Path: outputPath,
		OS:   target.OS,
		Arch: target.Arch,
	}, nil
}

// Ensure GoBuilder implements the Builder interface.
var _ build.Builder = (*GoBuilder)(nil)

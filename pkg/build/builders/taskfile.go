// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"path"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// TaskfileBuilder builds projects using Taskfile (https://taskfile.dev/).
// This is a generic builder that can handle any project type that has a Taskfile.
// Usage example: declare a value of type builders.TaskfileBuilder in integrating code.
type TaskfileBuilder struct{}

// NewTaskfileBuilder creates a new Taskfile builder.
// Usage example: call builders.NewTaskfileBuilder(...) from integrating code.
func NewTaskfileBuilder() *TaskfileBuilder {
	return &TaskfileBuilder{}
}

// Name returns the builder's identifier.
// Usage example: call value.Name(...) from integrating code.
func (b *TaskfileBuilder) Name() string {
	return "taskfile"
}

// Detect checks if a Taskfile exists in the directory.
// Usage example: call value.Detect(...) from integrating code.
func (b *TaskfileBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	// Check for Taskfile.yml, Taskfile.yaml, or Taskfile
	taskfiles := []string{
		"Taskfile.yml",
		"Taskfile.yaml",
		"Taskfile",
		"taskfile.yml",
		"taskfile.yaml",
	}

	for _, tf := range taskfiles {
		if fs.IsFile(ax.Join(dir, tf)) {
			return true, nil
		}
	}
	return false, nil
}

// Build runs the Taskfile build task for each target platform.
// Usage example: call value.Build(...) from integrating code.
func (b *TaskfileBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	taskCommand, err := b.resolveTaskCli()
	if err != nil {
		return nil, err
	}

	// Create output directory
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(cfg.ProjectDir, "dist")
	}
	if err := cfg.FS.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("TaskfileBuilder.Build", "failed to create output directory", err)
	}

	var artifacts []build.Artifact

	// If no targets specified, just run the build task once
	if len(targets) == 0 {
		if err := b.runTask(ctx, cfg, taskCommand, "", ""); err != nil {
			return nil, err
		}

		// Try to find artifacts in output directory
		found := b.findArtifacts(cfg.FS, outputDir)
		artifacts = append(artifacts, found...)
	} else {
		// Run build task for each target
		for _, target := range targets {
			if err := b.runTask(ctx, cfg, taskCommand, target.OS, target.Arch); err != nil {
				return nil, err
			}

			// Try to find artifacts for this target
			found := b.findArtifactsForTarget(cfg.FS, outputDir, target)
			artifacts = append(artifacts, found...)
		}
	}

	return artifacts, nil
}

// runTask executes the Taskfile build task.
func (b *TaskfileBuilder) runTask(ctx context.Context, cfg *build.Config, taskCommand, goos, goarch string) error {
	// Build task command
	args := []string{"build"}
	env := []string{}

	// Pass variables if targets are specified
	if goos != "" {
		value := core.Sprintf("GOOS=%s", goos)
		args = append(args, value)
		env = append(env, value)
	}
	if goarch != "" {
		value := core.Sprintf("GOARCH=%s", goarch)
		args = append(args, value)
		env = append(env, value)
	}
	if cfg.OutputDir != "" {
		value := core.Sprintf("OUTPUT_DIR=%s", cfg.OutputDir)
		args = append(args, value)
		env = append(env, value)
	}
	if cfg.Name != "" {
		value := core.Sprintf("NAME=%s", cfg.Name)
		args = append(args, value)
		env = append(env, value)
	}
	if cfg.Version != "" {
		value := core.Sprintf("VERSION=%s", cfg.Version)
		args = append(args, value)
		env = append(env, value)
	}

	if goos != "" && goarch != "" {
		core.Print(nil, "Running task build for %s/%s", goos, goarch)
	} else {
		core.Print(nil, "Running task build")
	}

	if err := ax.ExecWithEnv(ctx, cfg.ProjectDir, env, taskCommand, args...); err != nil {
		return coreerr.E("TaskfileBuilder.runTask", "task build failed", err)
	}

	return nil
}

// findArtifacts searches for built artifacts in the output directory.
func (b *TaskfileBuilder) findArtifacts(fs io.Medium, outputDir string) []build.Artifact {
	var artifacts []build.Artifact

	entries, err := fs.List(outputDir)
	if err != nil {
		return artifacts
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip common non-artifact files
		name := entry.Name()
		if core.HasPrefix(name, ".") || name == "CHECKSUMS.txt" {
			continue
		}

		artifacts = append(artifacts, build.Artifact{
			Path: ax.Join(outputDir, name),
			OS:   "",
			Arch: "",
		})
	}

	return artifacts
}

// findArtifactsForTarget searches for built artifacts for a specific target.
func (b *TaskfileBuilder) findArtifactsForTarget(fs io.Medium, outputDir string, target build.Target) []build.Artifact {
	var artifacts []build.Artifact

	// 1. Look for platform-specific subdirectory: output/os_arch/
	platformSubdir := ax.Join(outputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	if fs.IsDir(platformSubdir) {
		entries, _ := fs.List(platformSubdir)
		for _, entry := range entries {
			if entry.IsDir() {
				// Handle .app bundles on macOS
				if target.OS == "darwin" && core.HasSuffix(entry.Name(), ".app") {
					artifacts = append(artifacts, build.Artifact{
						Path: ax.Join(platformSubdir, entry.Name()),
						OS:   target.OS,
						Arch: target.Arch,
					})
				}
				continue
			}
			// Skip hidden files
			if core.HasPrefix(entry.Name(), ".") {
				continue
			}
			artifacts = append(artifacts, build.Artifact{
				Path: ax.Join(platformSubdir, entry.Name()),
				OS:   target.OS,
				Arch: target.Arch,
			})
		}
		if len(artifacts) > 0 {
			return artifacts
		}
	}

	// 2. Look for files matching the target pattern in the root output dir
	patterns := []string{
		core.Sprintf("*-%s-%s*", target.OS, target.Arch),
		core.Sprintf("*_%s_%s*", target.OS, target.Arch),
		core.Sprintf("*-%s*", target.Arch),
	}

	for _, pattern := range patterns {
		entries, _ := fs.List(outputDir)
		for _, entry := range entries {
			match := entry.Name()
			// Simple glob matching
			if b.matchPattern(match, pattern) {
				fullPath := ax.Join(outputDir, match)
				if fs.IsDir(fullPath) {
					continue
				}

				artifacts = append(artifacts, build.Artifact{
					Path: fullPath,
					OS:   target.OS,
					Arch: target.Arch,
				})
			}
		}

		if len(artifacts) > 0 {
			break // Found matches, stop looking
		}
	}

	return artifacts
}

// matchPattern implements glob matching for Taskfile artifacts.
func (b *TaskfileBuilder) matchPattern(name, pattern string) bool {
	matched, _ := path.Match(pattern, name)
	return matched
}

// resolveTaskCli returns the executable path for the task CLI.
func (b *TaskfileBuilder) resolveTaskCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/task",
			"/opt/homebrew/bin/task",
		}
	}

	command, err := ax.ResolveCommand("task", paths...)
	if err != nil {
		return "", coreerr.E("TaskfileBuilder.resolveTaskCli", "task CLI not found. Install with: brew install go-task (macOS), go install github.com/go-task/task/v3/cmd/task@latest, or see https://taskfile.dev/installation/", err)
	}

	return command, nil
}

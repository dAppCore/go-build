// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"path"
	"runtime"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// TaskfileBuilder builds projects using Taskfile (https://taskfile.dev/).
// This is a generic builder that can handle any project type that has a Taskfile.
//
// b := builders.NewTaskfileBuilder()
type TaskfileBuilder struct{}

// NewTaskfileBuilder creates a new Taskfile builder.
//
// b := builders.NewTaskfileBuilder()
func NewTaskfileBuilder() *TaskfileBuilder {
	return &TaskfileBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "taskfile"
func (b *TaskfileBuilder) Name() string {
	return "taskfile"
}

// Detect checks if a Taskfile exists in the directory.
//
// ok, err := b.Detect(io.Local, ".")
func (b *TaskfileBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsTaskfileProject(fs, dir), nil
}

// Build runs the Taskfile build task for each target platform.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *TaskfileBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("TaskfileBuilder.Build", "config is nil", nil)
	}
	filesystem := ensureBuildFilesystem(cfg)

	taskCommand, err := b.resolveTaskCli()
	if err != nil {
		return nil, err
	}

	// Create output directory
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir(cfg)
	}
	if err := ensureOutputDir(filesystem, outputDir, "TaskfileBuilder.Build"); err != nil {
		return nil, err
	}

	var artifacts []build.Artifact

	// If no targets are specified, build the host target so Taskfile builds
	// still receive the standard GOOS/GOARCH surface.
	targets = defaultRuntimeTargets(targets, runtime.GOOS, runtime.GOARCH)

	// Run build task for each target
	for _, target := range targets {
		if err := b.runTask(ctx, cfg, taskCommand, outputDir, target); err != nil {
			return nil, err
		}

		// Try to find artifacts for this target
		found := b.findArtifactsForTarget(cfg.FS, outputDir, target)
		artifacts = append(artifacts, found...)
	}

	return artifacts, nil
}

// runTask executes the Taskfile build task.
func (b *TaskfileBuilder) runTask(ctx context.Context, cfg *build.Config, taskCommand string, outputDir string, target build.Target) error {
	// Build task command
	args := []string{"build"}
	env := build.BuildEnvironment(cfg)
	targetDir := platformDir(outputDir, target)
	values := standardTargetValues(outputDir, targetDir, target)
	if cfg.Name != "" {
		values = append(values, core.Sprintf("NAME=%s", cfg.Name))
	}
	if cfg.Version != "" {
		values = append(values, core.Sprintf("VERSION=%s", cfg.Version))
	}
	values = append(values, cgoEnvValue(cfg.CGO))
	args = append(args, values...)
	env = append(env, values...)

	cleanup := func() {}
	if cfg != nil {
		var err error
		args, env, cleanup, err = b.applyWailsV3BuildSurface(cfg, target, args, env)
		if err != nil {
			return err
		}
	}
	defer cleanup()

	if target.OS != "" && target.Arch != "" {
		core.Print(nil, "Running task build for %s/%s", target.OS, target.Arch)
	} else {
		core.Print(nil, "Running task build")
	}

	if err := ax.ExecWithEnv(ctx, cfg.ProjectDir, env, taskCommand, args...); err != nil {
		return coreerr.E("TaskfileBuilder.runTask", "task build failed", err)
	}

	return nil
}

func (b *TaskfileBuilder) applyWailsV3BuildSurface(cfg *build.Config, target build.Target, args, env []string) ([]string, []string, func(), error) {
	if cfg == nil || cfg.ProjectDir == "" {
		return args, env, func() {}, nil
	}

	fs := cfg.FS
	if fs == nil {
		fs = io.Local
	}

	wailsBuilder := NewWailsBuilder()
	if !build.IsWailsProject(fs, cfg.ProjectDir) || !wailsBuilder.isWailsV3(fs, cfg.ProjectDir) {
		return args, env, func() {}, nil
	}

	if goflags, err := buildV3GoFlags(cfg); err != nil {
		return nil, nil, nil, err
	} else if goflags != "" {
		env = append(env, "GOFLAGS="+goflags)
	}

	taskVars, err := buildV3TaskVars(cfg, target)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(taskVars) > 0 {
		args = append(args, taskVars...)
		env = append(env, taskVars...)
	}

	if !cfg.Obfuscate {
		return args, env, func() {}, nil
	}

	env, cleanup, err := wailsBuilder.prepareV3Obfuscation(env)
	if err != nil {
		return nil, nil, nil, err
	}

	return args, env, cleanup, nil
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

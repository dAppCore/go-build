package projectdetect

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core/io"
)

func TestDetectProjectType_Good(t *testing.T) {
	fs := io.Local

	t.Run("prefers configured build type from .core/build.yaml even without markers", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: docker\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeDocker, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeDocker, projectType)
		}

	})

	t.Run("prefers core marker types over fallback builders", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeGo, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeGo, projectType)
		}

	})

	t.Run("detects Go workspaces", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.work"), []byte("go 1.22\nuse ./app"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeGo, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeGo, projectType)
		}

	})

	t.Run("detects Docker projects", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeDocker, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeDocker, projectType)
		}

	})

	t.Run("detects LinuxKit projects", func(t *testing.T) {
		dir := t.TempDir()
		linuxkitDir := ax.Join(dir, ".core", "linuxkit")
		if err := ax.MkdirAll(linuxkitDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(linuxkitDir, "server.yml"), []byte("kernel:\n  image: test"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeLinuxKit, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeLinuxKit, projectType)
		}

	})

	t.Run("detects C++ projects", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeCPP, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeCPP, projectType)
		}

	})

	t.Run("detects Taskfile projects", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte("version: '3'"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeTaskfile, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeTaskfile, projectType)
		}

	})

	t.Run("detects nested Node.js projects", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeNode, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeNode, projectType)
		}

	})

	t.Run("detects nested Deno projects", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "site")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "deno.jsonc"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeNode, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeNode, projectType)
		}

	})

	t.Run("prefers generic Node markers over docs", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "mkdocs.yml"), []byte("site_name: Demo"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeNode, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeNode, projectType)
		}

	})

	t.Run("detects Wails projects from go.mod and root package.json", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeWails, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeWails, projectType)
		}

	})

	t.Run("detects Wails monorepos from go.mod and nested frontend manifests", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nested := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		projectType, err := DetectProjectType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(build.ProjectTypeWails, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeWails, projectType)
		}

	})
}

func TestDetectProjectType_Bad(t *testing.T) {
	fs := io.Local

	t.Run("returns empty type for empty directory", func(t *testing.T) {
		projectType, err := DetectProjectType(fs, t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEmpty(projectType) {
			t.Fatalf("expected empty, got %v", projectType)
		}

	})
}

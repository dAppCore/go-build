package builders

import (
	"archive/zip"
	"context"
	stdio "io"
	"os"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

func TestDocs_DocsBuilderName_Good(t *testing.T) {
	builder := NewDocsBuilder()
	if !stdlibAssertEqual("docs", builder.Name()) {
		t.Fatalf("want %v, got %v", "docs", builder.Name())
	}

}

func TestDocs_DocsBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects mkdocs.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "mkdocs.yml"), []byte("site_name: Demo\n"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewDocsBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects mkdocs.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "mkdocs.yaml"), []byte("site_name: Demo\n"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewDocsBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false without mkdocs.yml", func(t *testing.T) {
		builder := NewDocsBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestDocs_DocsBuilderBuild_Good(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mkdocs test fixture uses a shell script")
	}

	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "mkdocs.yaml"), []byte("site_name: Demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binDir := t.TempDir()
	mkdocsPath := ax.Join(binDir, "mkdocs")
	script := "#!/bin/sh\nset -eu\nif [ -n \"${DOCS_BUILD_LOG_FILE:-}\" ]; then\n  env | sort > \"${DOCS_BUILD_LOG_FILE}\"\nfi\nsite_dir=\"\"\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--site-dir\" ]; then\n    shift\n    site_dir=\"$1\"\n  fi\n  shift\ndone\nmkdir -p \"$site_dir\"\nprintf '%s' 'demo docs' > \"$site_dir/index.html\"\n"
	if err := ax.WriteFile(mkdocsPath, []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	logPath := ax.Join(t.TempDir(), "docs.env")

	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: dir,
		OutputDir:  ax.Join(dir, "dist"),
		Name:       "demo-site",
		Env:        []string{"FOO=bar", "DOCS_BUILD_LOG_FILE=" + logPath},
	}

	builder := NewDocsBuilder()
	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	artifact := artifacts[0]
	if !stdlibAssertEqual("linux", artifact.OS) {
		t.Fatalf("want %v, got %v", "linux", artifact.OS)
	}
	if !stdlibAssertEqual("amd64", artifact.Arch) {
		t.Fatalf("want %v, got %v", "amd64", artifact.Arch)
	}
	if _, err := os.Stat(artifact.Path); err != nil {
		t.Fatalf("expected file to exist: %v", artifact.Path)
	}

	reader, err := zip.OpenReader(artifact.Path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = reader.Close() }()
	if len(reader.File) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(reader.File))
	}
	if !stdlibAssertEqual("index.html", reader.File[0].Name) {
		t.Fatalf("want %v, got %v", "index.html", reader.File[0].Name)
	}

	file, err := reader.File[0].Open()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = file.Close() }()

	data, err := stdio.ReadAll(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("demo docs", string(data)) {
		t.Fatalf("want %v, got %v", "demo docs", string(data))
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(content), "FOO=bar") {
		t.Fatalf("expected %v to contain %v", string(content), "FOO=bar")
	}
	if !stdlibAssertContains(string(content), "GOOS=linux") {
		t.Fatalf("expected %v to contain %v", string(content), "GOOS=linux")
	}
	if !stdlibAssertContains(string(content), "GOARCH=amd64") {
		t.Fatalf("expected %v to contain %v", string(content), "GOARCH=amd64")
	}
	if !stdlibAssertContains(string(content), "TARGET_OS=linux") {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_OS=linux")
	}
	if !stdlibAssertContains(string(content), "TARGET_ARCH=amd64") {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_ARCH=amd64")
	}
	if !stdlibAssertContains(string(content), "OUTPUT_DIR="+ax.Join(dir, "dist")) {
		t.Fatalf("expected %v to contain %v", string(content), "OUTPUT_DIR="+ax.Join(dir, "dist"))
	}
	if !stdlibAssertContains(string(content), "TARGET_DIR="+ax.Join(dir, "dist", "linux_amd64")) {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_DIR="+ax.Join(dir, "dist", "linux_amd64"))
	}
	if !stdlibAssertContains(string(content), "NAME=demo-site") {
		t.Fatalf("expected %v to contain %v", string(content), "NAME=demo-site")
	}

}

func TestDocs_DocsBuilderBuild_Good_NestedConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mkdocs test fixture uses a shell script")
	}

	dir := t.TempDir()
	if err := ax.MkdirAll(ax.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "docs", "mkdocs.yaml"), []byte("site_name: Demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binDir := t.TempDir()
	mkdocsPath := ax.Join(binDir, "mkdocs")
	script := "#!/bin/sh\nset -eu\nif [ -n \"${DOCS_BUILD_LOG_FILE:-}\" ]; then\n  env | sort >> \"${DOCS_BUILD_LOG_FILE}\"\n  printf '%s\\n' \"$@\" >> \"${DOCS_BUILD_LOG_FILE}\"\nfi\nsite_dir=\"\"\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--site-dir\" ]; then\n    shift\n    site_dir=\"$1\"\n  fi\n  shift\ndone\nmkdir -p \"$site_dir\"\nprintf '%s' 'demo docs' > \"$site_dir/index.html\"\n"
	if err := ax.WriteFile(mkdocsPath, []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	logPath := ax.Join(t.TempDir(), "docs.args")

	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: dir,
		OutputDir:  ax.Join(dir, "dist"),
		Name:       "demo-site",
		Env:        []string{"DOCS_BUILD_LOG_FILE=" + logPath},
	}

	builder := NewDocsBuilder()
	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(content), "--config-file") {
		t.Fatalf("expected %v to contain %v", string(content), "--config-file")
	}
	if !stdlibAssertContains(string(content), "docs/mkdocs.yaml") {
		t.Fatalf("expected %v to contain %v", string(content), "docs/mkdocs.yaml")
	}
	if !stdlibAssertContains(string(content), "TARGET_DIR="+ax.Join(dir, "dist", "linux_amd64")) {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_DIR="+ax.Join(dir, "dist", "linux_amd64"))
	}

}

func TestDocs_DocsBuilderBuild_Bad(t *testing.T) {
	builder := NewDocsBuilder()

	t.Run("returns error when config is nil", func(t *testing.T) {
		artifacts, err := builder.Build(context.Background(), nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertNil(artifacts) {
			t.Fatalf("expected nil, got %v", artifacts)
		}

	})

	t.Run("returns error when mkdocs.yml is missing", func(t *testing.T) {
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: t.TempDir(),
			OutputDir:  t.TempDir(),
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertNil(artifacts) {
			t.Fatalf("expected nil, got %v", artifacts)
		}

	})
}

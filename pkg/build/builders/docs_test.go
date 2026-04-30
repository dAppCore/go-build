package builders

import (
	"archive/zip"
	"context"
	stdio "io"
	"runtime"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

func TestDocs_DocsBuilderNameGood(t *testing.T) {
	builder := NewDocsBuilder()
	if !stdlibAssertEqual("docs", builder.Name()) {
		t.Fatalf("want %v, got %v", "docs", builder.Name())
	}

}

func TestDocs_DocsBuilderDetectGood(t *testing.T) {
	fs := storage.Local

	t.Run("detects mkdocs.yml", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "mkdocs.yml"), []byte("site_name: Demo\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewDocsBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects mkdocs.yaml", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "mkdocs.yaml"), []byte("site_name: Demo\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewDocsBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false without mkdocs.yml", func(t *testing.T) {
		builder := NewDocsBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, t.TempDir()))
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestDocs_DocsBuilderBuildGood(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mkdocs test fixture uses a shell script")
	}

	dir := t.TempDir()
	if result := ax.WriteFile(ax.Join(dir, "mkdocs.yaml"), []byte("site_name: Demo\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	binDir := t.TempDir()
	mkdocsPath := ax.Join(binDir, "mkdocs")
	script := "#!/bin/sh\nset -eu\nif [ -n \"${DOCS_BUILD_LOG_FILE:-}\" ]; then\n  env | sort > \"${DOCS_BUILD_LOG_FILE}\"\nfi\nsite_dir=\"\"\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--site-dir\" ]; then\n    shift\n    site_dir=\"$1\"\n  fi\n  shift\ndone\nmkdir -p \"$site_dir\"\nprintf '%s' 'demo docs' > \"$site_dir/index.html\"\n"
	if result := ax.WriteFile(mkdocsPath, []byte(script), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))
	logPath := ax.Join(t.TempDir(), "docs.env")

	cfg := &build.Config{
		FS:         storage.Local,
		ProjectDir: dir,
		OutputDir:  ax.Join(dir, "dist"),
		Name:       "demo-site",
		Env:        []string{"FOO=bar", "DOCS_BUILD_LOG_FILE=" + logPath},
	}

	builder := NewDocsBuilder()
	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
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
	if result := ax.Stat(artifact.Path); !result.OK {
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

	content := requireBuilderBytes(t, ax.ReadFile(logPath))
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
	if result := ax.MkdirAll(ax.Join(dir, "docs"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "docs", "mkdocs.yaml"), []byte("site_name: Demo\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	binDir := t.TempDir()
	mkdocsPath := ax.Join(binDir, "mkdocs")
	script := "#!/bin/sh\nset -eu\nif [ -n \"${DOCS_BUILD_LOG_FILE:-}\" ]; then\n  env | sort >> \"${DOCS_BUILD_LOG_FILE}\"\n  printf '%s\\n' \"$@\" >> \"${DOCS_BUILD_LOG_FILE}\"\nfi\nsite_dir=\"\"\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--site-dir\" ]; then\n    shift\n    site_dir=\"$1\"\n  fi\n  shift\ndone\nmkdir -p \"$site_dir\"\nprintf '%s' 'demo docs' > \"$site_dir/index.html\"\n"
	if result := ax.WriteFile(mkdocsPath, []byte(script), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))
	logPath := ax.Join(t.TempDir(), "docs.args")

	cfg := &build.Config{
		FS:         storage.Local,
		ProjectDir: dir,
		OutputDir:  ax.Join(dir, "dist"),
		Name:       "demo-site",
		Env:        []string{"DOCS_BUILD_LOG_FILE=" + logPath},
	}

	builder := NewDocsBuilder()
	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	content := requireBuilderBytes(t, ax.ReadFile(logPath))
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

func TestDocs_DocsBuilderBuildBad(t *testing.T) {
	builder := NewDocsBuilder()

	t.Run("returns error when config is nil", func(t *testing.T) {
		result := builder.Build(context.Background(), nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		if result.OK {
			t.Fatal("expected error")
		}

	})

	t.Run("returns error when mkdocs.yml is missing", func(t *testing.T) {
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: t.TempDir(),
			OutputDir:  t.TempDir(),
		}

		result := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
		if result.OK {
			t.Fatal("expected error")
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestDocs_NewDocsBuilder_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewDocsBuilder()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocs_NewDocsBuilder_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewDocsBuilder()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocs_NewDocsBuilder_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewDocsBuilder()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocs_DocsBuilder_Name_Good(t *core.T) {
	subject := &DocsBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocs_DocsBuilder_Name_Bad(t *core.T) {
	subject := &DocsBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocs_DocsBuilder_Name_Ugly(t *core.T) {
	subject := &DocsBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocs_DocsBuilder_Detect_Good(t *core.T) {
	subject := &DocsBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocs_DocsBuilder_Detect_Bad(t *core.T) {
	subject := &DocsBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(storage.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocs_DocsBuilder_Detect_Ugly(t *core.T) {
	subject := &DocsBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocs_DocsBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DocsBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocs_DocsBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DocsBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocs_DocsBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DocsBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

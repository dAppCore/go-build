package service

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestShouldSkipWatchPath_Good(t *testing.T) {
	projectDir := t.TempDir()
	if !(shouldSkipWatchPath(projectDir, filepath.Join(projectDir, ".git", "HEAD"))) {
		t.Fatal("expected true")
	}
	if !(shouldSkipWatchPath(projectDir, filepath.Join(projectDir, "dist", "core-build"))) {
		t.Fatal("expected true")
	}
	if !(shouldSkipWatchPath(projectDir, filepath.Join(projectDir, ".core", "cache", "state.json"))) {
		t.Fatal("expected true")
	}
	if shouldSkipWatchPath(projectDir, filepath.Join(projectDir, ".core", "build.yaml")) {
		t.Fatal("expected false")
	}
	if shouldSkipWatchPath(projectDir, filepath.Join(projectDir, "cmd", "main.go")) {
		t.Fatal("expected false")
	}

}

func TestSnapshotFiles_ExcludesGeneratedOutputs_Good(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "cmd"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "dist"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "cmd", "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "dist", "app"), []byte("binary"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".git", "HEAD"), []byte("ref: refs/heads/dev"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshot, err := snapshotFiles(Config{ProjectDir: projectDir, WatchPaths: []string{projectDir}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(snapshot, filepath.Join(projectDir, "cmd", "main.go")) {
		t.Fatalf("expected %v to contain %v", snapshot, filepath.Join(projectDir, "cmd", "main.go"))
	}
	if stdlibAssertContains(snapshot, filepath.Join(projectDir, "dist", "app")) {
		t.Fatalf("expected %v not to contain %v", snapshot, filepath.Join(projectDir, "dist", "app"))
	}
	if stdlibAssertContains(snapshot, filepath.Join(projectDir, ".git", "HEAD")) {
		t.Fatalf("expected %v not to contain %v", snapshot, filepath.Join(projectDir, ".git", "HEAD"))
	}

}

func TestDiffSnapshots_Good(t *testing.T) {
	now := time.Now()
	before := map[string]time.Time{
		"/tmp/a": now,
		"/tmp/b": now,
	}
	after := map[string]time.Time{
		"/tmp/a": now.Add(time.Second),
		"/tmp/c": now,
	}

	changed := diffSnapshots(before, after)
	if !stdlibAssertEqual([]string{"/tmp/a", "/tmp/b", "/tmp/c"}, changed) {
		t.Fatalf("want %v, got %v", []string{"/tmp/a", "/tmp/b", "/tmp/c"}, changed)
	}

}

func TestDefaultRunWatchedBuild_WithoutBuildConfig_UsesLocalTarget_Good(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/daemon\n\ngo 1.20\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := defaultRunWatchedBuild(context.Background(), projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distDir := filepath.Join(projectDir, "dist")
	entries, err := os.ReadDir(distDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(entries))
	}
	if !stdlibAssertEqual(runtime.GOOS+"_"+runtime.GOARCH, entries[0].Name()) {
		t.Fatalf("want %v, got %v", runtime.GOOS+"_"+runtime.GOARCH, entries[0].Name())
	}

	platformEntries, err := os.ReadDir(filepath.Join(distDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(platformEntries) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(platformEntries))
	}

}

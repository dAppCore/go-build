package builders

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

var wailsArtifactTarget = build.Target{OS: "darwin", Arch: "arm64"}

// findWailsConventionArtifacts reports the bin/ executable a Taskfile build
// wrote there (where wails v3 outputs, not go-build's OUTPUT_DIR).
func TestTaskfileBuilder_findWailsConventionArtifacts_Good(t *testing.T) {
	dir := t.TempDir()
	if r := ax.WriteFile(ax.Join(dir, "bin", "lem-runtime"), []byte("#!/bin/sh\n"), 0o755); !r.OK {
		t.Fatalf("write executable: %v", r.Error())
	}

	got := NewTaskfileBuilder().findWailsConventionArtifacts(storage.Local, dir, wailsArtifactTarget)
	if len(got) != 1 || got[0].Path != ax.Join(dir, "bin", "lem-runtime") {
		t.Fatalf("expected the bin/ executable, got %+v", got)
	}
}

// A macOS .app bundle under bin/ is reported as a directory artifact.
func TestTaskfileBuilder_findWailsConventionArtifacts_AppBundle_Good(t *testing.T) {
	dir := t.TempDir()
	if r := storage.Local.EnsureDir(ax.Join(dir, "bin", "LEM Runtime.app", "Contents", "MacOS")); !r.OK {
		t.Fatalf("mkdir .app: %v", r.Error())
	}

	got := NewTaskfileBuilder().findWailsConventionArtifacts(storage.Local, dir, wailsArtifactTarget)
	if len(got) != 1 || got[0].Path != ax.Join(dir, "bin", "LEM Runtime.app") {
		t.Fatalf("expected the .app bundle, got %+v", got)
	}
}

// No bin/ → no artifacts, so the caller reports the real emptiness.
func TestTaskfileBuilder_findWailsConventionArtifacts_Empty_Bad(t *testing.T) {
	dir := t.TempDir()
	if got := NewTaskfileBuilder().findWailsConventionArtifacts(storage.Local, dir, wailsArtifactTarget); len(got) != 0 {
		t.Fatalf("expected no artifacts, got %+v", got)
	}
}

// Hidden files and loose non-executables in bin/ are not build products.
func TestTaskfileBuilder_findWailsConventionArtifacts_SkipsNoise_Ugly(t *testing.T) {
	dir := t.TempDir()
	if r := ax.WriteFile(ax.Join(dir, "bin", ".gitignore"), []byte("*\n"), 0o644); !r.OK {
		t.Fatalf("write .gitignore: %v", r.Error())
	}
	if r := ax.WriteFile(ax.Join(dir, "bin", "CHECKSUMS.txt"), []byte("x\n"), 0o644); !r.OK {
		t.Fatalf("write checksums: %v", r.Error())
	}

	if got := NewTaskfileBuilder().findWailsConventionArtifacts(storage.Local, dir, wailsArtifactTarget); len(got) != 0 {
		t.Fatalf("expected no artifacts from noise-only bin/, got %+v", got)
	}
}

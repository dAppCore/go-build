package buildcmd

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

const taskfileWithPackage = `version: '3'
tasks:
  build:
    cmds:
      - echo build
  package:
    cmds:
      - echo package
`

// taskfileDeclaresTarget detects a declared package target.
func TestBuildCmd_taskfileDeclaresTarget_Good(t *testing.T) {
	dir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte(taskfileWithPackage), 0o644))

	if !taskfileDeclaresTarget(storage.Local, dir, "package") {
		t.Fatal("expected the package target to be detected")
	}
	if taskfileDeclaresTarget(storage.Local, dir, "missing") {
		t.Fatal("did not expect a 'missing' target to be detected")
	}
}

// taskfileDeclaresTarget reports false when no Taskfile is present.
func TestBuildCmd_taskfileDeclaresTarget_Bad(t *testing.T) {
	dir := t.TempDir() // no Taskfile written

	if taskfileDeclaresTarget(storage.Local, dir, "package") {
		t.Fatal("expected no target when no Taskfile is present")
	}
}

// A namespaced `darwin:package` must not satisfy the bare `package` target.
func TestBuildCmd_taskfileDeclaresTarget_Ugly(t *testing.T) {
	dir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(dir, "Taskfile.yml"),
		[]byte("version: '3'\ntasks:\n  darwin:package:\n    cmds:\n      - echo hi\n"), 0o644))

	if taskfileDeclaresTarget(storage.Local, dir, "package") {
		t.Fatal("darwin:package should not satisfy the bare package target")
	}
}

// Upload flows (notarise / TestFlight / App Store) must not delegate to the
// Taskfile — they need the in-pipeline credential handling.
func TestBuildCmd_tryTaskfileApplePackage_UploadFlowFallsThrough_Good(t *testing.T) {
	dir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte(taskfileWithPackage), 0o644))

	for _, options := range []build.AppleOptions{
		{Sign: true, Notarise: true},
		{Sign: true, TestFlight: true},
		{Sign: true, AppStore: true},
	} {
		_, handled := tryTaskfileApplePackage(context.Background(), storage.Local, dir, options, "v1.0.0")
		if handled {
			t.Fatalf("upload flow %+v must fall through to the generic pipeline", options)
		}
	}
}

// With no Taskfile package target, the delegation falls through so the generic
// build.BuildApple pipeline still runs.
func TestBuildCmd_tryTaskfileApplePackage_NoTargetFallsThrough_Good(t *testing.T) {
	dir := t.TempDir() // no Taskfile written

	_, handled := tryTaskfileApplePackage(context.Background(), storage.Local, dir, build.AppleOptions{Sign: true}, "v1.0.0")
	if handled {
		t.Fatal("no Taskfile package target → delegation must not claim the build")
	}
}

// findAppleBundleArtifact finds a .app under bin/ regardless of its name.
func TestBuildCmd_findAppleBundleArtifact_Good(t *testing.T) {
	dir := t.TempDir()
	requireBuildCmdOK(t, storage.Local.EnsureDir(ax.Join(dir, "bin", "My App.app", "Contents", "MacOS")))

	result := findAppleBundleArtifact(storage.Local, dir)
	if !result.OK {
		t.Fatalf("expected to find the .app bundle: %v", result.Error())
	}
	if got := result.Value.(string); got != ax.Join(dir, "bin", "My App.app") {
		t.Fatalf("unexpected bundle path: %s", got)
	}
}

// findAppleBundleArtifact fails clearly when no .app was produced.
func TestBuildCmd_findAppleBundleArtifact_Bad(t *testing.T) {
	dir := t.TempDir() // no bin/*.app

	if findAppleBundleArtifact(storage.Local, dir).OK {
		t.Fatal("expected failure when no .app bundle exists")
	}
}

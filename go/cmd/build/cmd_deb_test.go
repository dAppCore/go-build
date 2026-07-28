package buildcmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	storage "dappco.re/go/build/pkg/storage"
)

func TestBuildCmd_AddDebCommand_Good(t *testing.T) {
	c := core.New()

	if !AddDebCommand(c).OK {
		t.Fatal("expected build/deb to register")
	}
	if !(c.Command("build/deb").OK) {
		t.Fatal("expected build/deb to be registered")
	}
}

// newDebProject lays out a project with a built binary named the way the build
// action names its release assets.
func newDebProject(t *testing.T, binaryName string) (string, string) {
	t.Helper()
	projectDir := t.TempDir()
	requireBuildCmdOK(t, storage.Local.EnsureDir(ax.Join(projectDir, ".core")))
	requireBuildCmdOK(t, storage.Local.Write(ax.Join(projectDir, ".core", "build.yaml"), `version: 1
project:
  binary: core
`))
	binary := ax.Join(projectDir, binaryName)
	requireBuildCmdOK(t, storage.Local.Write(binary, "ELFCONTENT"))
	return projectDir, binary
}

func TestBuildCmd_runBuildDebInDir_Good(t *testing.T) {
	projectDir, binary := newDebProject(t, "core")

	requireBuildCmdOK(t, runBuildDebInDir(context.Background(), projectDir, BuildDebRequest{
		Binary:       binary,
		Name:         "core",
		Version:      "v1.2.3",
		Architecture: "amd64",
		Maintainer:   "Lethean <dev@lethean.io>",
	}))

	requireBuildCmdOK(t, ax.Stat(ax.Join(projectDir, "dist", "core_1.2.3_amd64.deb")))
}

// The architecture comes from the asset name when not given, because a
// cross-compiled binary must not inherit the build host's.
func TestBuildCmd_runBuildDebInDir_InfersArch_Good(t *testing.T) {
	projectDir, binary := newDebProject(t, "core-linux-arm64")

	requireBuildCmdOK(t, runBuildDebInDir(context.Background(), projectDir, BuildDebRequest{
		Binary:  binary,
		Name:    "core",
		Version: "v1.2.3",
	}))

	requireBuildCmdOK(t, ax.Stat(ax.Join(projectDir, "dist", "core_1.2.3_arm64.deb")))
}

func TestBuildCmd_runBuildDebInDir_Bad(t *testing.T) {
	projectDir, binary := newDebProject(t, "core")

	// No binary at all.
	if runBuildDebInDir(context.Background(), projectDir, BuildDebRequest{Version: "v1.0.0", Architecture: "amd64"}).OK {
		t.Fatal("expected failure with no binary")
	}
	// Binary that is not there.
	if runBuildDebInDir(context.Background(), projectDir, BuildDebRequest{
		Binary: ax.Join(projectDir, "absent"), Version: "v1.0.0", Architecture: "amd64",
	}).OK {
		t.Fatal("expected failure for missing binary")
	}
	// Architecture neither given nor inferable: refuse rather than guess.
	if runBuildDebInDir(context.Background(), projectDir, BuildDebRequest{
		Binary: binary, Version: "v1.0.0",
	}).OK {
		t.Fatal("expected failure when architecture cannot be inferred")
	}
}

func TestBuildCmd_archFromBinaryName_Good(t *testing.T) {
	cases := map[string]string{
		"/dist/core-linux-amd64":     "amd64",
		"/dist/core_linux_arm64":     "arm64",
		"/dist/core-windows-386.exe": "386",
		"/dist/core-linux-ppc64le":   "ppc64le",
	}
	for path, want := range cases {
		if got := archFromBinaryName(path); got != want {
			t.Fatalf("archFromBinaryName(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestBuildCmd_archFromBinaryName_Bad(t *testing.T) {
	// A bare name carries no architecture, and guessing one would produce a
	// package that installs on the wrong machine.
	for _, path := range []string{"/dist/core", "/dist/core.exe", ""} {
		if got := archFromBinaryName(path); got != "" {
			t.Fatalf("archFromBinaryName(%q) = %q, want empty", path, got)
		}
	}
}

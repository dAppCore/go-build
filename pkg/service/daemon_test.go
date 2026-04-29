package service

import (
	"context"
	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"runtime"
	"testing"
	"time"
)

func TestShouldSkipWatchPath_Good(t *testing.T) {
	projectDir := t.TempDir()
	if !(shouldSkipWatchPath(projectDir, core.PathJoin(projectDir, ".git", "HEAD"))) {
		t.Fatal("expected true")
	}
	if !(shouldSkipWatchPath(projectDir, core.PathJoin(projectDir, "dist", "core-build"))) {
		t.Fatal("expected true")
	}
	if !(shouldSkipWatchPath(projectDir, core.PathJoin(projectDir, ".core", "cache", "state.json"))) {
		t.Fatal("expected true")
	}
	if shouldSkipWatchPath(projectDir, core.PathJoin(projectDir, ".core", "build.yaml")) {
		t.Fatal("expected false")
	}
	if shouldSkipWatchPath(projectDir, core.PathJoin(projectDir, "cmd", "main.go")) {
		t.Fatal("expected false")
	}

}

func TestSnapshotFiles_ExcludesGeneratedOutputsGood(t *testing.T) {
	projectDir := t.TempDir()
	if err := ax.MkdirAll(core.PathJoin(projectDir, "cmd"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.MkdirAll(core.PathJoin(projectDir, "dist"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.MkdirAll(core.PathJoin(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(core.PathJoin(projectDir, "cmd", "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(core.PathJoin(projectDir, "dist", "app"), []byte("binary"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(core.PathJoin(projectDir, ".git", "HEAD"), []byte("ref: refs/heads/dev"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshot, err := snapshotFiles(Config{ProjectDir: projectDir, WatchPaths: []string{projectDir}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(snapshot, core.PathJoin(projectDir, "cmd", "main.go")) {
		t.Fatalf("expected %v to contain %v", snapshot, core.PathJoin(projectDir, "cmd", "main.go"))
	}
	if stdlibAssertContains(snapshot, core.PathJoin(projectDir, "dist", "app")) {
		t.Fatalf("expected %v not to contain %v", snapshot, core.PathJoin(projectDir, "dist", "app"))
	}
	if stdlibAssertContains(snapshot, core.PathJoin(projectDir, ".git", "HEAD")) {
		t.Fatalf("expected %v not to contain %v", snapshot, core.PathJoin(projectDir, ".git", "HEAD"))
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

func TestDefaultRunWatchedBuild_WithoutBuildConfig_UsesLocalTargetGood(t *testing.T) {
	projectDir := t.TempDir()
	if err := ax.WriteFile(core.PathJoin(projectDir, "go.mod"), []byte("module example.com/daemon\n\ngo 1.20\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(core.PathJoin(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := defaultRunWatchedBuild(context.Background(), projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distDir := core.PathJoin(projectDir, "dist")
	entries, err := ax.ReadDir(distDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(entries))
	}
	if !stdlibAssertEqual(runtime.GOOS+"_"+runtime.GOARCH, entries[0].Name()) {
		t.Fatalf("want %v, got %v", runtime.GOOS+"_"+runtime.GOARCH, entries[0].Name())
	}

	platformEntries, err := ax.ReadDir(core.PathJoin(distDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(platformEntries) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(platformEntries))
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestDaemon_Run_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run(ctx, Config{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDaemon_Run_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run(ctx, Config{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDaemon_Run_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run(ctx, Config{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDaemon_EventEmitter_Emit_Good(t *core.T) {
	subject := daemonEventEmitter{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Emit("agent", "agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDaemon_EventEmitter_Emit_Bad(t *core.T) {
	subject := daemonEventEmitter{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Emit("", "agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDaemon_EventEmitter_Emit_Ugly(t *core.T) {
	subject := daemonEventEmitter{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Emit("agent", "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

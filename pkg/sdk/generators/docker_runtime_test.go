package generators

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"dappco.re/go/build/internal/ax"
)

const dockerRuntimeTestProbeTimeout = 10 * time.Second

func resetDockerRuntimeState() {
	resetDockerRuntimeAvailabilityCache()
	availabilityProbeTimeout = dockerRuntimeTestProbeTimeout
}

func setAvailabilityProbeTimeout(t *testing.T, timeout time.Duration) {
	t.Helper()

	previous := availabilityProbeTimeout
	availabilityProbeTimeout = timeout
	t.Cleanup(func() {
		availabilityProbeTimeout = previous
	})
}

func writeFakeDockerRuntime(t *testing.T, dir, script string) string {
	t.Helper()

	dockerPath := ax.Join(dir, "docker")
	if err := ax.WriteFile(dockerPath, []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return dockerPath
}

func TestSDK_ResolveDockerRuntimeCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "docker")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := resolveDockerRuntimeCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestSDK_ResolveDockerRuntimeCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := resolveDockerRuntimeCli(ax.Join(t.TempDir(), "missing-docker"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "docker CLI not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "docker CLI not found")
	}

}

func TestSDK_GeneratorAvailabilityUsesDockerFallback_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")
	}
	if !(NewGoGenerator().Available()) {
		t.Fatal("expected true")
	}
	if !(NewPythonGenerator().Available()) {
		t.Fatal("expected true")
	}
	if !(NewTypeScriptGenerator().Available()) {
		t.Fatal("expected true")
	}
	if !(NewPHPGenerator().Available()) {
		t.Fatal("expected true")
	}

}

func TestSDK_DockerRuntimeAvailabilityCachesSuccessfulProbe_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	countFile := ax.Join(dockerDir, "count.txt")
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  echo probe >> '"+countFile+"'\n  exit 0\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")
	}
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")
	}

	content, err := ax.ReadFile(countFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(strings.Fields(string(content))) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(strings.Fields(string(content))))
	}

}

func TestSDK_DockerRuntimeAvailabilityCachesFailedProbe_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	countFile := ax.Join(dockerDir, "count.txt")
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  echo probe >> '"+countFile+"'\n  exit 1\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}

	content, err := ax.ReadFile(countFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(strings.Fields(string(content))) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(strings.Fields(string(content))))
	}

}

func TestSDK_DockerRuntimeAvailabilityRespectsCancelledContext_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if dockerRuntimeAvailableWithContext(ctx) {
		t.Fatal("expected false")
	}
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")
	}

}

func TestSDK_DockerRuntimeAvailabilityRespectsCancelledContextAfterCachedSuccess_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", dockerDir)
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if dockerRuntimeAvailableWithContext(ctx) {
		t.Fatal("expected false")
	}

}

func TestSDK_DockerRuntimeAvailabilityUsesProbeTimeout_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)
	setAvailabilityProbeTimeout(t, 20*time.Millisecond)

	dockerDir := t.TempDir()
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  while :; do :; done\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	started := time.Now()
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}
	if time.Since(started) >= 500*time.Millisecond {
		t.Fatalf("expected %v to be less than %v", time.Since(started), 500*time.Millisecond)
	}

}

func TestSDK_DockerRuntimeAvailabilityRechecksAfterFailure_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	dockerPath := writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 1\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}
	if err := ax.WriteFile(dockerPath, []byte("#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")
	}

}

func TestSDK_DockerRuntimeAvailabilityInvalidatesCachedSuccessWhenCommandChanges_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	successDir := t.TempDir()
	writeFakeDockerRuntime(t, successDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n")
	t.Setenv("PATH", successDir)
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")
	}

	failureDir := t.TempDir()
	writeFakeDockerRuntime(t, failureDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 1\nfi\nexit 0\n")
	t.Setenv("PATH", failureDir)
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}

}

func TestSDK_DockerRuntimeAvailabilityInvalidatesCachedSuccessWhenCommandMutatesInPlace_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	dockerPath := writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")

		// Preserve monotonic ordering for filesystems with coarse mtimes.
	}

	time.Sleep(20 * time.Millisecond)
	if err := ax.WriteFile(dockerPath, []byte("#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 1\nfi\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}

}

func TestSDK_DockerRuntimeAvailabilityInvalidatesCachedSuccessWhenCommandKeepsSizeAndMTime_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	successScript := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n"
	failureScript := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 1\nfi\nexit 0\n"
	dockerPath := writeFakeDockerRuntime(t, dockerDir, successScript)
	t.Setenv("PATH", dockerDir)
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")
	}

	info, err := os.Stat(dockerPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(dockerPath, []byte(failureScript), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.Chtimes(dockerPath, info.ModTime(), info.ModTime()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}

}

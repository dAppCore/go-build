package generators

import (
	"context"
	"io/fs"
	"testing"
	"time"

	core "dappco.re/go"
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
	if result := ax.WriteFile(dockerPath, []byte(script), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	return dockerPath
}

func TestSDK_ResolveDockerRuntimeCliGood(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "docker")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	commandResult := resolveDockerRuntimeCli(fallbackPath)
	if !commandResult.OK {
		t.Fatalf("unexpected error: %v", commandResult.Error())
	}
	command := commandResult.Value.(string)
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestSDK_ResolveDockerRuntimeCliBad(t *testing.T) {
	t.Setenv("PATH", "")
	result := resolveDockerRuntimeCli(ax.Join(t.TempDir(), "missing-docker"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "docker CLI not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "docker CLI not found")
	}

}

func TestSDK_GeneratorAvailabilityUsesDockerFallbackGood(t *testing.T) {
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

func TestSDK_DockerRuntimeAvailabilityCachesSuccessfulProbeGood(t *testing.T) {
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

	contentResult := ax.ReadFile(countFile)
	if !contentResult.OK {
		t.Fatalf("unexpected error: %v", contentResult.Error())
	}
	content := contentResult.Value.([]byte)
	if len(dockerRuntimeFields(string(content))) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(dockerRuntimeFields(string(content))))
	}

}

func TestSDK_DockerRuntimeAvailabilityCachesFailedProbeBad(t *testing.T) {
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

	contentResult := ax.ReadFile(countFile)
	if !contentResult.OK {
		t.Fatalf("unexpected error: %v", contentResult.Error())
	}
	content := contentResult.Value.([]byte)
	if len(dockerRuntimeFields(string(content))) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(dockerRuntimeFields(string(content))))
	}

}

func dockerRuntimeFields(value string) []string {
	value = core.Trim(value)
	var fields []string
	start := -1
	for i, r := range value {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if start >= 0 {
				fields = append(fields, value[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		fields = append(fields, value[start:])
	}
	return fields
}

func TestSDK_DockerRuntimeAvailabilityRespectsCancelledContextBad(t *testing.T) {
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

func TestSDK_DockerRuntimeAvailabilityRespectsCancelledContextAfterCachedSuccessBad(t *testing.T) {
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

func TestSDK_DockerRuntimeAvailabilityUsesProbeTimeoutBad(t *testing.T) {
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

func TestSDK_DockerRuntimeAvailabilityRechecksAfterFailureGood(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	dockerPath := writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 1\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}
	if result := ax.WriteFile(dockerPath, []byte("#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if !(dockerRuntimeAvailable()) {
		t.Fatal("expected true")
	}

}

func TestSDK_DockerRuntimeAvailabilityInvalidatesCachedSuccessWhenCommandChangesGood(t *testing.T) {
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

func TestSDK_DockerRuntimeAvailabilityInvalidatesCachedSuccessWhenCommandMutatesInPlaceGood(t *testing.T) {
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
	if result := ax.WriteFile(dockerPath, []byte("#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 1\nfi\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}

}

func TestSDK_DockerRuntimeAvailabilityInvalidatesCachedSuccessWhenCommandKeepsSizeAndMTimeGood(t *testing.T) {
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

	infoResult := ax.Stat(dockerPath)
	if !infoResult.OK {
		t.Fatalf("unexpected error: %v", infoResult.Error())
	}
	info := infoResult.Value.(fs.FileInfo)
	if result := ax.WriteFile(dockerPath, []byte(failureScript), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.Chtimes(dockerPath, info.ModTime(), info.ModTime()); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if dockerRuntimeAvailable() {
		t.Fatal("expected false")
	}

}

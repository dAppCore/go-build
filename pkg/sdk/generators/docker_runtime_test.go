package generators

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"dappco.re/go/build/internal/ax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetDockerRuntimeState() {
	dockerRuntimeMu = sync.Mutex{}
	dockerRuntimeChecked = false
	dockerRuntimeOK = false
	dockerRuntimeCommand = ""
	dockerRuntimeState = ""
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
	require.NoError(t, ax.WriteFile(dockerPath, []byte(script), 0o755))
	return dockerPath
}

func TestSDK_ResolveDockerRuntimeCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "docker")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := resolveDockerRuntimeCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestSDK_ResolveDockerRuntimeCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := resolveDockerRuntimeCli(ax.Join(t.TempDir(), "missing-docker"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker CLI not found")
}

func TestSDK_GeneratorAvailabilityUsesDockerFallback_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	assert.True(t, dockerRuntimeAvailable())
	assert.True(t, NewGoGenerator().Available())
	assert.True(t, NewPythonGenerator().Available())
	assert.True(t, NewTypeScriptGenerator().Available())
	assert.True(t, NewPHPGenerator().Available())
}

func TestSDK_DockerRuntimeAvailabilityCachesSuccessfulProbe_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	countFile := ax.Join(dockerDir, "count.txt")
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  echo probe >> '"+countFile+"'\n  exit 0\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	assert.True(t, dockerRuntimeAvailable())
	assert.True(t, dockerRuntimeAvailable())

	content, err := ax.ReadFile(countFile)
	require.NoError(t, err)
	assert.Len(t, strings.Fields(string(content)), 1)
}

func TestSDK_DockerRuntimeAvailabilityCachesFailedProbe_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	countFile := ax.Join(dockerDir, "count.txt")
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  echo probe >> '"+countFile+"'\n  exit 1\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	assert.False(t, dockerRuntimeAvailable())
	assert.False(t, dockerRuntimeAvailable())

	content, err := ax.ReadFile(countFile)
	require.NoError(t, err)
	assert.Len(t, strings.Fields(string(content)), 1)
}

func TestSDK_DockerRuntimeAvailabilityRespectsCancelledContext_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.False(t, dockerRuntimeAvailableWithContext(ctx))
	assert.True(t, dockerRuntimeAvailable())
}

func TestSDK_DockerRuntimeAvailabilityRespectsCancelledContextAfterCachedSuccess_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	assert.True(t, dockerRuntimeAvailable())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.False(t, dockerRuntimeAvailableWithContext(ctx))
}

func TestSDK_DockerRuntimeAvailabilityUsesProbeTimeout_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)
	setAvailabilityProbeTimeout(t, 20*time.Millisecond)

	dockerDir := t.TempDir()
	writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  while :; do :; done\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	started := time.Now()
	assert.False(t, dockerRuntimeAvailable())
	assert.Less(t, time.Since(started), 500*time.Millisecond)
}

func TestSDK_DockerRuntimeAvailabilityRechecksAfterFailure_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	dockerPath := writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 1\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	assert.False(t, dockerRuntimeAvailable())

	require.NoError(t, ax.WriteFile(dockerPath, []byte("#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n"), 0o755))
	assert.True(t, dockerRuntimeAvailable())
}

func TestSDK_DockerRuntimeAvailabilityInvalidatesCachedSuccessWhenCommandChanges_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	successDir := t.TempDir()
	writeFakeDockerRuntime(t, successDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n")
	t.Setenv("PATH", successDir)

	assert.True(t, dockerRuntimeAvailable())

	failureDir := t.TempDir()
	writeFakeDockerRuntime(t, failureDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 1\nfi\nexit 0\n")
	t.Setenv("PATH", failureDir)

	assert.False(t, dockerRuntimeAvailable())
}

func TestSDK_DockerRuntimeAvailabilityInvalidatesCachedSuccessWhenCommandMutatesInPlace_Good(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	dockerPath := writeFakeDockerRuntime(t, dockerDir, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 0\nfi\nexit 0\n")
	t.Setenv("PATH", dockerDir)

	assert.True(t, dockerRuntimeAvailable())

	// Preserve monotonic ordering for filesystems with coarse mtimes.
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, ax.WriteFile(dockerPath, []byte("#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then\n  exit 1\nfi\nexit 0\n"), 0o755))

	assert.False(t, dockerRuntimeAvailable())
}

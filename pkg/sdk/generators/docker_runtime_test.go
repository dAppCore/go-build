package generators

import (
	"context"
	"sync"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetDockerRuntimeState() {
	dockerRuntimeMu = sync.Mutex{}
	dockerRuntimeChecked = false
	dockerRuntimeOK = false
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
	dockerPath := ax.Join(dockerDir, "docker")
	require.NoError(t, ax.WriteFile(dockerPath, []byte("#!/bin/sh\nif [ \"$1\" = \"info\" ]; then\n  exit 0\nfi\nexit 0\n"), 0o755))
	t.Setenv("PATH", dockerDir)

	assert.True(t, dockerRuntimeAvailable())
	assert.True(t, NewGoGenerator().Available())
	assert.True(t, NewPythonGenerator().Available())
	assert.True(t, NewTypeScriptGenerator().Available())
	assert.True(t, NewPHPGenerator().Available())
}

func TestSDK_DockerRuntimeAvailabilityRespectsCancelledContext_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	dockerPath := ax.Join(dockerDir, "docker")
	require.NoError(t, ax.WriteFile(dockerPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", dockerDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.False(t, dockerRuntimeAvailableWithContext(ctx))
	assert.True(t, dockerRuntimeAvailable())
}

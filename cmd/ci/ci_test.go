package ci

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/release"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCI_runCIReleaseInitInDir_Good(t *testing.T) {
	projectDir := t.TempDir()

	err := runCIReleaseInitInDir(projectDir)
	require.NoError(t, err)

	configPath := release.ConfigPath(projectDir)
	content, err := ax.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "sdk:")
	assert.Contains(t, string(content), "spec: api/openapi.yaml")
	assert.Contains(t, string(content), "languages:")
	assert.Contains(t, string(content), "- typescript")
}

package sdk

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_DetectSpecConfigPath_Good(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "api", "spec.yaml")
	err := ax.MkdirAll(ax.Dir(specPath), 0755)
	require.NoError(t, err)
	err = ax.WriteFile(specPath, []byte("openapi: 3.0.0"), 0644)
	require.NoError(t, err)

	sdk := New(tmpDir, &Config{Spec: "api/spec.yaml"})
	got, err := sdk.DetectSpec()
	assert.NoError(t, err)
	assert.Equal(t, specPath, got)
}

func TestDetect_DetectSpecCommonPath_Good(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "openapi.yaml")
	err := ax.WriteFile(specPath, []byte("openapi: 3.0.0"), 0644)
	require.NoError(t, err)

	sdk := New(tmpDir, nil)
	got, err := sdk.DetectSpec()
	assert.NoError(t, err)
	assert.Equal(t, specPath, got)
}

func TestDetect_DetectSpecNotFound_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	sdk := New(tmpDir, nil)
	_, err := sdk.DetectSpec()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no OpenAPI spec found")
}

func TestDetect_DetectSpecConfigNotFound_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	sdk := New(tmpDir, &Config{Spec: "non-existent.yaml"})
	_, err := sdk.DetectSpec()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configured spec not found")
}

func TestDetect_ContainsScramble_Good(t *testing.T) {
	tests := []struct {
		data     string
		expected bool
	}{
		{`{"require": {"dedoc/scramble": "^0.1"}}`, true},
		{`{"require": {"scramble": "^0.1"}}`, true},
		{`{"require": {"laravel/framework": "^11.0"}}`, false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, containsScramble(tt.data))
	}
}

func TestDetect_DetectScramble_Bad(t *testing.T) {
	t.Run("no composer.json", func(t *testing.T) {
		sdk := New(t.TempDir(), nil)
		_, err := sdk.detectScramble()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no composer.json")
	})

	t.Run("no scramble in composer.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := ax.WriteFile(ax.Join(tmpDir, "composer.json"), []byte(`{}`), 0644)
		require.NoError(t, err)

		sdk := New(tmpDir, nil)
		_, err = sdk.detectScramble()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scramble not found")
	})
}

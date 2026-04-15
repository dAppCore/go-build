package signing

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGPG_GPGSignerName_Good(t *testing.T) {
	s := NewGPGSigner("ABCD1234")
	assert.Equal(t, "gpg", s.Name())
}

func TestGPG_GPGSignerAvailable_Good(t *testing.T) {
	s := NewGPGSigner("ABCD1234")
	_ = s.Available()
}

func TestGPG_GPGSignerNoKey_Bad(t *testing.T) {
	s := NewGPGSigner("")
	assert.False(t, s.Available())
}

func TestGPG_GPGSignerSign_Bad(t *testing.T) {
	fs := io.Local
	t.Run("fails when no key", func(t *testing.T) {
		s := NewGPGSigner("")
		err := s.Sign(context.Background(), fs, "test.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not available or key not configured")
	})
}

func TestGPG_ResolveGpgCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "gpg")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := resolveGpgCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestGPG_ResolveGpgCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveGpgCli(ax.Join(t.TempDir(), "missing-gpg"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gpg CLI not found")
}

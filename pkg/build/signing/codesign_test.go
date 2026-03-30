package signing

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodesign_MacOSSignerName_Good(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{Identity: "Developer ID Application: Test"})
	assert.Equal(t, "codesign", s.Name())
}

func TestCodesign_MacOSSignerAvailable_Good(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{Identity: "Developer ID Application: Test"})

	if runtime.GOOS == "darwin" {
		// Just verify it doesn't panic
		_ = s.Available()
	} else {
		assert.False(t, s.Available())
	}
}

func TestCodesign_MacOSSignerNoIdentity_Bad(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{})
	assert.False(t, s.Available())
}

func TestCodesign_MacOSSignerSign_Bad(t *testing.T) {
	t.Run("fails when not available", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			t.Skip("skipping on macOS")
		}
		fs := io.Local
		s := NewMacOSSigner(MacOSConfig{Identity: "test"})
		err := s.Sign(context.Background(), fs, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only available on macOS")
	})
}

func TestCodesign_MacOSSignerNotarize_Bad(t *testing.T) {
	fs := io.Local
	t.Run("fails with missing credentials", func(t *testing.T) {
		s := NewMacOSSigner(MacOSConfig{})
		err := s.Notarize(context.Background(), fs, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing Apple credentials")
	})
}

func TestCodesign_MacOSSignerShouldNotarize_Good(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{Notarize: true})
	assert.True(t, s.ShouldNotarize())

	s2 := NewMacOSSigner(MacOSConfig{Notarize: false})
	assert.False(t, s2.ShouldNotarize())
}

func TestCodesign_ResolveCodesignCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "codesign")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := resolveCodesignCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestCodesign_ResolveCodesignCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveCodesignCli(ax.Join(t.TempDir(), "missing-codesign"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "codesign tool not found")
}

func TestCodesign_ResolveZipCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "zip")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := resolveZipCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestCodesign_ResolveZipCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveZipCli(ax.Join(t.TempDir(), "missing-zip"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zip tool not found")
}

func TestCodesign_ResolveXcrunCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "xcrun")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := resolveXcrunCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestCodesign_ResolveXcrunCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveXcrunCli(ax.Join(t.TempDir(), "missing-xcrun"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "xcrun tool not found")
}

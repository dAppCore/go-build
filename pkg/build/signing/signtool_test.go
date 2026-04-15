package signing

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSigntool_NewWindowsSigner_Good(t *testing.T) {
	signer := NewWindowsSigner(WindowsConfig{
		Signtool:    true,
		Certificate: "cert.pfx",
		Password:    "secret",
	})

	assert.Equal(t, "signtool", signer.Name())
}

func TestSigntool_NewWindowsSigner_Bad(t *testing.T) {
	t.Run("available is false when the explicit toggle disables signtool", func(t *testing.T) {
		signer := NewWindowsSigner(WindowsConfig{
			Signtool:         false,
			Certificate:      "cert.pfx",
			signtoolExplicit: true,
		})

		assert.False(t, signer.Available())
	})
}

func TestSigntool_NewWindowsSigner_Ugly(t *testing.T) {
	t.Run("available is false without a certificate", func(t *testing.T) {
		signer := NewWindowsSigner(WindowsConfig{Signtool: true})
		assert.False(t, signer.Available())
	})
}

func TestSigntool_Available_Good(t *testing.T) {
	t.Skip("missing seam: runtime.GOOS and installed signtool are not injectable in unit tests; Windows success path requires a Windows host seam")
}

func TestSigntool_Sign_Bad(t *testing.T) {
	t.Run("returns the platform guard on non-Windows hosts", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("this assertion is specific to non-Windows hosts")
		}

		signer := NewWindowsSigner(WindowsConfig{
			Signtool:    true,
			Certificate: "cert.pfx",
		})

		err := signer.Sign(context.Background(), io.Local, "test.exe")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only available on Windows")
	})
}

func TestSigntool_Sign_Good(t *testing.T) {
	t.Skip("missing seam: signtool success requires a Windows host with signtool.exe available on PATH")
}

func TestSigntool_ResolveSigntoolCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "signtool.exe")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := resolveSigntoolCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestSigntool_ResolveSigntoolCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveSigntoolCli(ax.Join(t.TempDir(), "missing-signtool.exe"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signtool tool not found")
}

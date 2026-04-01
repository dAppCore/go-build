package buildcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPwa_FindManifestURL_Good(t *testing.T) {
	t.Run("accepts a standard manifest link", func(t *testing.T) {
		htmlContent := `<html><head><link rel="manifest" href="/manifest.json"></head></html>`

		got, err := findManifestURL(htmlContent, "https://example.test/app/")
		require.NoError(t, err)
		assert.Equal(t, "https://example.test/manifest.json", got)
	})

	t.Run("accepts case-insensitive tokenised rel values", func(t *testing.T) {
		htmlContent := `<html><head><link rel="Manifest icon" href="manifest.json"></head></html>`

		got, err := findManifestURL(htmlContent, "https://example.test/app/")
		require.NoError(t, err)
		assert.Equal(t, "https://example.test/app/manifest.json", got)
	})
}

func TestPwa_FindManifestURL_Bad(t *testing.T) {
	t.Run("returns an error when no manifest link exists", func(t *testing.T) {
		htmlContent := `<html><head><link rel="icon" href="/icon.png"></head></html>`

		got, err := findManifestURL(htmlContent, "https://example.test/app/")
		assert.Error(t, err)
		assert.Empty(t, got)
		assert.Contains(t, err.Error(), "pwa.findManifestURL")
	})
}

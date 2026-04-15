package buildcmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"dappco.re/go/core/build/internal/ax"
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

func TestPwa_ExtractHTMLMetadataAndAssets_Good(t *testing.T) {
	htmlContent := `
<!doctype html>
<html>
  <head>
    <title>Example App</title>
    <meta name="description" content="Example description">
    <link rel="manifest" href="/manifest.json">
    <link rel="stylesheet" href="/assets/app.css">
    <link rel="icon" href="/assets/icon.png">
    <script src="/assets/app.js"></script>
  </head>
  <body>
    <img src="/assets/logo.png" srcset="/assets/logo.png 1x, /assets/logo@2x.png 2x">
  </body>
</html>`

	metadata, assets, err := extractHTMLMetadataAndAssets(htmlContent, "https://example.test/app/")
	require.NoError(t, err)

	assert.Equal(t, "Example App", metadata.DisplayName)
	assert.Equal(t, "Example description", metadata.Description)
	assert.Equal(t, "https://example.test/manifest.json", metadata.ManifestURL)
	assert.Equal(t, []string{"https://example.test/assets/icon.png"}, metadata.Icons)
	assert.ElementsMatch(t, []string{
		"https://example.test/manifest.json",
		"https://example.test/assets/app.css",
		"https://example.test/assets/icon.png",
		"https://example.test/assets/app.js",
		"https://example.test/assets/logo.png",
		"https://example.test/assets/logo@2x.png",
	}, assets)
}

func TestPwa_DownloadPWA_DownloadsHTMLAndManifestAssets_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app":
			_, _ = w.Write([]byte(`<!doctype html>
<html>
  <head>
    <title>Example App</title>
    <meta name="description" content="Example description">
    <link rel="manifest" href="/manifest.json">
    <link rel="stylesheet" href="/assets/app.css">
    <script src="/assets/app.js"></script>
  </head>
  <body>
    <img src="/assets/logo.png">
  </body>
</html>`))
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/manifest+json")
			_, _ = w.Write([]byte(`{
  "name": "Manifest App",
  "description": "Manifest description",
  "start_url": "/launch.html",
  "icons": [
    {"src": "/assets/icon-192.png"}
  ]
}`))
		case "/assets/app.css":
			_, _ = w.Write([]byte("body { color: red; }"))
		case "/assets/app.js":
			_, _ = w.Write([]byte("console.log('app');"))
		case "/assets/logo.png":
			_, _ = w.Write([]byte("logo"))
		case "/assets/icon-192.png":
			_, _ = w.Write([]byte("icon"))
		case "/launch.html":
			_, _ = w.Write([]byte("<html><body>launch</body></html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	destDir := t.TempDir()
	require.NoError(t, downloadPWA(context.Background(), server.URL+"/app", destDir))

	indexBody, err := ax.ReadFile(ax.Join(destDir, "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(indexBody), "<title>Example App</title>")

	manifestBody, err := ax.ReadFile(ax.Join(destDir, "manifest.json"))
	require.NoError(t, err)
	assert.Contains(t, string(manifestBody), `"name": "Manifest App"`)

	for _, relPath := range []string{
		"assets/app.css",
		"assets/app.js",
		"assets/logo.png",
		"assets/icon-192.png",
		"launch.html",
	} {
		assert.True(t, ax.IsFile(ax.Join(destDir, relPath)), relPath)
	}
}

func TestPwa_ResolvePWAAppConfig_UsesLocalMetadata_Good(t *testing.T) {
	projectDir := t.TempDir()

	require.NoError(t, ax.WriteString(ax.Join(projectDir, "index.html"), `<!doctype html>
<html>
  <head>
    <title>Fallback Title</title>
    <meta name="description" content="HTML description">
    <link rel="manifest" href="/manifest.json">
  </head>
</html>`, 0o644))
	require.NoError(t, ax.WriteString(ax.Join(projectDir, "manifest.json"), `{
  "name": "Manifest App",
  "description": "Manifest description",
  "icons": [{"src": "/icon.png"}]
}`, 0o644))

	cfg := resolvePWAAppConfig(projectDir)
	assert.Equal(t, "manifest-app", cfg.ModuleName)
	assert.Equal(t, "Manifest App", cfg.DisplayName)
	assert.Equal(t, "Manifest description", cfg.Description)
}

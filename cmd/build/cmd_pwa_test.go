package buildcmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"dappco.re/go/build/internal/ax"
)

func TestPwa_FindManifestURL_Good(t *testing.T) {
	t.Run("accepts a standard manifest link", func(t *testing.T) {
		htmlContent := `<html><head><link rel="manifest" href="/manifest.json"></head></html>`

		got, err := findManifestURL(htmlContent, "https://example.test/app/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("https://example.test/manifest.json", got) {
			t.Fatalf("want %v, got %v", "https://example.test/manifest.json", got)
		}

	})

	t.Run("accepts case-insensitive tokenised rel values", func(t *testing.T) {
		htmlContent := `<html><head><link rel="Manifest icon" href="manifest.json"></head></html>`

		got, err := findManifestURL(htmlContent, "https://example.test/app/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("https://example.test/app/manifest.json", got) {
			t.Fatalf("want %v, got %v", "https://example.test/app/manifest.json", got)
		}

	})
}

func TestPwa_FindManifestURL_Bad(t *testing.T) {
	t.Run("returns an error when no manifest link exists", func(t *testing.T) {
		htmlContent := `<html><head><link rel="icon" href="/icon.png"></head></html>`

		got, err := findManifestURL(htmlContent, "https://example.test/app/")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(got) {
			t.Fatalf("expected empty, got %v", got)
		}
		if !stdlibAssertContains(err.Error(), "pwa.findManifestURL") {
			t.Fatalf("expected %v to contain %v", err.Error(), "pwa.findManifestURL")
		}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("Example App", metadata.DisplayName) {
		t.Fatalf("want %v, got %v", "Example App", metadata.DisplayName)
	}
	if !stdlibAssertEqual("Example description", metadata.Description) {
		t.Fatalf("want %v, got %v", "Example description", metadata.Description)
	}
	if !stdlibAssertEqual("https://example.test/manifest.json", metadata.ManifestURL) {
		t.Fatalf("want %v, got %v", "https://example.test/manifest.json", metadata.ManifestURL)
	}
	if !stdlibAssertEqual([]string{"https://example.test/assets/icon.png"}, metadata.Icons) {
		t.Fatalf("want %v, got %v", []string{"https://example.test/assets/icon.png"}, metadata.Icons)
	}
	if !stdlibAssertElementsMatch([]string{"https://example.test/manifest.json", "https://example.test/assets/app.css", "https://example.test/assets/icon.png", "https://example.test/assets/app.js", "https://example.test/assets/logo.png", "https://example.test/assets/logo@2x.png"}, assets) {
		t.Fatalf("expected elements %v, got %v", []string{"https://example.test/manifest.json", "https://example.test/assets/app.css", "https://example.test/assets/icon.png", "https://example.test/assets/app.js", "https://example.test/assets/logo.png", "https://example.test/assets/logo@2x.png"}, assets)
	}

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
	if err := downloadPWA(context.Background(), server.URL+"/app", destDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	indexBody, err := ax.ReadFile(ax.Join(destDir, "index.html"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(indexBody), "<title>Example App</title>") {
		t.Fatalf("expected %v to contain %v", string(indexBody), "<title>Example App</title>")
	}

	manifestBody, err := ax.ReadFile(ax.Join(destDir, "manifest.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(manifestBody), `"name": "Manifest App"`) {
		t.Fatalf("expected %v to contain %v", string(manifestBody), `"name": "Manifest App"`)
	}

	for _, relPath := range []string{
		"assets/app.css",
		"assets/app.js",
		"assets/logo.png",
		"assets/icon-192.png",
		"launch.html",
	} {
		if !(ax.IsFile(ax.Join(destDir, relPath))) {
			t.Fatal(relPath)
		}

	}
}

func TestPwa_ResolvePWAAppConfig_UsesLocalMetadata_Good(t *testing.T) {
	projectDir := t.TempDir()
	if err := ax.WriteString(ax.Join(projectDir, "index.html"), `<!doctype html>
<html>
  <head>
    <title>Fallback Title</title>
    <meta name="description" content="HTML description">
    <link rel="manifest" href="/manifest.json">
  </head>
</html>`, 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteString(ax.Join(projectDir, "manifest.json"), `{
  "name": "Manifest App",
  "description": "Manifest description",
  "icons": [{"src": "/icon.png"}]
}`, 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := resolvePWAAppConfig(projectDir)
	if !stdlibAssertEqual("manifest-app", cfg.ModuleName) {
		t.Fatalf("want %v, got %v", "manifest-app", cfg.ModuleName)
	}
	if !stdlibAssertEqual("Manifest App", cfg.DisplayName) {
		t.Fatalf("want %v, got %v", "Manifest App", cfg.DisplayName)
	}
	if !stdlibAssertEqual("Manifest description", cfg.Description) {
		t.Fatalf("want %v, got %v", "Manifest description", cfg.Description)
	}

}

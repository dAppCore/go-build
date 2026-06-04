package buildcmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

// --- joinPWAURLPath (cmd_pwa.go) ---

func TestPwa_joinPWAURLPath_Good(t *core.T) {
	core.AssertEqual(t, "a/b/c", joinPWAURLPath("a", "b", "c"))
}

func TestPwa_joinPWAURLPath_Bad(t *core.T) {
	// No parts joins to an empty string which cleans to "." (current dir),
	// not a usable URL path — the degenerate case callers must avoid.
	core.AssertEqual(t, ".", joinPWAURLPath())
}

func TestPwa_joinPWAURLPath_Ugly(t *core.T) {
	// Edge case: leading/trailing slashes in the parts are normalised away.
	core.AssertEqual(t, "/a/b", joinPWAURLPath("/a/", "/b/"))
}

// --- copyDir (cmd_pwa.go) ---

func TestPwa_copyDir_Good(t *core.T) {
	src := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(src, "a.txt"), []byte("A"), 0o644))
	requireBuildCmdOK(t, ax.MkdirAll(ax.Join(src, "sub"), 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(src, "sub", "b.txt"), []byte("B"), 0o644))

	dst := ax.Join(t.TempDir(), "out")
	result := copyDir(src, dst)
	core.AssertTrue(t, result.OK)
	// Files and nested directories are copied recursively.
	core.AssertTrue(t, ax.Exists(ax.Join(dst, "a.txt")))
	core.AssertTrue(t, ax.Exists(ax.Join(dst, "sub", "b.txt")))
	copied := requireBuildCmdBytes(t, ax.ReadFile(ax.Join(dst, "sub", "b.txt")))
	core.AssertEqual(t, "B", string(copied))
}

func TestPwa_copyDir_Bad(t *core.T) {
	// A non-existent source directory fails when its entries cannot be read.
	result := copyDir(ax.Join(t.TempDir(), "does-not-exist"), ax.Join(t.TempDir(), "out"))
	core.AssertFalse(t, result.OK)
}

func TestPwa_copyDir_Ugly(t *core.T) {
	// Edge case: an empty source directory copies to an empty destination,
	// creating the destination directory.
	src := t.TempDir()
	dst := ax.Join(t.TempDir(), "empty-out")
	result := copyDir(src, dst)
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, ax.IsDir(dst))
}

// --- runBuild / runPwaBuild error paths (cmd_pwa.go) ---
//
// The success paths shell `go mod tidy` and `go build` after template
// extraction (and, for runPwaBuild, a network download). Those external/network
// steps are not exercised here; the deterministic validation/error branches are.

func TestPwa_runBuild_Bad(t *core.T) {
	captureBuildStdout(t)
	// A path that is not a directory is rejected before any compilation.
	result := runBuild(context.Background(), ax.Join(t.TempDir(), "not-a-directory"))
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "must be a directory")
}

func TestPwa_runPwaBuild_Bad(t *core.T) {
	captureBuildStdout(t)
	// An unreachable/invalid URL fails the download step before any build.
	result := runPwaBuild(context.Background(), "http://127.0.0.1:1/does-not-exist")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "failed to download PWA")
}

func TestPwa_FindManifestURLGood(t *testing.T) {
	t.Run("accepts a standard manifest link", func(t *testing.T) {
		htmlContent := `<html><head><link rel="manifest" href="/manifest.json"></head></html>`

		got := requireBuildCmdString(t, findManifestURL(htmlContent, "https://example.test/app/"))
		if !stdlibAssertEqual("https://example.test/manifest.json", got) {
			t.Fatalf("want %v, got %v", "https://example.test/manifest.json", got)
		}

	})

	t.Run("accepts case-insensitive tokenised rel values", func(t *testing.T) {
		htmlContent := `<html><head><link rel="Manifest icon" href="manifest.json"></head></html>`

		got := requireBuildCmdString(t, findManifestURL(htmlContent, "https://example.test/app/"))
		if !stdlibAssertEqual("https://example.test/app/manifest.json", got) {
			t.Fatalf("want %v, got %v", "https://example.test/app/manifest.json", got)
		}

	})
}

func TestPwa_FindManifestURLBad(t *testing.T) {
	t.Run("returns an error when no manifest link exists", func(t *testing.T) {
		htmlContent := `<html><head><link rel="icon" href="/icon.png"></head></html>`

		result := findManifestURL(htmlContent, "https://example.test/app/")
		message := requireBuildCmdError(t, result)
		got, _ := result.Value.(string)
		if !stdlibAssertEmpty(got) {
			t.Fatalf("expected empty, got %v", got)
		}
		if !stdlibAssertContains(message, "pwa.findManifestURL") {
			t.Fatalf("expected %v to contain %v", message, "pwa.findManifestURL")
		}

	})
}

func TestPwa_ExtractHTMLMetadataAndAssetsGood(t *testing.T) {
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

	extracted := requireBuildCmdPWAExtraction(t, extractHTMLMetadataAndAssets(htmlContent, "https://example.test/app/"))
	metadata := extracted.Metadata
	assets := extracted.Assets
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

func TestPwa_DownloadPWA_DownloadsHTMLAndManifestAssetsGood(t *testing.T) {
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
	requireBuildCmdOK(t, downloadPWA(context.Background(), server.URL+"/app", destDir))

	indexBody := requireBuildCmdBytes(t, ax.ReadFile(ax.Join(destDir, "index.html")))
	if !stdlibAssertContains(string(indexBody), "<title>Example App</title>") {
		t.Fatalf("expected %v to contain %v", string(indexBody), "<title>Example App</title>")
	}

	manifestBody := requireBuildCmdBytes(t, ax.ReadFile(ax.Join(destDir, "manifest.json")))
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

func TestPwa_ResolvePWAAppConfig_UsesLocalMetadataGood(t *testing.T) {
	projectDir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteString(ax.Join(projectDir, "index.html"), `<!doctype html>
<html>
  <head>
    <title>Fallback Title</title>
    <meta name="description" content="HTML description">
    <link rel="manifest" href="/manifest.json">
  </head>
</html>`, 0o644))
	requireBuildCmdOK(t, ax.WriteString(ax.Join(projectDir, "manifest.json"), `{
  "name": "Manifest App",
  "description": "Manifest description",
  "icons": [{"src": "/icon.png"}]
}`, 0o644))

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

// --- resolvePWAAppConfig fallbacks (cmd_pwa.go) ---

func TestPwa_resolvePWAAppConfig_Good(t *core.T) {
	// A directory with no index.html falls back to the directory base name for
	// the display name and a slugified module name.
	dir := ax.Join(t.TempDir(), "MyCoolApp")
	requireBuildCmdOK(t, ax.MkdirAll(dir, 0o755))

	cfg := resolvePWAAppConfig(dir)
	core.AssertEqual(t, "MyCoolApp", cfg.DisplayName)
	core.AssertEqual(t, "mycoolapp", cfg.ModuleName)
	core.AssertNotEmpty(t, cfg.Description)
}

func TestPwa_resolvePWAAppConfig_Bad(t *core.T) {
	// A temp-build directory name is masked to a generic "PWA App" rather than
	// leaking the scratch directory name.
	dir := ax.Join(t.TempDir(), "core-pwa-build-123456")
	requireBuildCmdOK(t, ax.MkdirAll(dir, 0o755))

	cfg := resolvePWAAppConfig(dir)
	core.AssertEqual(t, "PWA App", cfg.DisplayName)
	core.AssertEqual(t, "pwa-app", cfg.ModuleName)
}

func TestPwa_resolvePWAAppConfig_Ugly(t *core.T) {
	// Edge case: local index.html metadata takes precedence over the directory
	// name for both display name and module slug.
	dir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(dir, "index.html"),
		[]byte(`<html><head><title>Stardust Console</title></head></html>`), 0o644))

	cfg := resolvePWAAppConfig(dir)
	core.AssertEqual(t, "Stardust Console", cfg.DisplayName)
	core.AssertEqual(t, "stardust-console", cfg.ModuleName)
}

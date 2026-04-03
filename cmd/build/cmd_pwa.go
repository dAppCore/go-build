// cmd_pwa.go implements PWA and legacy GUI build functionality.
//
// Supports building desktop applications from:
//   - Local static web application directories
//   - Live PWA URLs (downloads and packages)

package buildcmd

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/i18n"
	coreerr "dappco.re/go/core/log"
	"github.com/leaanthony/debme"
	"github.com/leaanthony/gosod"
	"golang.org/x/net/html"
)

// Error sentinels for build commands
var (
	errPathRequired = coreerr.E("buildcmd.Init", "the --path flag is required", nil)
	errURLRequired  = coreerr.E("buildcmd.Init", "the --url flag is required", nil)
)

// runLocalPwaBuild points at the local PWA build entrypoint.
// Tests replace this to avoid invoking the real build toolchain.
var runLocalPwaBuild = runBuild

// runPwaBuild downloads a PWA from URL and builds it.
func runPwaBuild(ctx context.Context, pwaURL string) error {
	core.Print(nil, "%s %s", i18n.T("cmd.build.pwa.starting"), pwaURL)

	tempDir, err := ax.TempDir("core-pwa-build-*")
	if err != nil {
		return coreerr.E("pwa.runPwaBuild", i18n.T("common.error.failed", map[string]any{"Action": "create temporary directory"}), err)
	}
	// defer os.RemoveAll(tempDir) // Keep temp dir for debugging
	core.Print(nil, "%s %s", i18n.T("cmd.build.pwa.downloading_to"), tempDir)

	if err := downloadPWA(ctx, pwaURL, tempDir); err != nil {
		return coreerr.E("pwa.runPwaBuild", i18n.T("common.error.failed", map[string]any{"Action": "download PWA"}), err)
	}

	return runBuild(ctx, tempDir)
}

// downloadPWA fetches a PWA from a URL and saves assets locally.
func downloadPWA(ctx context.Context, baseURL, destDir string) error {
	// Fetch the main HTML page
	resp, err := getWithContext(ctx, baseURL)
	if err != nil {
		return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "fetch URL"})+" "+baseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "read response body"}), err)
	}

	// Find the manifest URL from the HTML
	manifestURL, err := findManifestURL(string(body), baseURL)
	if err != nil {
		// If no manifest, it's not a PWA, but we can still try to package it as a simple site.
		core.Print(nil, "%s %s", i18n.T("common.label.warning"), i18n.T("cmd.build.pwa.no_manifest"))
		if err := ax.WriteString(ax.Join(destDir, "index.html"), string(body), 0o644); err != nil {
			return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "write index.html"}), err)
		}
		return nil
	}

	core.Print(nil, "%s %s", i18n.T("cmd.build.pwa.found_manifest"), manifestURL)

	// Fetch and parse the manifest
	manifest, err := fetchManifest(ctx, manifestURL)
	if err != nil {
		return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "fetch or parse manifest"}), err)
	}

	// Download all assets listed in the manifest
	assets := collectAssets(manifest, manifestURL)
	for _, assetURL := range assets {
		if err := downloadAsset(ctx, assetURL, destDir); err != nil {
			if ctx.Err() != nil {
				return coreerr.E("pwa.downloadPWA", "download cancelled", ctx.Err())
			}
			core.Print(nil, "%s %s %s: %v", i18n.T("common.label.warning"), i18n.T("common.error.failed", map[string]any{"Action": "download asset"}), assetURL, err)
		}
	}

	// Also save the root index.html
	if err := ax.WriteString(ax.Join(destDir, "index.html"), string(body), 0o644); err != nil {
		return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "write index.html"}), err)
	}

	core.Println(i18n.T("cmd.build.pwa.download_complete"))
	return nil
}

// findManifestURL extracts the manifest URL from HTML content.
func findManifestURL(htmlContent, baseURL string) (string, error) {
	doc, err := html.Parse(core.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	var manifestPath string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href string
			for _, a := range n.Attr {
				if a.Key == "rel" {
					rel = a.Val
				}
				if a.Key == "href" {
					href = a.Val
				}
			}
			if relIncludesManifest(rel) && href != "" {
				manifestPath = href
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if manifestPath == "" {
		return "", coreerr.E("pwa.findManifestURL", i18n.T("cmd.build.pwa.error.no_manifest_tag"), nil)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	manifestURL, err := base.Parse(manifestPath)
	if err != nil {
		return "", err
	}

	return manifestURL.String(), nil
}

// relIncludesManifest reports whether a rel attribute declares a manifest link.
// HTML allows multiple space-separated tokens and case-insensitive values.
func relIncludesManifest(rel string) bool {
	for _, token := range strings.Fields(rel) {
		if strings.EqualFold(token, "manifest") {
			return true
		}
	}
	return false
}

// fetchManifest downloads and parses a PWA manifest.
func fetchManifest(ctx context.Context, manifestURL string) (map[string]any, error) {
	resp, err := getWithContext(ctx, manifestURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var manifest map[string]any
	if err := ax.JSONUnmarshal(body, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

// collectAssets extracts asset URLs from a PWA manifest.
func collectAssets(manifest map[string]any, manifestURL string) []string {
	var assets []string
	base, _ := url.Parse(manifestURL)

	// Add start_url
	if startURL, ok := manifest["start_url"].(string); ok {
		if resolved, err := base.Parse(startURL); err == nil {
			assets = append(assets, resolved.String())
		}
	}

	// Add icons
	if icons, ok := manifest["icons"].([]any); ok {
		for _, icon := range icons {
			if iconMap, ok := icon.(map[string]any); ok {
				if src, ok := iconMap["src"].(string); ok {
					if resolved, err := base.Parse(src); err == nil {
						assets = append(assets, resolved.String())
					}
				}
			}
		}
	}

	return assets
}

// downloadAsset fetches a single asset and saves it locally.
func downloadAsset(ctx context.Context, assetURL, destDir string) error {
	resp, err := getWithContext(ctx, assetURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	u, err := url.Parse(assetURL)
	if err != nil {
		return err
	}

	assetPath := core.TrimPrefix(ax.FromSlash(u.Path), ax.DS())
	path := ax.Join(destDir, assetPath)
	if err := ax.MkdirAll(ax.Dir(path), 0o755); err != nil {
		return err
	}

	out, err := ax.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
}

// runBuild builds a desktop application from a local directory.
func runBuild(ctx context.Context, fromPath string) error {
	core.Print(nil, "%s %s", i18n.T("cmd.build.from_path.starting"), fromPath)

	if !ax.IsDir(fromPath) {
		return coreerr.E("pwa.runBuild", i18n.T("cmd.build.from_path.error.must_be_directory"), nil)
	}

	buildDir := ".core/build/app"
	htmlDir := ax.Join(buildDir, "html")
	appName := ax.Base(fromPath)
	if core.HasPrefix(appName, "core-pwa-build-") {
		appName = "pwa-app"
	}
	outputExe := appName

	if err := ax.RemoveAll(buildDir); err != nil {
		return coreerr.E("pwa.runBuild", i18n.T("common.error.failed", map[string]any{"Action": "clean build directory"}), err)
	}

	// 1. Generate the project from the embedded template
	core.Println(i18n.T("cmd.build.from_path.generating_template"))
	templateFS, err := debme.FS(guiTemplate, "tmpl/gui")
	if err != nil {
		return coreerr.E("pwa.runBuild", i18n.T("common.error.failed", map[string]any{"Action": "anchor template filesystem"}), err)
	}
	sod := gosod.New(templateFS)
	if sod == nil {
		return coreerr.E("pwa.runBuild", i18n.T("common.error.failed", map[string]any{"Action": "create new sod instance"}), nil)
	}

	templateData := map[string]string{"AppName": appName}
	if err := sod.Extract(buildDir, templateData); err != nil {
		return coreerr.E("pwa.runBuild", i18n.T("common.error.failed", map[string]any{"Action": "extract template"}), err)
	}

	// 2. Copy the user's web app files
	core.Println(i18n.T("cmd.build.from_path.copying_files"))
	if err := copyDir(fromPath, htmlDir); err != nil {
		return coreerr.E("pwa.runBuild", i18n.T("common.error.failed", map[string]any{"Action": "copy application files"}), err)
	}

	// 3. Compile the application
	core.Println(i18n.T("cmd.build.from_path.compiling"))

	// Run go mod tidy
	if err := ax.ExecDir(ctx, buildDir, "go", "mod", "tidy"); err != nil {
		return coreerr.E("pwa.runBuild", i18n.T("cmd.build.from_path.error.go_mod_tidy"), err)
	}

	// Run go build
	if err := ax.ExecDir(ctx, buildDir, "go", "build", "-o", outputExe); err != nil {
		return coreerr.E("pwa.runBuild", i18n.T("cmd.build.from_path.error.go_build"), err)
	}

	core.Println()
	core.Print(nil, "%s %s/%s", i18n.T("cmd.build.from_path.success"), buildDir, outputExe)
	return nil
}

func getWithContext(ctx context.Context, targetURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	if err := ax.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := ax.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := ax.Join(src, entry.Name())
		dstPath := ax.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		srcFile, err := ax.Open(srcPath)
		if err != nil {
			return err
		}

		dstFile, err := ax.Create(dstPath)
		if err != nil {
			_ = srcFile.Close()
			return err
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			_ = srcFile.Close()
			_ = dstFile.Close()
			return err
		}
		if err := srcFile.Close(); err != nil {
			_ = dstFile.Close()
			return err
		}
		if err := dstFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

// cmd_pwa.go implements PWA and legacy GUI build functionality.
//
// Supports building desktop applications from:
//   - Local static web application directories
//   - Live PWA URLs (downloads and packages)

package buildcmd

import (
	"context"
	stdio "io" // AX-6 intrinsic: response and file streams expose standard io contracts.
	"net/http" // AX-6 intrinsic: core has no HTTP fetch primitive yet.

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	"dappco.re/go/core/i18n"
	coreerr "dappco.re/go/core/log"
	"github.com/leaanthony/debme"
	"github.com/leaanthony/gosod"
	"golang.org/x/net/html"
)

// Error sentinels for build commands
var (
	errPathRequired     = coreerr.E("buildcmd.Init", "the --path flag is required", nil)
	errURLRequired      = coreerr.E("buildcmd.Init", "the --url flag is required", nil)
	errPWAInputRequired = coreerr.E("buildcmd.Init", "either --path or --url is required", nil)
)

// runLocalPwaBuild points at the local PWA build entrypoint.
// Tests replace this to avoid invoking the real build toolchain.
var runLocalPwaBuild = runBuild

const defaultPWADescription = "A web application enclaved by Core."

type pwaMetadata struct {
	DisplayName string
	Description string
	ManifestURL string
	Icons       []string
}

type pwaAppConfig struct {
	ModuleName  string
	DisplayName string
	Description string
}

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
	resp, err := getWithContext(ctx, baseURL)
	if err != nil {
		return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "fetch URL"})+" "+baseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := stdio.ReadAll(resp.Body)
	if err != nil {
		return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "read response body"}), err)
	}

	pageMetadata, assets, err := extractHTMLMetadataAndAssets(string(body), baseURL)
	if err != nil {
		return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "parse HTML entry point"}), err)
	}

	if err := ax.WriteFile(ax.Join(destDir, "index.html"), body, 0o644); err != nil {
		return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "write index.html"}), err)
	}

	downloaded := map[string]struct{}{
		normalizeAssetURL(baseURL): {},
	}

	if pageMetadata.ManifestURL == "" {
		core.Print(nil, "%s %s", i18n.T("common.label.warning"), i18n.T("cmd.build.pwa.no_manifest"))
	} else {
		core.Print(nil, "%s %s", i18n.T("cmd.build.pwa.found_manifest"), pageMetadata.ManifestURL)

		manifest, manifestBody, err := fetchManifest(ctx, pageMetadata.ManifestURL)
		if err != nil {
			return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "fetch or parse manifest"}), err)
		}

		if err := writeURLAsset(destDir, pageMetadata.ManifestURL, manifestBody); err != nil {
			return coreerr.E("pwa.downloadPWA", i18n.T("common.error.failed", map[string]any{"Action": "write manifest"}), err)
		}
		downloaded[normalizeAssetURL(pageMetadata.ManifestURL)] = struct{}{}
		assets = append(assets, collectAssets(manifest, pageMetadata.ManifestURL)...)
	}

	for _, assetURL := range uniquePWAStrings(assets) {
		normalized := normalizeAssetURL(assetURL)
		if normalized == "" {
			continue
		}
		if _, ok := downloaded[normalized]; ok {
			continue
		}
		if err := downloadAsset(ctx, assetURL, destDir); err != nil {
			if ctx.Err() != nil {
				return coreerr.E("pwa.downloadPWA", "download cancelled", ctx.Err())
			}
			core.Print(nil, "%s %s %s: %v", i18n.T("common.label.warning"), i18n.T("common.error.failed", map[string]any{"Action": "download asset"}), assetURL, err)
			continue
		}
		downloaded[normalized] = struct{}{}
	}

	core.Println(i18n.T("cmd.build.pwa.download_complete"))
	return nil
}

// findManifestURL extracts the manifest URL from HTML content.
func findManifestURL(htmlContent, baseURL string) (string, error) {
	metadata, _, err := extractHTMLMetadataAndAssets(htmlContent, baseURL)
	if err != nil {
		return "", err
	}
	if metadata.ManifestURL == "" {
		return "", coreerr.E("pwa.findManifestURL", i18n.T("cmd.build.pwa.error.no_manifest_tag"), nil)
	}
	return metadata.ManifestURL, nil
}

func extractHTMLMetadataAndAssets(htmlContent, baseURL string) (pwaMetadata, []string, error) {
	doc, err := html.Parse(core.NewReader(htmlContent))
	if err != nil {
		return pwaMetadata{}, nil, err
	}

	if _, err := core.URLParse(baseURL); err != nil {
		return pwaMetadata{}, nil, err
	}

	var (
		metadata pwaMetadata
		assets   []string
	)

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch core.Lower(core.Trim(node.Data)) {
			case "title":
				if metadata.DisplayName == "" {
					metadata.DisplayName = core.Trim(nodeText(node))
				}
			case "meta":
				content := core.Trim(attributeValue(node, "content"))
				name := core.Lower(core.Trim(attributeValue(node, "name")))
				property := core.Lower(core.Trim(attributeValue(node, "property")))
				if content != "" && (name == "description" || property == "og:description" || property == "twitter:description") && metadata.Description == "" {
					metadata.Description = content
				}
			case "link":
				relValue := attributeValue(node, "rel")
				href := attributeValue(node, "href")
				rel := parseRelTokens(relValue)
				resolved := resolveAssetURL(baseURL, href)
				if resolved != "" && relHasAny(rel, "stylesheet", "icon", "shortcut", "apple-touch-icon", "mask-icon", "preload", "modulepreload", "prefetch", "manifest") {
					assets = append(assets, resolved)
				}
				if relIncludesManifest(relValue) && resolved != "" && metadata.ManifestURL == "" {
					metadata.ManifestURL = resolved
				}
				if resolved != "" && relHasAny(rel, "icon", "apple-touch-icon", "mask-icon") {
					metadata.Icons = append(metadata.Icons, resolved)
				}
			case "script":
				appendResolvedAsset(&assets, baseURL, attributeValue(node, "src"))
			case "img":
				appendResolvedAsset(&assets, baseURL, attributeValue(node, "src"))
				appendResolvedSrcSet(&assets, baseURL, attributeValue(node, "srcset"))
			case "source":
				appendResolvedAsset(&assets, baseURL, attributeValue(node, "src"))
				appendResolvedSrcSet(&assets, baseURL, attributeValue(node, "srcset"))
			case "video":
				appendResolvedAsset(&assets, baseURL, attributeValue(node, "poster"))
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	metadata.Icons = uniquePWAStrings(metadata.Icons)
	assets = uniquePWAStrings(assets)
	return metadata, assets, nil
}

// relIncludesManifest reports whether a rel attribute declares a manifest link.
// HTML allows multiple space-separated tokens and case-insensitive values.
func relIncludesManifest(rel string) bool {
	for _, token := range parseRelTokens(rel) {
		if token == "manifest" {
			return true
		}
	}
	return false
}

// fetchManifest downloads and parses a PWA manifest.
func fetchManifest(ctx context.Context, manifestURL string) (map[string]any, []byte, error) {
	resp, err := getWithContext(ctx, manifestURL)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := stdio.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var manifest map[string]any
	if err := ax.JSONUnmarshal(body, &manifest); err != nil {
		return nil, nil, err
	}
	return manifest, body, nil
}

// collectAssets extracts asset URLs from a PWA manifest.
func collectAssets(manifest map[string]any, manifestURL string) []string {
	_, assets := manifestMetadataAndAssets(manifest, manifestURL)
	return assets
}

// downloadAsset fetches a single asset and saves it locally.
func downloadAsset(ctx context.Context, assetURL, destDir string) error {
	resp, err := getWithContext(ctx, assetURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := stdio.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return writeURLAsset(destDir, assetURL, body)
}

func writeURLAsset(destDir, assetURL string, body []byte) error {
	targetPath, err := resolveAssetDestination(destDir, assetURL)
	if err != nil {
		return err
	}
	if err := ax.MkdirAll(ax.Dir(targetPath), 0o755); err != nil {
		return err
	}
	return ax.WriteFile(targetPath, body, 0o644)
}

// runBuild builds a desktop application from a local directory.
func runBuild(ctx context.Context, fromPath string) error {
	core.Print(nil, "%s %s", i18n.T("cmd.build.from_path.starting"), fromPath)

	if !ax.IsDir(fromPath) {
		return coreerr.E("pwa.runBuild", i18n.T("cmd.build.from_path.error.must_be_directory"), nil)
	}

	buildDir := ".core/build/app"
	htmlDir := ax.Join(buildDir, "html")
	appConfig := resolvePWAAppConfig(fromPath)
	outputExe := appConfig.ModuleName

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

	templateData := map[string]string{
		"AppModule":             appConfig.ModuleName,
		"AppDisplayNameLiteral": core.Sprintf("%q", appConfig.DisplayName),
		"AppDescriptionLiteral": core.Sprintf("%q", appConfig.Description),
	}
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

func resolvePWAAppConfig(fromPath string) pwaAppConfig {
	fallbackName := ax.Base(fromPath)
	if core.HasPrefix(fallbackName, "core-pwa-build-") {
		fallbackName = "PWA App"
	}

	metadata := loadLocalPWAMetadata(fromPath)
	displayName := core.Trim(metadata.DisplayName)
	if displayName == "" {
		displayName = fallbackName
	}

	description := core.Trim(metadata.Description)
	if description == "" {
		description = defaultPWADescription
	}

	moduleName := slugifyPWAName(displayName)
	if moduleName == "" {
		moduleName = slugifyPWAName(fallbackName)
	}
	if moduleName == "" {
		moduleName = "pwa-app"
	}

	return pwaAppConfig{
		ModuleName:  moduleName,
		DisplayName: displayName,
		Description: description,
	}
}

func loadLocalPWAMetadata(dir string) pwaMetadata {
	indexPath := ax.Join(dir, "index.html")
	if !ax.IsFile(indexPath) {
		return pwaMetadata{}
	}

	content, err := ax.ReadFile(indexPath)
	if err != nil {
		return pwaMetadata{}
	}

	metadata, _, err := extractHTMLMetadataAndAssets(string(content), "https://local.core/")
	if err != nil {
		return pwaMetadata{}
	}

	for _, manifestPath := range localManifestCandidates(dir, metadata.ManifestURL) {
		if !ax.IsFile(manifestPath) {
			continue
		}

		manifestBody, err := ax.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		relativePath, err := ax.Rel(dir, manifestPath)
		if err != nil {
			continue
		}
		manifestURLPath := core.CleanPath(core.Concat("/", core.Replace(relativePath, ax.DS(), "/")), "/")
		manifestURL := core.Concat("https://local.core/", core.TrimPrefix(manifestURLPath, "/"))
		manifestMetadata, _ := manifestMetadataAndAssetsFromBytes(manifestBody, manifestURL)
		return mergePWAMetadata(metadata, manifestMetadata)
	}

	return metadata
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

		if _, err := stdio.Copy(dstFile, srcFile); err != nil {
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

func manifestMetadataAndAssets(manifest map[string]any, manifestURL string) (pwaMetadata, []string) {
	metadata := pwaMetadata{}
	var assets []string

	if name, ok := manifest["name"].(string); ok && core.Trim(name) != "" {
		metadata.DisplayName = core.Trim(name)
	} else if shortName, ok := manifest["short_name"].(string); ok {
		metadata.DisplayName = core.Trim(shortName)
	}

	if description, ok := manifest["description"].(string); ok {
		metadata.Description = core.Trim(description)
	}

	if startURL, ok := manifest["start_url"].(string); ok {
		appendResolvedAsset(&assets, manifestURL, startURL)
	}

	if icons, ok := manifest["icons"].([]any); ok {
		for _, icon := range icons {
			iconMap, ok := icon.(map[string]any)
			if !ok {
				continue
			}
			src, _ := iconMap["src"].(string)
			resolved := resolveAssetURL(manifestURL, src)
			if resolved == "" {
				continue
			}
			metadata.Icons = append(metadata.Icons, resolved)
			assets = append(assets, resolved)
		}
	}

	metadata.Icons = uniquePWAStrings(metadata.Icons)
	assets = uniquePWAStrings(assets)
	return metadata, assets
}

func manifestMetadataAndAssetsFromBytes(body []byte, manifestURL string) (pwaMetadata, []string) {
	var manifest map[string]any
	if err := ax.JSONUnmarshal(body, &manifest); err != nil {
		return pwaMetadata{}, nil
	}
	return manifestMetadataAndAssets(manifest, manifestURL)
}

func mergePWAMetadata(base, override pwaMetadata) pwaMetadata {
	merged := base
	if core.Trim(override.DisplayName) != "" {
		merged.DisplayName = core.Trim(override.DisplayName)
	}
	if core.Trim(override.Description) != "" {
		merged.Description = core.Trim(override.Description)
	}
	if core.Trim(override.ManifestURL) != "" {
		merged.ManifestURL = core.Trim(override.ManifestURL)
	}
	merged.Icons = uniquePWAStrings(append(append([]string{}, base.Icons...), override.Icons...))
	return merged
}

func attributeValue(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if core.Lower(attribute.Key) == core.Lower(name) {
			return attribute.Val
		}
	}
	return ""
}

func nodeText(node *html.Node) string {
	b := core.NewBuilder()
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			b.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return b.String()
}

func parseRelTokens(value string) []string {
	return uniquePWAStrings(pwaFields(core.Lower(core.Trim(value))))
}

func pwaFields(value string) []string {
	var fields []string
	fieldStart := -1
	for i, r := range value {
		if pwaIsSpace(r) {
			if fieldStart >= 0 {
				fields = append(fields, value[fieldStart:i])
				fieldStart = -1
			}
			continue
		}
		if fieldStart < 0 {
			fieldStart = i
		}
	}
	if fieldStart >= 0 {
		fields = append(fields, value[fieldStart:])
	}
	return fields
}

func pwaIsSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

func trimPWAEdgeDashes(value string) string {
	start := 0
	for start < len(value) && value[start] == '-' {
		start++
	}
	end := len(value)
	for end > start && value[end-1] == '-' {
		end--
	}
	return value[start:end]
}

func relHasAny(tokens []string, candidates ...string) bool {
	for _, token := range tokens {
		for _, candidate := range candidates {
			if token == candidate {
				return true
			}
		}
	}
	return false
}

func resolveAssetURL(baseURL, raw string) string {
	raw = core.Trim(raw)
	if raw == "" || core.HasPrefix(raw, "#") {
		return ""
	}

	lower := core.Lower(raw)
	if core.HasPrefix(lower, "data:") || core.HasPrefix(lower, "javascript:") || core.HasPrefix(lower, "mailto:") {
		return ""
	}

	base, err := core.URLParse(baseURL)
	if err != nil {
		return ""
	}
	resolved, err := base.Parse(raw)
	if err != nil {
		return ""
	}
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	resolved.Fragment = ""
	return resolved.String()
}

func appendResolvedAsset(assets *[]string, baseURL, raw string) {
	resolved := resolveAssetURL(baseURL, raw)
	if resolved != "" {
		*assets = append(*assets, resolved)
	}
}

func appendResolvedSrcSet(assets *[]string, baseURL, raw string) {
	for _, candidate := range core.Split(raw, ",") {
		candidate = core.Trim(candidate)
		if candidate == "" {
			continue
		}
		fields := pwaFields(candidate)
		if len(fields) == 0 {
			continue
		}
		appendResolvedAsset(assets, baseURL, fields[0])
	}
}

func uniquePWAStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = core.Trim(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeAssetURL(raw string) string {
	parsed, err := core.URLParse(core.Trim(raw))
	if err != nil {
		return ""
	}
	parsed.Fragment = ""
	return parsed.String()
}

func resolveAssetDestination(destDir, assetURL string) (string, error) {
	parsed, err := core.URLParse(assetURL)
	if err != nil {
		return "", err
	}

	relativePath := core.CleanPath(core.Concat("/", parsed.Path), "/")
	switch {
	case relativePath == "/" || relativePath == ".":
		relativePath = "/index.html"
	case core.HasSuffix(parsed.Path, "/"):
		relativePath = core.JoinPath(relativePath, "index.html")
	}

	return ax.Join(destDir, ax.FromSlash(core.TrimPrefix(relativePath, "/"))), nil
}

func localManifestCandidates(dir, manifestURL string) []string {
	candidates := make([]string, 0, 3)
	if manifestURL != "" {
		if localPath := localAssetPath(dir, manifestURL); localPath != "" {
			candidates = append(candidates, localPath)
		}
	}
	candidates = append(candidates, ax.Join(dir, "manifest.json"), ax.Join(dir, "manifest.webmanifest"))
	return uniquePWAStrings(candidates)
}

func localAssetPath(dir, assetURL string) string {
	parsed, err := core.URLParse(assetURL)
	if err != nil {
		return ""
	}

	relativePath := core.CleanPath(core.Concat("/", parsed.Path), "/")
	if relativePath == "/" || relativePath == "." {
		relativePath = "/index.html"
	}
	return ax.Join(dir, ax.FromSlash(core.TrimPrefix(relativePath, "/")))
}

func slugifyPWAName(name string) string {
	name = core.Trim(name)
	if name == "" {
		return ""
	}

	b := core.NewBuilder()
	lastDash := false
	for _, r := range core.Lower(name) {
		switch {
		case r <= 127 && (core.IsLetter(r) || core.IsDigit(r)):
			b.WriteRune(r)
			lastDash = false
		case pwaIsSpace(r) || r == '-' || r == '_' || r == '.':
			if b.Len() == 0 || lastDash {
				continue
			}
			b.WriteByte('-')
			lastDash = true
		}
	}

	slug := trimPWAEdgeDashes(b.String())
	if slug == "" {
		return ""
	}
	if slug[0] >= '0' && slug[0] <= '9' {
		return "app-" + slug
	}
	return slug
}

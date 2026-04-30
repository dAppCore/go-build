// cmd_pwa.go implements PWA and legacy GUI build functionality.
//
// Supports building desktop applications from:
//   - Local static web application directories
//   - Live PWA URLs (downloads and packages)

package buildcmd

import (
	// Note: AX-6 — context.Context is the command cancellation contract; core has no equivalent API.
	"context"
	"io/fs"
	// Note: AX-6 — net/http is required for PWA downloads; core has no HTTP client primitive.
	"net/http"
	// Note: AX-6 — net/url is required for standards-compliant URL parsing/resolution; core has only path/string primitives here.
	"net/url"
	// Note: AX-6 — unicode preserves Fields/slug whitespace semantics; core has no rune category primitive.
	"unicode"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"github.com/leaanthony/debme"
	"github.com/leaanthony/gosod"
	"golang.org/x/net/html"
)

// Error sentinels for build commands
var (
	errPathRequired     = core.E("buildcmd.Init", "the --path flag is required", nil)
	errURLRequired      = core.E("buildcmd.Init", "the --url flag is required", nil)
	errPWAInputRequired = core.E("buildcmd.Init", "either --path or --url is required", nil)
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

type pwaHTMLExtraction struct {
	Metadata pwaMetadata
	Assets   []string
}

type pwaManifestFetch struct {
	Manifest map[string]any
	Body     []byte
}

// runPwaBuild downloads a PWA from URL and builds it.
func runPwaBuild(ctx context.Context, pwaURL string) core.Result {
	core.Print(nil, "%s %s", "Building PWA", pwaURL)

	tempDirResult := ax.TempDir("core-pwa-build-*")
	if !tempDirResult.OK {
		return core.Fail(core.E("pwa.runPwaBuild", "failed to create temporary directory", core.NewError(tempDirResult.Error())))
	}
	tempDir := tempDirResult.Value.(string)
	// defer os.RemoveAll(tempDir) // Keep temp dir for debugging
	core.Print(nil, "%s %s", "Downloading to", tempDir)

	downloaded := downloadPWA(ctx, pwaURL, tempDir)
	if !downloaded.OK {
		return core.Fail(core.E("pwa.runPwaBuild", "failed to download PWA", core.NewError(downloaded.Error())))
	}

	return runBuild(ctx, tempDir)
}

// downloadPWA fetches a PWA from a URL and saves assets locally.
func downloadPWA(ctx context.Context, baseURL, destDir string) core.Result {
	respResult := getWithContext(ctx, baseURL)
	if !respResult.OK {
		return core.Fail(core.E("pwa.downloadPWA", "failed to fetch URL "+baseURL, core.NewError(respResult.Error())))
	}
	resp := respResult.Value.(*http.Response)
	bodyResult := readAllBytes(resp.Body)
	if !bodyResult.OK {
		return core.Fail(core.E("pwa.downloadPWA", "failed to read response body", core.NewError(bodyResult.Error())))
	}
	body := bodyResult.Value.([]byte)

	extractedResult := extractHTMLMetadataAndAssets(string(body), baseURL)
	if !extractedResult.OK {
		return core.Fail(core.E("pwa.downloadPWA", "failed to parse HTML entry point", core.NewError(extractedResult.Error())))
	}
	extracted := extractedResult.Value.(pwaHTMLExtraction)
	pageMetadata := extracted.Metadata
	assets := extracted.Assets

	writtenIndex := ax.WriteFile(ax.Join(destDir, "index.html"), body, 0o644)
	if !writtenIndex.OK {
		return core.Fail(core.E("pwa.downloadPWA", "failed to write index.html", core.NewError(writtenIndex.Error())))
	}

	downloaded := map[string]struct{}{
		normalizeAssetURL(baseURL): {},
	}

	if pageMetadata.ManifestURL == "" {
		core.Print(nil, "%s %s", "warning", "no manifest found")
	} else {
		core.Print(nil, "%s %s", "Found manifest", pageMetadata.ManifestURL)

		manifestResult := fetchManifest(ctx, pageMetadata.ManifestURL)
		if !manifestResult.OK {
			return core.Fail(core.E("pwa.downloadPWA", "failed to fetch or parse manifest", core.NewError(manifestResult.Error())))
		}
		manifestFetch := manifestResult.Value.(pwaManifestFetch)

		manifestWritten := writeURLAsset(destDir, pageMetadata.ManifestURL, manifestFetch.Body)
		if !manifestWritten.OK {
			return core.Fail(core.E("pwa.downloadPWA", "failed to write manifest", core.NewError(manifestWritten.Error())))
		}
		downloaded[normalizeAssetURL(pageMetadata.ManifestURL)] = struct{}{}
		assets = append(assets, collectAssets(manifestFetch.Manifest, pageMetadata.ManifestURL)...)
	}

	for _, assetURL := range uniquePWAStrings(assets) {
		normalized := normalizeAssetURL(assetURL)
		if normalized == "" {
			continue
		}
		if _, ok := downloaded[normalized]; ok {
			continue
		}
		assetDownloaded := downloadAsset(ctx, assetURL, destDir)
		if !assetDownloaded.OK {
			if ctx.Err() != nil {
				return core.Fail(core.E("pwa.downloadPWA", "download cancelled", ctx.Err()))
			}
			core.Print(nil, "%s %s %s: %v", "warning", "failed to download asset", assetURL, assetDownloaded.Error())
			continue
		}
		downloaded[normalized] = struct{}{}
	}

	core.Println("PWA download complete")
	return core.Ok(nil)
}

// findManifestURL extracts the manifest URL from HTML content.
func findManifestURL(htmlContent, baseURL string) core.Result {
	extracted := extractHTMLMetadataAndAssets(htmlContent, baseURL)
	if !extracted.OK {
		return extracted
	}
	metadata := extracted.Value.(pwaHTMLExtraction).Metadata
	if metadata.ManifestURL == "" {
		return core.Fail(core.E("pwa.findManifestURL", "manifest tag not found", nil))
	}
	return core.Ok(metadata.ManifestURL)
}

func extractHTMLMetadataAndAssets(htmlContent, baseURL string) core.Result {
	doc, err := html.Parse(core.NewReader(htmlContent))
	if err != nil {
		return core.Fail(err)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return core.Fail(err)
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
				resolved := resolveAssetURL(base, href)
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
				appendResolvedAsset(&assets, base, attributeValue(node, "src"))
			case "img":
				appendResolvedAsset(&assets, base, attributeValue(node, "src"))
				appendResolvedSrcSet(&assets, base, attributeValue(node, "srcset"))
			case "source":
				appendResolvedAsset(&assets, base, attributeValue(node, "src"))
				appendResolvedSrcSet(&assets, base, attributeValue(node, "srcset"))
			case "video":
				appendResolvedAsset(&assets, base, attributeValue(node, "poster"))
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	metadata.Icons = uniquePWAStrings(metadata.Icons)
	assets = uniquePWAStrings(assets)
	return core.Ok(pwaHTMLExtraction{Metadata: metadata, Assets: assets})
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
func fetchManifest(ctx context.Context, manifestURL string) core.Result {
	respResult := getWithContext(ctx, manifestURL)
	if !respResult.OK {
		return respResult
	}
	resp := respResult.Value.(*http.Response)
	bodyResult := readAllBytes(resp.Body)
	if !bodyResult.OK {
		return bodyResult
	}
	body := bodyResult.Value.([]byte)

	var manifest map[string]any
	decoded := ax.JSONUnmarshal(body, &manifest)
	if !decoded.OK {
		return decoded
	}
	return core.Ok(pwaManifestFetch{Manifest: manifest, Body: body})
}

// collectAssets extracts asset URLs from a PWA manifest.
func collectAssets(manifest map[string]any, manifestURL string) []string {
	_, assets := manifestMetadataAndAssets(manifest, manifestURL)
	return assets
}

// downloadAsset fetches a single asset and saves it locally.
func downloadAsset(ctx context.Context, assetURL, destDir string) core.Result {
	respResult := getWithContext(ctx, assetURL)
	if !respResult.OK {
		return respResult
	}
	resp := respResult.Value.(*http.Response)
	bodyResult := readAllBytes(resp.Body)
	if !bodyResult.OK {
		return bodyResult
	}
	body := bodyResult.Value.([]byte)

	return writeURLAsset(destDir, assetURL, body)
}

func writeURLAsset(destDir, assetURL string, body []byte) core.Result {
	targetPathResult := resolveAssetDestination(destDir, assetURL)
	if !targetPathResult.OK {
		return targetPathResult
	}
	targetPath := targetPathResult.Value.(string)
	created := ax.MkdirAll(ax.Dir(targetPath), 0o755)
	if !created.OK {
		return created
	}
	return ax.WriteFile(targetPath, body, 0o644)
}

// runBuild builds a desktop application from a local directory.
func runBuild(ctx context.Context, fromPath string) core.Result {
	core.Print(nil, "%s %s", "Building from path", fromPath)

	if !ax.IsDir(fromPath) {
		return core.Fail(core.E("pwa.runBuild", "path must be a directory", nil))
	}

	buildDir := ".core/build/app"
	htmlDir := ax.Join(buildDir, "html")
	appConfig := resolvePWAAppConfig(fromPath)
	outputExe := appConfig.ModuleName

	removed := ax.RemoveAll(buildDir)
	if !removed.OK {
		return core.Fail(core.E("pwa.runBuild", "failed to clean build directory", core.NewError(removed.Error())))
	}

	// 1. Generate the project from the embedded template
	core.Println("Generating template")
	templateFS, err := debme.FS(guiTemplate, "tmpl/gui")
	if err != nil {
		return core.Fail(core.E("pwa.runBuild", "failed to anchor template filesystem", err))
	}
	sod := gosod.New(templateFS)
	if sod == nil {
		return core.Fail(core.E("pwa.runBuild", "failed to create new sod instance", nil))
	}

	templateData := map[string]string{
		"AppModule":             appConfig.ModuleName,
		"AppDisplayNameLiteral": core.Sprintf("%q", appConfig.DisplayName),
		"AppDescriptionLiteral": core.Sprintf("%q", appConfig.Description),
	}
	if err := sod.Extract(buildDir, templateData); err != nil {
		return core.Fail(core.E("pwa.runBuild", "failed to extract template", err))
	}

	// 2. Copy the user's web app files
	core.Println("Copying files")
	copied := copyDir(fromPath, htmlDir)
	if !copied.OK {
		return core.Fail(core.E("pwa.runBuild", "failed to copy application files", core.NewError(copied.Error())))
	}

	// 3. Compile the application
	core.Println("Compiling")

	// Run go mod tidy
	tidied := ax.ExecDir(ctx, buildDir, "go", "mod", "tidy")
	if !tidied.OK {
		return core.Fail(core.E("pwa.runBuild", "go mod tidy failed", core.NewError(tidied.Error())))
	}

	// Run go build
	built := ax.ExecDir(ctx, buildDir, "go", "build", "-o", outputExe)
	if !built.OK {
		return core.Fail(core.E("pwa.runBuild", "go build failed", core.NewError(built.Error())))
	}

	core.Println()
	core.Print(nil, "%s %s/%s", "Built", buildDir, outputExe)
	return core.Ok(nil)
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

	contentResult := ax.ReadFile(indexPath)
	if !contentResult.OK {
		return pwaMetadata{}
	}
	content := contentResult.Value.([]byte)

	extracted := extractHTMLMetadataAndAssets(string(content), "https://local.core/")
	if !extracted.OK {
		return pwaMetadata{}
	}
	metadata := extracted.Value.(pwaHTMLExtraction).Metadata

	for _, manifestPath := range localManifestCandidates(dir, metadata.ManifestURL) {
		if !ax.IsFile(manifestPath) {
			continue
		}

		manifestBodyResult := ax.ReadFile(manifestPath)
		if !manifestBodyResult.OK {
			continue
		}
		manifestBody := manifestBodyResult.Value.([]byte)

		relativePathResult := ax.Rel(dir, manifestPath)
		if !relativePathResult.OK {
			continue
		}
		relativePath := relativePathResult.Value.(string)
		manifestURL := core.Concat("https://local.core/", localPWAURLPath(relativePath))
		manifestMetadata, _ := manifestMetadataAndAssetsFromBytes(manifestBody, manifestURL)
		return mergePWAMetadata(metadata, manifestMetadata)
	}

	return metadata
}

func getWithContext(ctx context.Context, targetURL string) core.Result {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return core.Fail(err)
	}
	return core.ResultOf(http.DefaultClient.Do(req))
}

func readAllBytes(reader any) core.Result {
	result := core.ReadAll(reader)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return core.Fail(err)
		}
		return core.Fail(core.E("pwa.readAllBytes", "failed to read stream", nil))
	}

	content, ok := result.Value.(string)
	if !ok {
		return core.Fail(core.E("pwa.readAllBytes", "read stream returned non-string content", nil))
	}
	return core.Ok([]byte(content))
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) core.Result {
	created := ax.MkdirAll(dst, 0o755)
	if !created.OK {
		return created
	}

	entriesResult := ax.ReadDir(src)
	if !entriesResult.OK {
		return entriesResult
	}
	entries := entriesResult.Value.([]fs.DirEntry)

	for _, entry := range entries {
		srcPath := ax.Join(src, entry.Name())
		dstPath := ax.Join(dst, entry.Name())

		if entry.IsDir() {
			copied := copyDir(srcPath, dstPath)
			if !copied.OK {
				return copied
			}
			continue
		}

		srcFile := ax.Open(srcPath)
		if !srcFile.OK {
			return srcFile
		}

		content := readAllBytes(srcFile.Value)
		if !content.OK {
			return content
		}

		written := ax.WriteFile(dstPath, content.Value.([]byte), 0o644)
		if !written.OK {
			return written
		}
	}

	return core.Ok(nil)
}

func manifestMetadataAndAssets(manifest map[string]any, manifestURL string) (pwaMetadata, []string) {
	metadata := pwaMetadata{}
	var assets []string
	base, _ := url.Parse(manifestURL)

	if name, ok := manifest["name"].(string); ok && core.Trim(name) != "" {
		metadata.DisplayName = core.Trim(name)
	} else if shortName, ok := manifest["short_name"].(string); ok {
		metadata.DisplayName = core.Trim(shortName)
	}

	if description, ok := manifest["description"].(string); ok {
		metadata.Description = core.Trim(description)
	}

	if startURL, ok := manifest["start_url"].(string); ok {
		appendResolvedAsset(&assets, base, startURL)
	}

	if icons, ok := manifest["icons"].([]any); ok {
		for _, icon := range icons {
			iconMap, ok := icon.(map[string]any)
			if !ok {
				continue
			}
			src, _ := iconMap["src"].(string)
			resolved := resolveAssetURL(base, src)
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
	decoded := ax.JSONUnmarshal(body, &manifest)
	if !decoded.OK {
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
	needle := core.Lower(name)
	for _, attribute := range node.Attr {
		if core.Lower(attribute.Key) == needle {
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

func resolveAssetURL(base *url.URL, raw string) string {
	raw = core.Trim(raw)
	if raw == "" || core.HasPrefix(raw, "#") {
		return ""
	}

	lower := core.Lower(raw)
	if core.HasPrefix(lower, "data:") || core.HasPrefix(lower, "javascript:") || core.HasPrefix(lower, "mailto:") {
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

func appendResolvedAsset(assets *[]string, base *url.URL, raw string) {
	resolved := resolveAssetURL(base, raw)
	if resolved != "" {
		*assets = append(*assets, resolved)
	}
}

func appendResolvedSrcSet(assets *[]string, base *url.URL, raw string) {
	for _, candidate := range core.Split(raw, ",") {
		candidate = core.Trim(candidate)
		if candidate == "" {
			continue
		}
		fields := pwaFields(candidate)
		if len(fields) == 0 {
			continue
		}
		appendResolvedAsset(assets, base, fields[0])
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
	parsed, err := url.Parse(core.Trim(raw))
	if err != nil {
		return ""
	}
	parsed.Fragment = ""
	return parsed.String()
}

func resolveAssetDestination(destDir, assetURL string) core.Result {
	parsed, err := url.Parse(assetURL)
	if err != nil {
		return core.Fail(err)
	}

	relativePath := cleanPWAURLPath(core.Concat("/", parsed.Path))
	switch {
	case relativePath == "/" || relativePath == ".":
		relativePath = "/index.html"
	case core.HasSuffix(parsed.Path, "/"):
		relativePath = joinPWAURLPath(relativePath, "index.html")
	}

	return core.Ok(ax.Join(destDir, ax.FromSlash(core.TrimPrefix(relativePath, "/"))))
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
	parsed, err := url.Parse(assetURL)
	if err != nil {
		return ""
	}

	relativePath := cleanPWAURLPath(core.Concat("/", parsed.Path))
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
		case isPWAASCIILetter(r) || isPWAASCIIDigit(r):
			b.WriteRune(r)
			lastDash = false
		case isPWASpace(r) || r == '-' || r == '_' || r == '.':
			if b.Len() == 0 || lastDash {
				continue
			}
			b.WriteByte('-')
			lastDash = true
		}
	}

	slug := trimPWAHyphens(b.String())
	if slug == "" {
		return ""
	}
	if slug[0] >= '0' && slug[0] <= '9' {
		return core.Concat("app-", slug)
	}
	return slug
}

func cleanPWAURLPath(value string) string {
	return core.CleanPath(value, "/")
}

func joinPWAURLPath(parts ...string) string {
	return cleanPWAURLPath(core.Join("/", parts...))
}

func localPWAURLPath(relativePath string) string {
	return core.TrimPrefix(cleanPWAURLPath(core.Concat("/", core.Replace(relativePath, ax.DS(), "/"))), "/")
}

func pwaFields(value string) []string {
	fields := []string{}
	start := -1
	for i, r := range value {
		if isPWASpace(r) {
			if start >= 0 {
				fields = append(fields, value[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		fields = append(fields, value[start:])
	}
	return fields
}

func trimPWAHyphens(value string) string {
	for len(value) > 0 && value[0] == '-' {
		value = value[1:]
	}
	for len(value) > 0 && value[len(value)-1] == '-' {
		value = value[:len(value)-1]
	}
	return value
}

func isPWAASCIILetter(r rune) bool {
	return r >= 'a' && r <= 'z'
}

func isPWAASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isPWASpace(r rune) bool {
	return unicode.IsSpace(r)
}

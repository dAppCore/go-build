package build

import (
	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	"strings"
)

// Marker files for project type detection.
const (
	markerGoMod              = "go.mod"
	markerWails              = "wails.json"
	markerNodePackage        = "package.json"
	markerComposer           = "composer.json"
	markerMkDocs             = "mkdocs.yml"
	markerMkDocsYAML         = "mkdocs.yaml"
	markerDocsMkDocs         = "docs/mkdocs.yml"
	markerDocsMkDocsYAML     = "docs/mkdocs.yaml"
	markerPyProject          = "pyproject.toml"
	markerRequirements       = "requirements.txt"
	markerCargo              = "Cargo.toml"
	markerDockerfile         = "Dockerfile"
	markerFrontendPackage    = "frontend/package.json"
	markerFrontendDenoJSON   = "frontend/deno.json"
	markerFrontendDenoJSONC  = "frontend/deno.jsonc"
	markerLinuxKitYAML       = "linuxkit.yml"
	markerLinuxKitYAMLAlt    = "linuxkit.yaml"
	markerTaskfileYML        = "Taskfile.yml"
	markerTaskfileYAML       = "Taskfile.yaml"
	markerTaskfileBare       = "Taskfile"
	markerTaskfileLowerYML   = "taskfile.yml"
	markerTaskfileLowerYAML  = "taskfile.yaml"
	markerLinuxKitNestedYML  = ".core/linuxkit/*.yml"
	markerLinuxKitNestedYAML = ".core/linuxkit/*.yaml"
)

// projectMarker maps a marker file to its project type.
type projectMarker struct {
	file        string
	projectType ProjectType
}

// markers defines the detection order. More specific types come first.
// Wails projects have both wails.json and go.mod, so wails is checked first.
var markers = []projectMarker{
	{markerWails, ProjectTypeWails},
	{markerGoMod, ProjectTypeGo},
	{markerNodePackage, ProjectTypeNode},
	{markerComposer, ProjectTypePHP},
	{markerMkDocs, ProjectTypeDocs},
	{markerMkDocsYAML, ProjectTypeDocs},
	{markerPyProject, ProjectTypePython},
	{markerRequirements, ProjectTypePython},
	{markerCargo, ProjectTypeRust},
}

// Discover detects project types in the given directory by checking for marker files.
// Returns a slice of detected project types, ordered by priority (most specific first).
// For example, a Wails project returns [wails, go] since it has both wails.json and go.mod.
//
// types, err := build.Discover(io.Local, "/home/user/my-project") // → [go]
func Discover(fs io.Medium, dir string) ([]ProjectType, error) {
	var detected []ProjectType

	for _, m := range markers {
		path := ax.Join(dir, m.file)
		if fileExists(fs, path) {
			// Avoid duplicates (shouldn't happen with current markers, but defensive)
			if !core.NewArray(detected...).Contains(m.projectType) {
				detected = append(detected, m.projectType)
			}
		}
	}

	additionalTypes := []struct {
		projectType ProjectType
		detected    bool
	}{
		{ProjectTypeNode, IsNodeProject(fs, dir) || HasSubtreeNpm(fs, dir)},
		{ProjectTypeDocs, IsMkDocsProject(fs, dir)},
		{ProjectTypeDocker, IsDockerProject(fs, dir)},
		{ProjectTypeLinuxKit, IsLinuxKitProject(fs, dir)},
		{ProjectTypeCPP, IsCPPProject(fs, dir)},
		{ProjectTypeTaskfile, IsTaskfileProject(fs, dir)},
	}
	for _, candidate := range additionalTypes {
		if candidate.detected && !core.NewArray(detected...).Contains(candidate.projectType) {
			detected = append(detected, candidate.projectType)
		}
	}

	return detected, nil
}

// PrimaryType returns the most specific project type detected in the directory.
// Returns empty string if no project type is detected.
//
// pt, err := build.PrimaryType(io.Local, ".") // → "go"
func PrimaryType(fs io.Medium, dir string) (ProjectType, error) {
	types, err := Discover(fs, dir)
	if err != nil {
		return "", err
	}
	if len(types) == 0 {
		return "", nil
	}
	return types[0], nil
}

// IsGoProject checks if the directory contains a Go project (go.mod or wails.json).
//
// if build.IsGoProject(io.Local, ".") { ... }
func IsGoProject(fs io.Medium, dir string) bool {
	return fileExists(fs, ax.Join(dir, markerGoMod)) ||
		fileExists(fs, ax.Join(dir, markerWails))
}

// IsWailsProject checks if the directory contains a Wails project.
//
// if build.IsWailsProject(io.Local, ".") { ... }
func IsWailsProject(fs io.Medium, dir string) bool {
	return fileExists(fs, ax.Join(dir, markerWails))
}

// IsNodeProject checks if the directory contains a Node.js project.
//
// if build.IsNodeProject(io.Local, ".") { ... }
func IsNodeProject(fs io.Medium, dir string) bool {
	return fileExists(fs, ax.Join(dir, markerNodePackage))
}

// IsPHPProject checks if the directory contains a PHP project.
//
// if build.IsPHPProject(io.Local, ".") { ... }
func IsPHPProject(fs io.Medium, dir string) bool {
	return fileExists(fs, ax.Join(dir, markerComposer))
}

// IsCPPProject checks if the directory contains a C++ project (CMakeLists.txt).
//
// if build.IsCPPProject(io.Local, ".") { ... }
func IsCPPProject(fs io.Medium, dir string) bool {
	return fileExists(fs, ax.Join(dir, "CMakeLists.txt"))
}

// IsMkDocsProject checks for MkDocs config at the project root or in docs/.
//
//	ok := build.IsMkDocsProject(io.Local, ".")
func IsMkDocsProject(fs io.Medium, dir string) bool {
	return ResolveMkDocsConfigPath(fs, dir) != ""
}

// ResolveMkDocsConfigPath returns the first MkDocs config path that exists.
//
//	configPath := build.ResolveMkDocsConfigPath(io.Local, ".")
func ResolveMkDocsConfigPath(fs io.Medium, dir string) string {
	for _, path := range []string{
		ax.Join(dir, markerMkDocs),
		ax.Join(dir, markerMkDocsYAML),
		ax.Join(dir, "docs", "mkdocs.yml"),
		ax.Join(dir, "docs", "mkdocs.yaml"),
	} {
		if fileExists(fs, path) {
			return path
		}
	}
	return ""
}

// HasSubtreeNpm checks for package.json within depth 2 subdirectories.
// Ignores root package.json and node_modules directories.
// Returns true when a monorepo-style nested package.json is found.
//
//	ok := build.HasSubtreeNpm(io.Local, ".") // true if apps/web/package.json exists
func HasSubtreeNpm(fs io.Medium, dir string) bool {
	// Depth 1: list immediate subdirectories
	entries, err := fs.List(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "node_modules" {
			continue
		}

		subdir := ax.Join(dir, name)

		// Depth 1: check subdir/package.json
		if fileExists(fs, ax.Join(subdir, markerNodePackage)) {
			return true
		}

		// Depth 2: list subdirectories of subdir
		subEntries, err := fs.List(subdir)
		if err != nil {
			continue
		}
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() {
				continue
			}
			if subEntry.Name() == "node_modules" {
				continue
			}
			nested := ax.Join(subdir, subEntry.Name())
			if fileExists(fs, ax.Join(nested, markerNodePackage)) {
				return true
			}
		}
	}

	return false
}

// IsPythonProject checks for pyproject.toml or requirements.txt at the project root.
//
//	ok := build.IsPythonProject(io.Local, ".")
func IsPythonProject(fs io.Medium, dir string) bool {
	return fileExists(fs, ax.Join(dir, markerPyProject)) ||
		fileExists(fs, ax.Join(dir, markerRequirements))
}

// IsRustProject checks for Cargo.toml at the project root.
//
//	ok := build.IsRustProject(io.Local, ".")
func IsRustProject(fs io.Medium, dir string) bool {
	return fileExists(fs, ax.Join(dir, markerCargo))
}

// DiscoveryResult holds the full project analysis from DiscoverFull().
//
//	result, err := build.DiscoverFull(io.Local, ".")
//	fmt.Println(result.PrimaryStack) // "wails"
type DiscoveryResult struct {
	// Types lists all detected project types in priority order.
	Types []ProjectType
	// PrimaryStack is the best stack suggestion based on detected types.
	PrimaryStack string
	// HasFrontend is true when a root or frontend/ package.json/deno manifest is found,
	// or when a nested frontend tree is detected.
	HasFrontend bool
	// HasSubtreeNpm is true when a nested package.json exists within depth 2.
	HasSubtreeNpm bool
	// Markers records the presence of each raw marker file checked.
	Markers map[string]bool
	// Distro holds the detected Linux distribution version (e.g., "24.04").
	// Used by ComputeOptions to inject webkit2_41 tag on Ubuntu 24.04+.
	Distro string
}

// DiscoverFull returns a rich discovery result with all markers and metadata.
//
//	result, err := build.DiscoverFull(io.Local, ".")
//	if result.HasFrontend { ... }
func DiscoverFull(fs io.Medium, dir string) (*DiscoveryResult, error) {
	types, err := Discover(fs, dir)
	if err != nil {
		return nil, err
	}

	result := &DiscoveryResult{
		Types:   types,
		Markers: make(map[string]bool),
	}

	// Record raw marker presence
	allMarkers := []string{
		markerGoMod, markerWails, markerNodePackage, markerComposer,
		markerMkDocs, markerMkDocsYAML, markerDocsMkDocs, markerDocsMkDocsYAML,
		markerPyProject, markerRequirements, markerCargo,
		"CMakeLists.txt", markerDockerfile,
		markerFrontendPackage, markerFrontendDenoJSON, markerFrontendDenoJSONC,
		markerLinuxKitYAML, markerLinuxKitYAMLAlt,
		markerTaskfileYML, markerTaskfileYAML, markerTaskfileBare,
		markerTaskfileLowerYML, markerTaskfileLowerYAML,
	}
	for _, m := range allMarkers {
		result.Markers[m] = fileExists(fs, ax.Join(dir, m))
	}

	// Pattern-based marker: LinuxKit configs may live in .core/linuxkit/*.yml or *.yaml.
	result.Markers[markerLinuxKitNestedYML] = hasYAMLInDir(fs, ax.Join(dir, ".core", "linuxkit"))
	result.Markers[markerLinuxKitNestedYAML] = result.Markers[markerLinuxKitNestedYML]

	// Subtree npm detection
	result.HasSubtreeNpm = HasSubtreeNpm(fs, dir)

	// Frontend detection: root manifests, frontend/ manifests, or nested frontend trees.
	result.HasFrontend = hasFrontendManifest(fs, dir) ||
		hasFrontendManifest(fs, ax.Join(dir, "frontend")) ||
		hasSubtreeFrontendManifest(fs, dir) ||
		result.HasSubtreeNpm

	result.Types = types

	// Linux distro detection: used for distro-sensitive build flags.
	result.Distro = detectDistroVersion(fs)

	// Primary stack: first detected type as string, or empty
	if len(types) > 0 {
		result.PrimaryStack = string(types[0])
	}

	return result, nil
}

// hasFrontendManifest reports whether a frontend directory contains a supported manifest.
func hasFrontendManifest(fs io.Medium, dir string) bool {
	return fs.IsFile(ax.Join(dir, markerNodePackage)) ||
		fs.IsFile(ax.Join(dir, "deno.json")) ||
		fs.IsFile(ax.Join(dir, "deno.jsonc"))
}

// hasSubtreeFrontendManifest checks for package.json or deno.json within depth 2 subdirectories.
func hasSubtreeFrontendManifest(fs io.Medium, dir string) bool {
	entries, err := fs.List(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "node_modules" || strings.HasPrefix(name, ".") {
			continue
		}

		subdir := ax.Join(dir, name)
		if hasFrontendManifest(fs, subdir) {
			return true
		}

		subEntries, err := fs.List(subdir)
		if err != nil {
			continue
		}
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() {
				continue
			}
			if subEntry.Name() == "node_modules" || strings.HasPrefix(subEntry.Name(), ".") {
				continue
			}
			nested := ax.Join(subdir, subEntry.Name())
			if hasFrontendManifest(fs, nested) {
				return true
			}
		}
	}

	return false
}

// fileExists checks if a file exists and is not a directory.
func fileExists(fs io.Medium, path string) bool {
	return fs.IsFile(path)
}

// IsDockerProject checks if the directory contains a Dockerfile.
//
//	if build.IsDockerProject(io.Local, ".") { ... }
func IsDockerProject(fs io.Medium, dir string) bool {
	return fileExists(fs, ax.Join(dir, markerDockerfile))
}

// IsLinuxKitProject checks for linuxkit.yml or .core/linuxkit/*.yml.
//
//	ok := build.IsLinuxKitProject(io.Local, ".")
func IsLinuxKitProject(fs io.Medium, dir string) bool {
	if fileExists(fs, ax.Join(dir, markerLinuxKitYAML)) ||
		fileExists(fs, ax.Join(dir, markerLinuxKitYAMLAlt)) {
		return true
	}
	return hasYAMLInDir(fs, ax.Join(dir, ".core", "linuxkit"))
}

// IsTaskfileProject checks for supported Taskfile names in the project root.
//
//	ok := build.IsTaskfileProject(io.Local, ".")
func IsTaskfileProject(fs io.Medium, dir string) bool {
	for _, name := range []string{
		markerTaskfileYML,
		markerTaskfileYAML,
		markerTaskfileBare,
		markerTaskfileLowerYML,
		markerTaskfileLowerYAML,
	} {
		if fileExists(fs, ax.Join(dir, name)) {
			return true
		}
	}
	return false
}

// hasYAMLInDir reports whether a directory contains at least one YAML file.
func hasYAMLInDir(fs io.Medium, dir string) bool {
	if !fs.IsDir(dir) {
		return false
	}

	entries, err := fs.List(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			return true
		}
	}

	return false
}

// detectDistroVersion extracts the Ubuntu VERSION_ID from os-release data.
func detectDistroVersion(fs io.Medium) string {
	if fs == nil {
		return ""
	}

	for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		content, err := fs.Read(path)
		if err != nil {
			continue
		}

		if distro := parseOSReleaseDistro(content); distro != "" {
			return distro
		}
	}

	return ""
}

// parseOSReleaseDistro returns VERSION_ID for Ubuntu-style os-release content.
func parseOSReleaseDistro(content string) string {
	var id string
	var idLike string
	var version string

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		switch key {
		case "ID":
			id = value
		case "ID_LIKE":
			idLike = value
		case "VERSION_ID":
			version = value
		}
	}

	if version == "" {
		return ""
	}

	if id == "ubuntu" || strings.Contains(" "+idLike+" ", " ubuntu ") {
		return version
	}

	return ""
}

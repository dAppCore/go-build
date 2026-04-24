package build

import (
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	"dappco.re/go/core/io"
)

// Marker files for project type detection.
const (
	markerBuildConfig        = ".core/build.yaml"
	markerGoMod              = "go.mod"
	markerGoWork             = "go.work"
	markerMainGo             = "main.go"
	markerWails              = "wails.json"
	markerNodePackage        = "package.json"
	markerDenoJSON           = "deno.json"
	markerDenoJSONC          = "deno.jsonc"
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

type discoveryRule struct {
	projectType ProjectType
	matches     func(io.Medium, string) bool
}

func normalizeMedium(fs io.Medium) io.Medium {
	if fs == nil {
		return io.Local
	}
	return fs
}

var discoveryRules = []discoveryRule{
	{projectType: ProjectTypeWails, matches: IsWailsProject},
	{projectType: ProjectTypeGo, matches: func(fs io.Medium, dir string) bool {
		return fileExists(fs, ax.Join(dir, markerGoMod)) || fileExists(fs, ax.Join(dir, markerGoWork))
	}},
	{projectType: ProjectTypeNode, matches: IsNodeProject},
	{projectType: ProjectTypePHP, matches: IsPHPProject},
	{projectType: ProjectTypePython, matches: IsPythonProject},
	{projectType: ProjectTypeRust, matches: IsRustProject},
	{projectType: ProjectTypeCPP, matches: IsCPPProject},
	{projectType: ProjectTypeDocker, matches: IsDockerProject},
	{projectType: ProjectTypeLinuxKit, matches: IsLinuxKitProject},
	{projectType: ProjectTypeTaskfile, matches: IsTaskfileProject},
	{projectType: ProjectTypeDocs, matches: IsDocsProject},
}

var discoveryMarkerPaths = []string{
	markerBuildConfig,
	markerGoMod, markerGoWork, markerMainGo, markerWails, markerNodePackage, markerDenoJSON, markerDenoJSONC, markerComposer,
	markerMkDocs, markerMkDocsYAML, markerDocsMkDocs, markerDocsMkDocsYAML,
	markerPyProject, markerRequirements, markerCargo,
	"CMakeLists.txt", markerDockerfile, "Containerfile", "dockerfile", "containerfile",
	markerFrontendPackage, markerFrontendDenoJSON, markerFrontendDenoJSONC,
	markerLinuxKitYAML, markerLinuxKitYAMLAlt,
	markerTaskfileYML, markerTaskfileYAML, markerTaskfileBare,
	markerTaskfileLowerYML, markerTaskfileLowerYAML,
}

// Discover detects project types in the given directory by checking for marker files.
// Returns a slice of detected project types, ordered by priority (most specific first).
// For example, a Wails project returns [wails, go] since it has both wails.json and go.mod.
//
// types, err := build.Discover(io.Local, "/home/user/my-project") // → [go]
func Discover(fs io.Medium, dir string) ([]ProjectType, error) {
	fs = normalizeMedium(fs)
	var detected []ProjectType

	if configuredType, ok := configuredProjectType(fs, dir); ok {
		return []ProjectType{configuredType}, nil
	}

	appendType := func(projectType ProjectType, ok bool) {
		if !ok || core.NewArray(detected...).Contains(projectType) {
			return
		}
		detected = append(detected, projectType)
	}

	for _, rule := range discoveryRules {
		appendType(rule.projectType, rule.matches(fs, dir))
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

// IsGoProject checks if the directory contains a Go project (go.mod, go.work, or wails.json).
//
// if build.IsGoProject(io.Local, ".") { ... }
func IsGoProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return fileExists(fs, ax.Join(dir, markerGoMod)) ||
		fileExists(fs, ax.Join(dir, markerGoWork)) ||
		fileExists(fs, ax.Join(dir, markerWails))
}

// IsWailsProject checks if the directory contains a Wails project.
//
// if build.IsWailsProject(io.Local, ".") { ... }
func IsWailsProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	if fileExists(fs, ax.Join(dir, markerWails)) {
		return true
	}

	if !hasGoRootMarker(fs, dir) {
		return false
	}

	return hasFrontendManifest(fs, dir) ||
		hasFrontendManifest(fs, ax.Join(dir, "frontend")) ||
		hasSubtreeFrontendManifest(fs, dir)
}

// IsNodeProject checks if the directory contains a Node.js or Deno frontend
// project at the root, under frontend/, or in a visible nested subtree.
//
// if build.IsNodeProject(io.Local, ".") { ... }
func IsNodeProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return hasFrontendManifest(fs, dir) ||
		hasFrontendManifest(fs, ax.Join(dir, "frontend")) ||
		hasSubtreeFrontendManifest(fs, dir)
}

// IsPHPProject checks if the directory contains a PHP project.
//
// if build.IsPHPProject(io.Local, ".") { ... }
func IsPHPProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return fileExists(fs, ax.Join(dir, markerComposer))
}

// IsCPPProject checks if the directory contains a C++ project (CMakeLists.txt).
//
// if build.IsCPPProject(io.Local, ".") { ... }
func IsCPPProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return fileExists(fs, ax.Join(dir, "CMakeLists.txt"))
}

// IsMkDocsProject checks for MkDocs config at the project root or in docs/.
//
//	ok := build.IsMkDocsProject(io.Local, ".")
func IsMkDocsProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return ResolveMkDocsConfigPath(fs, dir) != ""
}

// IsDocsProject is the predictable alias for IsMkDocsProject.
//
//	ok := build.IsDocsProject(io.Local, ".")
func IsDocsProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return IsMkDocsProject(fs, dir)
}

// ResolveMkDocsConfigPath returns the first MkDocs config path that exists.
//
//	configPath := build.ResolveMkDocsConfigPath(io.Local, ".")
func ResolveMkDocsConfigPath(fs io.Medium, dir string) string {
	fs = normalizeMedium(fs)
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

	if path := findMkDocsConfigInSubtree(fs, dir, 0); path != "" {
		return path
	}

	return ""
}

// HasSubtreeNpm checks for package.json within depth 2 subdirectories.
// Ignores root package.json, the conventional frontend/ directory, hidden
// directories, and node_modules directories.
// Returns true when a monorepo-style nested package.json is found.
//
//	ok := build.HasSubtreeNpm(io.Local, ".") // true if apps/web/package.json exists
func HasSubtreeNpm(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
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
		if shouldSkipSubtreeDir(name) || name == "frontend" {
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
			if shouldSkipSubtreeDir(subEntry.Name()) {
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
	fs = normalizeMedium(fs)
	return fileExists(fs, ax.Join(dir, markerPyProject)) ||
		fileExists(fs, ax.Join(dir, markerRequirements))
}

// IsRustProject checks for Cargo.toml at the project root.
//
//	ok := build.IsRustProject(io.Local, ".")
func IsRustProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return fileExists(fs, ax.Join(dir, markerCargo))
}

// DiscoveryResult holds the full project analysis from DiscoverFull().
//
//	result, err := build.DiscoverFull(io.Local, ".")
//	fmt.Println(result.PrimaryStack) // "wails"
type DiscoveryResult struct {
	// Types lists all detected project types in priority order.
	Types []ProjectType
	// ConfiguredType is the explicit build.type override from .core/build.yaml when present.
	ConfiguredType string
	// ConfiguredBuildType mirrors the workflow-facing discovery output name.
	ConfiguredBuildType string
	// OS is the current host operating system for the discovery run.
	OS string
	// Arch is the current host architecture for the discovery run.
	Arch string
	// PrimaryStack is the best stack suggestion based on detected types.
	PrimaryStack string
	// SuggestedStack is the richer action-oriented stack hint derived from markers.
	// This preserves the v3 action naming where Wails projects map to "wails2".
	SuggestedStack string
	// HasFrontend is true when a root or frontend/ package.json/deno manifest is found,
	// or when a nested frontend tree is detected.
	HasFrontend bool
	// HasRootPackageJSON reports whether package.json exists at the project root.
	HasRootPackageJSON bool
	// HasFrontendPackageJSON reports whether frontend/package.json exists.
	HasFrontendPackageJSON bool
	// HasRootComposerJSON reports whether composer.json exists at the project root.
	HasRootComposerJSON bool
	// HasRootCargoToml reports whether Cargo.toml exists at the project root.
	HasRootCargoToml bool
	// HasRootGoMod reports whether go.mod exists at the project root.
	HasRootGoMod bool
	// HasRootGoWork reports whether go.work exists at the project root.
	HasRootGoWork bool
	// HasRootMainGo reports whether main.go exists at the project root.
	HasRootMainGo bool
	// HasRootCMakeLists reports whether CMakeLists.txt exists at the project root.
	HasRootCMakeLists bool
	// HasRootWailsJSON reports whether wails.json exists at the project root.
	HasRootWailsJSON bool
	// HasPackageJSON reports whether package.json exists at the root, in frontend/,
	// or in a supported nested subtree.
	HasPackageJSON bool
	// HasDenoManifest reports whether deno.json or deno.jsonc exists at the root,
	// in frontend/, or in a supported nested subtree.
	HasDenoManifest bool
	// HasTaskfile reports whether any supported Taskfile name exists at the project root.
	HasTaskfile bool
	// HasSubtreeNpm is true when a nested package.json exists within depth 2.
	HasSubtreeNpm bool
	// HasSubtreePackageJSON mirrors the workflow-facing discovery output name.
	HasSubtreePackageJSON bool
	// HasSubtreeDenoManifest is true when a nested Deno manifest exists within depth 2.
	HasSubtreeDenoManifest bool
	// HasDocsConfig reports whether MkDocs config exists at the root or under docs/.
	HasDocsConfig bool
	// HasGoToolchain reports whether Go markers exist at the root or in a visible
	// nested subtree, mirroring the action discovery contract used for setup.
	HasGoToolchain bool
	// PrimaryStackSuggestion mirrors the richer action output name and marker-based
	// precedence used by the generated workflow discovery step.
	PrimaryStackSuggestion string
	// LinuxPackages lists distro-aware system dependencies needed by the detected stack.
	LinuxPackages []string
	// WebKitPackage is the Ubuntu-aware WebKit dependency selected for Wails builds.
	WebKitPackage string
	// Markers records the presence of each raw marker file checked.
	Markers map[string]bool
	// Distro holds the detected Linux distribution version (e.g., "24.04").
	// Used by ComputeOptions to inject webkit2_41 tag on Ubuntu 24.04+.
	Distro string
	// Ref is the Git ref when discovery runs under GitHub metadata.
	Ref string
	// Branch is the branch name when available from GitHub metadata.
	Branch string
	// Tag is the tag name when available from GitHub metadata.
	Tag string
	// IsTag reports whether Ref points at a tag.
	IsTag bool
	// SHA is the current GitHub commit SHA when available.
	SHA string
	// ShortSHA is the short GitHub commit SHA when available.
	ShortSHA string
	// Repo is the GitHub owner/repo string when available.
	Repo string
	// Owner is the GitHub repository owner when available.
	Owner string
}

// DiscoverFull returns a rich discovery result with all markers and metadata.
//
//	result, err := build.DiscoverFull(io.Local, ".")
//	if result.HasFrontend { ... }
func DiscoverFull(fs io.Medium, dir string) (*DiscoveryResult, error) {
	fs = normalizeMedium(fs)
	types, err := Discover(fs, dir)
	if err != nil {
		return nil, err
	}

	result := &DiscoveryResult{
		Types:   types,
		OS:      discoverHostOS(),
		Arch:    discoverHostArch(),
		Markers: make(map[string]bool),
	}

	// Record raw marker presence
	result.Markers = collectMarkerPresence(fs, dir, discoveryMarkerPaths)

	result.HasRootPackageJSON = result.Markers[markerNodePackage]
	result.HasFrontendPackageJSON = result.Markers[markerFrontendPackage]
	result.HasRootComposerJSON = result.Markers[markerComposer]
	result.HasRootCargoToml = result.Markers[markerCargo]
	result.HasRootGoMod = result.Markers[markerGoMod]
	result.HasRootGoWork = result.Markers[markerGoWork]
	result.HasRootMainGo = result.Markers[markerMainGo]
	result.HasRootCMakeLists = result.Markers["CMakeLists.txt"]
	result.HasRootWailsJSON = result.Markers[markerWails]
	result.HasTaskfile = result.Markers[markerTaskfileYML] ||
		result.Markers[markerTaskfileYAML] ||
		result.Markers[markerTaskfileBare] ||
		result.Markers[markerTaskfileLowerYML] ||
		result.Markers[markerTaskfileLowerYAML]
	result.HasDocsConfig = IsMkDocsProject(fs, dir)

	// Pattern-based marker: LinuxKit configs may live in .core/linuxkit/*.yml or *.yaml.
	result.Markers[markerLinuxKitNestedYML] = hasYAMLInDir(fs, ax.Join(dir, ".core", "linuxkit"))
	result.Markers[markerLinuxKitNestedYAML] = result.Markers[markerLinuxKitNestedYML]

	// Subtree npm detection
	result.HasSubtreeNpm = HasSubtreeNpm(fs, dir)
	result.HasSubtreePackageJSON = result.HasSubtreeNpm
	result.HasSubtreeDenoManifest = hasSubtreeDenoManifest(fs, dir)
	result.HasPackageJSON = result.HasRootPackageJSON || result.HasFrontendPackageJSON || result.HasSubtreeNpm
	result.HasDenoManifest = result.Markers[markerDenoJSON] ||
		result.Markers[markerDenoJSONC] ||
		result.Markers[markerFrontendDenoJSON] ||
		result.Markers[markerFrontendDenoJSONC] ||
		result.HasSubtreeDenoManifest

	// Frontend detection: root manifests, frontend/ manifests, or nested frontend trees.
	result.HasFrontend = result.HasPackageJSON || result.HasDenoManifest
	result.HasGoToolchain = result.HasRootGoMod || result.HasRootGoWork || hasNestedGoToolchain(fs, dir, 0)

	result.Types = types
	if configuredType, ok := configuredProjectType(fs, dir); ok {
		result.ConfiguredType = string(configuredType)
		result.ConfiguredBuildType = result.ConfiguredType
	}

	// Linux distro detection: used for distro-sensitive build flags.
	result.Distro = detectDistroVersion(fs)
	result.LinuxPackages = ResolveLinuxPackages(result.Types, result.Distro)
	result.WebKitPackage = firstString(result.LinuxPackages)
	if git := DetectGitHubMetadata(); git != nil {
		result.Ref = git.Ref
		result.Branch = git.Branch
		result.Tag = git.Tag
		result.IsTag = git.IsTag
		result.SHA = git.SHA
		result.ShortSHA = git.ShortSHA
		result.Repo = git.Repo
		result.Owner = git.Owner
	} else if git := detectLocalGitMetadata(dir); git != nil {
		result.Ref = git.Ref
		result.Branch = git.Branch
		result.Tag = git.Tag
		result.IsTag = git.IsTag
		result.SHA = git.SHA
		result.ShortSHA = git.ShortSHA
		result.Repo = git.Repo
		result.Owner = git.Owner
	}

	// Primary stack: first detected type as string, or empty
	if len(types) > 0 {
		result.PrimaryStack = string(types[0])
	}
	result.SuggestedStack = SuggestStack(types)
	result.PrimaryStackSuggestion = resolvePrimaryStackSuggestion(result)

	return result, nil
}

func discoverHostOS() string {
	if goos := core.Env("GOOS"); goos != "" {
		return goos
	}

	if hosttype := core.Env("HOSTTYPE"); hosttype != "" {
		return hosttype
	}

	if ostype := core.Env("OSTYPE"); ostype != "" {
		return ostype
	}

	return "linux"
}

func discoverHostArch() string {
	if goarch := core.Env("GOARCH"); goarch != "" {
		return goarch
	}

	if hosttype := core.Env("HOSTTYPE"); hosttype != "" {
		switch hosttype {
		case "x86_64", "amd64":
			return "amd64"
		case "x86", "i386", "i686":
			return "386"
		case "aarch64", "arm64":
			return "arm64"
		case "arm", "armv7l", "armv6l":
			return "arm"
		case "riscv64":
			return "riscv64"
		}

		return hosttype
	}

	return "amd64"
}

// SuggestStack returns the action-oriented stack suggestion for the detected
// project markers. This keeps discovery compatible with the v3 action naming,
// where Wails-backed projects use the "wails2" stack identifier.
//
//	stack := build.SuggestStack([]build.ProjectType{build.ProjectTypeWails}) // "wails2"
func SuggestStack(types []ProjectType) string {
	if len(types) == 0 {
		return "unknown"
	}

	switch types[0] {
	case ProjectTypeWails:
		return "wails2"
	case ProjectTypeCPP:
		return "cpp"
	case ProjectTypeDocs:
		return "docs"
	case ProjectTypeNode:
		return "node"
	default:
		return string(types[0])
	}
}

func configuredProjectType(fs io.Medium, dir string) (ProjectType, bool) {
	if fs == nil || !ConfigExists(fs, dir) {
		return "", false
	}

	cfg, err := LoadConfig(fs, dir)
	if err != nil || cfg == nil {
		return "", false
	}

	projectType, ok := parseProjectType(cfg.Build.Type)
	if !ok {
		return "", false
	}

	return projectType, true
}

func parseProjectType(value string) (ProjectType, bool) {
	projectType := ProjectType(core.Lower(core.Trim(value)))

	switch projectType {
	case ProjectTypeGo,
		ProjectTypeWails,
		ProjectTypeNode,
		ProjectTypePHP,
		ProjectTypeCPP,
		ProjectTypeDocker,
		ProjectTypeLinuxKit,
		ProjectTypeTaskfile,
		ProjectTypeDocs,
		ProjectTypePython,
		ProjectTypeRust:
		return projectType, true
	default:
		return "", false
	}
}

// ResolveLinuxPackages returns distro-aware system dependencies for the detected stack.
//
//	packages := build.ResolveLinuxPackages([]build.ProjectType{build.ProjectTypeWails}, "24.04")
//	// []string{"libwebkit2gtk-4.1-dev"}
func ResolveLinuxPackages(types []ProjectType, distro string) []string {
	if len(types) == 0 || distro == "" {
		return nil
	}

	var packages []string
	if containsProjectType(types, ProjectTypeWails) {
		if isUbuntu2404OrNewer(distro) {
			packages = append(packages, "libwebkit2gtk-4.1-dev")
		} else {
			packages = append(packages, "libwebkit2gtk-4.0-dev")
		}
	}

	return deduplicateStrings(packages)
}

func containsProjectType(types []ProjectType, projectType ProjectType) bool {
	for _, candidate := range types {
		if candidate == projectType {
			return true
		}
	}
	return false
}

// hasFrontendManifest reports whether a frontend directory contains a supported manifest.
func hasFrontendManifest(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return fs.IsFile(ax.Join(dir, markerNodePackage)) ||
		fs.IsFile(ax.Join(dir, "deno.json")) ||
		fs.IsFile(ax.Join(dir, "deno.jsonc"))
}

// hasSubtreeFrontendManifest checks for package.json or deno.json within depth 2 subdirectories.
func hasSubtreeFrontendManifest(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	entries, err := fs.List(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if shouldSkipSubtreeDir(name) || name == "frontend" {
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
			if shouldSkipSubtreeDir(subEntry.Name()) {
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

func hasSubtreeDenoManifest(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return hasSubtreeManifest(fs, dir, 0, func(fs io.Medium, candidate string) bool {
		return fs.IsFile(ax.Join(candidate, markerDenoJSON)) || fs.IsFile(ax.Join(candidate, markerDenoJSONC))
	})
}

func findMkDocsConfigInSubtree(fs io.Medium, dir string, depth int) string {
	fs = normalizeMedium(fs)
	if depth >= 2 {
		return ""
	}

	entries, err := fs.List(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if shouldSkipSubtreeDir(name) {
			continue
		}

		candidateDir := ax.Join(dir, name)
		for _, marker := range []string{markerMkDocs, markerMkDocsYAML} {
			if fileExists(fs, ax.Join(candidateDir, marker)) {
				return ax.Join(candidateDir, marker)
			}
		}

		if nested := findMkDocsConfigInSubtree(fs, candidateDir, depth+1); nested != "" {
			return nested
		}
	}

	return ""
}

func hasNestedGoToolchain(fs io.Medium, dir string, depth int) bool {
	fs = normalizeMedium(fs)
	return hasSubtreeManifest(fs, dir, depth, func(fs io.Medium, candidate string) bool {
		return fs.IsFile(ax.Join(candidate, markerGoMod)) || fs.IsFile(ax.Join(candidate, markerGoWork))
	}, 4)
}

func hasSubtreeManifest(fs io.Medium, dir string, depth int, match func(io.Medium, string) bool, maxDepth ...int) bool {
	fs = normalizeMedium(fs)
	limit := 2
	if len(maxDepth) > 0 {
		limit = maxDepth[0]
	}
	if depth >= limit {
		return false
	}

	entries, err := fs.List(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if shouldSkipSubtreeDir(name) || name == "frontend" {
			continue
		}

		candidateDir := ax.Join(dir, name)
		if match(fs, candidateDir) {
			return true
		}

		if hasSubtreeManifest(fs, candidateDir, depth+1, match, limit) {
			return true
		}
	}

	return false
}

func resolvePrimaryStackSuggestion(result *DiscoveryResult) string {
	if result == nil {
		return "unknown"
	}
	if result.ConfiguredType != "" {
		return SuggestStack([]ProjectType{ProjectType(result.ConfiguredType)})
	}

	switch {
	case result.HasRootWailsJSON:
		return "wails2"
	case (result.HasRootGoMod || result.HasRootGoWork) && result.HasFrontend:
		return "wails2"
	case result.HasRootCMakeLists:
		return "cpp"
	case result.HasDocsConfig && !result.HasGoToolchain:
		return "docs"
	case result.HasFrontend && !result.HasGoToolchain:
		return "node"
	case result.HasGoToolchain:
		return "go"
	case result.HasDocsConfig:
		return "docs"
	case result.HasFrontend:
		return "node"
	default:
		return "unknown"
	}
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// hasGoRootMarker reports whether the project root contains a Go module or workspace marker.
func hasGoRootMarker(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return fileExists(fs, ax.Join(dir, markerGoMod)) ||
		fileExists(fs, ax.Join(dir, markerGoWork))
}

// fileExists checks if a file exists and is not a directory.
func fileExists(fs io.Medium, path string) bool {
	fs = normalizeMedium(fs)
	return fs.IsFile(path)
}

func collectMarkerPresence(fs io.Medium, dir string, paths []string) map[string]bool {
	fs = normalizeMedium(fs)
	markers := make(map[string]bool, len(paths))
	for _, path := range paths {
		markers[path] = fileExists(fs, ax.Join(dir, path))
	}
	return markers
}

func shouldSkipSubtreeDir(name string) bool {
	return name == "node_modules" || core.HasPrefix(name, ".")
}

// ResolveDockerfilePath returns the first Docker manifest path that exists.
//
//	dockerfile := build.ResolveDockerfilePath(io.Local, ".")
func ResolveDockerfilePath(fs io.Medium, dir string) string {
	fs = normalizeMedium(fs)
	for _, path := range []string{
		ax.Join(dir, "Dockerfile"),
		ax.Join(dir, "Containerfile"),
		ax.Join(dir, "dockerfile"),
		ax.Join(dir, "containerfile"),
	} {
		if fileExists(fs, path) {
			return path
		}
	}
	return ""
}

// IsDockerProject checks if the directory contains a Dockerfile or Containerfile.
//
//	if build.IsDockerProject(io.Local, ".") { ... }
func IsDockerProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
	return ResolveDockerfilePath(fs, dir) != ""
}

// IsLinuxKitProject checks for linuxkit.yml or .core/linuxkit/*.yml.
//
//	ok := build.IsLinuxKitProject(io.Local, ".")
func IsLinuxKitProject(fs io.Medium, dir string) bool {
	fs = normalizeMedium(fs)
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
	fs = normalizeMedium(fs)
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
	fs = normalizeMedium(fs)
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
		name := core.Lower(entry.Name())
		if core.HasSuffix(name, ".yml") || core.HasSuffix(name, ".yaml") {
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

	for _, line := range core.Split(content, "\n") {
		line = core.Trim(line)
		if line == "" || core.HasPrefix(line, "#") {
			continue
		}

		parts := core.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := core.Trim(parts[0])
		value := core.Trim(parts[1])
		value = core.TrimPrefix(value, `"`)
		value = core.TrimSuffix(value, `"`)
		value = core.TrimPrefix(value, `'`)
		value = core.TrimSuffix(value, `'`)

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

	if id == "ubuntu" || core.Contains(" "+idLike+" ", " ubuntu ") {
		return version
	}

	return ""
}

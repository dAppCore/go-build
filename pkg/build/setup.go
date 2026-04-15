package build

import (
	"sort"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
)

// SetupTool identifies a toolchain or installer surface required by the
// action-style setup phase.
type SetupTool string

const (
	// SetupToolGo installs the Go toolchain.
	SetupToolGo SetupTool = "go"
	// SetupToolGarble installs garble for obfuscated Go and Wails builds.
	SetupToolGarble SetupTool = "garble"
	// SetupToolTask installs the Task CLI for Taskfile-driven builds.
	SetupToolTask SetupTool = "task"
	// SetupToolNode installs Node.js/Corepack for frontend-backed builds.
	SetupToolNode SetupTool = "node"
	// SetupToolWails installs the Wails CLI for Wails-backed builds.
	SetupToolWails SetupTool = "wails"
	// SetupToolPython installs Python for Conan and MkDocs flows.
	SetupToolPython SetupTool = "python"
	// SetupToolPHP installs PHP for Composer-backed builds.
	SetupToolPHP SetupTool = "php"
	// SetupToolComposer installs Composer for PHP builds.
	SetupToolComposer SetupTool = "composer"
	// SetupToolRust installs Rust/Cargo for Rust builds.
	SetupToolRust SetupTool = "rust"
	// SetupToolConan installs Conan for C++ builds.
	SetupToolConan SetupTool = "conan"
	// SetupToolMkDocs installs MkDocs for docs builds.
	SetupToolMkDocs SetupTool = "mkdocs"
	// SetupToolDeno installs Deno for manifest-backed or override-driven builds.
	SetupToolDeno SetupTool = "deno"
)

// SetupStep describes one toolchain requirement in the setup plan.
type SetupStep struct {
	Tool   SetupTool `json:"tool"`
	Reason string    `json:"reason"`
}

// SetupPlan is the Go-side equivalent of the action setup orchestration.
// It is pure data: discovery + config in, setup requirements out.
type SetupPlan struct {
	ProjectDir             string
	PrimaryStack           string
	PrimaryStackSuggestion string
	FrontendDirs           []string
	LinuxPackages          []string
	Steps                  []SetupStep
}

// ComputeSetupPlan derives the action-style setup requirements from discovery
// and config. When discovery is nil the function performs a fresh DiscoverFull
// pass using the provided filesystem and directory.
func ComputeSetupPlan(fs io.Medium, dir string, cfg *BuildConfig, discovery *DiscoveryResult) (*SetupPlan, error) {
	if fs == nil {
		fs = io.Local
	}

	if discovery == nil {
		var err error
		discovery, err = DiscoverFull(fs, dir)
		if err != nil {
			return nil, err
		}
	}

	configuredType := resolveConfiguredBuildType(cfg, discovery)
	denoRequested := DenoRequested(configuredDenoBuild(cfg))
	hasTaskfile := configuredType == string(ProjectTypeTaskfile) || discovery.HasTaskfile || containsProjectType(discovery.Types, ProjectTypeTaskfile)
	hasWails := configuredType == string(ProjectTypeWails) || discovery.PrimaryStackSuggestion == "wails2"
	hasCPP := configuredType == string(ProjectTypeCPP) || containsProjectType(discovery.Types, ProjectTypeCPP) || discovery.HasRootCMakeLists
	hasDocs := configuredType == string(ProjectTypeDocs) || containsProjectType(discovery.Types, ProjectTypeDocs) || discovery.HasDocsConfig
	hasPHP := configuredType == string(ProjectTypePHP) || containsProjectType(discovery.Types, ProjectTypePHP) || discovery.HasRootComposerJSON
	hasRust := configuredType == string(ProjectTypeRust) || containsProjectType(discovery.Types, ProjectTypeRust) || discovery.HasRootCargoToml
	hasNode := configuredType == string(ProjectTypeNode) || discovery.HasPackageJSON || discovery.PrimaryStackSuggestion == "wails2"
	hasGo := configuredType == string(ProjectTypeGo) || hasWails || hasTaskfile || discovery.HasGoToolchain || containsProjectType(discovery.Types, ProjectTypeGo)

	plan := &SetupPlan{
		ProjectDir:             dir,
		PrimaryStack:           discovery.PrimaryStack,
		PrimaryStackSuggestion: discovery.PrimaryStackSuggestion,
		FrontendDirs:           ResolveFrontendSetupDirs(fs, dir, denoRequested),
		LinuxPackages:          append([]string{}, discovery.LinuxPackages...),
	}

	if hasGo {
		plan.addStep(SetupToolGo, "Go-backed build stack detected")
	}
	if cfg != nil && cfg.Build.Obfuscate {
		plan.addStep(SetupToolGarble, "build.obfuscate is enabled")
	}
	if hasTaskfile {
		plan.addStep(SetupToolTask, "Taskfile project detected")
	}
	if hasNode {
		plan.addStep(SetupToolNode, "frontend package manifests detected")
	}
	if hasWails {
		plan.addStep(SetupToolWails, "Wails stack detected")
	}
	if hasCPP || hasDocs {
		plan.addStep(SetupToolPython, "docs and C++ setup relies on Python tooling")
	}
	if hasPHP {
		plan.addStep(SetupToolPHP, "composer.json detected")
		plan.addStep(SetupToolComposer, "composer-backed build detected")
	}
	if hasRust {
		plan.addStep(SetupToolRust, "Cargo.toml detected")
	}
	if hasCPP {
		plan.addStep(SetupToolConan, "C++ stack detected")
	}
	if hasDocs {
		plan.addStep(SetupToolMkDocs, "MkDocs config detected")
	}
	if discovery.HasDenoManifest || denoRequested {
		plan.addStep(SetupToolDeno, "Deno manifest or override detected")
	}

	return plan, nil
}

// ResolveFrontendSetupDirs returns frontend directories that participate in the
// action-style setup phase. It checks the project root, `frontend/`, and then
// searches nested subtrees up to depth 2, ignoring hidden directories and
// node_modules. When allowEmptyFallback is true, the function falls back to an
// existing `frontend/` directory or the project root even if no manifest exists.
func ResolveFrontendSetupDirs(fs io.Medium, dir string, allowEmptyFallback bool) []string {
	if fs == nil {
		fs = io.Local
	}

	var dirs []string

	rootHasManifest := hasFrontendManifest(fs, dir)
	frontendDir := ax.Join(dir, "frontend")
	frontendHasManifest := fs.IsDir(frontendDir) && hasFrontendManifest(fs, frontendDir)

	if rootHasManifest {
		dirs = append(dirs, dir)
	}
	if frontendHasManifest {
		dirs = append(dirs, frontendDir)
	}

	collectFrontendSetupDirs(fs, dir, 0, &dirs)

	if len(dirs) == 0 && allowEmptyFallback {
		if fs.IsDir(frontendDir) {
			dirs = append(dirs, frontendDir)
		} else {
			dirs = append(dirs, dir)
		}
	}

	return deduplicateAndSortPaths(dirs)
}

func collectFrontendSetupDirs(fs io.Medium, dir string, depth int, dirs *[]string) {
	if depth >= 2 {
		return
	}

	entries, err := fs.List(dir)
	if err != nil {
		return
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
		if hasFrontendManifest(fs, candidateDir) {
			*dirs = append(*dirs, candidateDir)
		}

		collectFrontendSetupDirs(fs, candidateDir, depth+1, dirs)
	}
}

func deduplicateAndSortPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))

	for _, path := range paths {
		path = ax.Clean(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}

	sort.Strings(result)
	return result
}

func configuredDenoBuild(cfg *BuildConfig) string {
	if cfg == nil {
		return ""
	}
	return core.Trim(cfg.Build.DenoBuild)
}

func resolveConfiguredBuildType(cfg *BuildConfig, discovery *DiscoveryResult) string {
	if cfg != nil {
		if value := core.Lower(core.Trim(cfg.Build.Type)); value != "" {
			return value
		}
	}
	if discovery != nil {
		return core.Lower(core.Trim(discovery.ConfiguredType))
	}
	return ""
}

func (p *SetupPlan) addStep(tool SetupTool, reason string) {
	if p == nil {
		return
	}

	for _, step := range p.Steps {
		if step.Tool == tool {
			return
		}
	}

	p.Steps = append(p.Steps, SetupStep{
		Tool:   tool,
		Reason: reason,
	})
}

package build

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

func TestSetup_ComputeSetupPlan_Good(t *testing.T) {
	t.Run("wails monorepo adds Go Node Wails Garble and Linux packages", func(t *testing.T) {
		dir := t.TempDir()
		nestedFrontend := ax.Join(dir, "apps", "web")
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.MkdirAll(nestedFrontend, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nestedFrontend, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg := DefaultConfig()
		cfg.Build.Obfuscate = true

		discovery := &DiscoveryResult{
			Types:                  []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode},
			PrimaryStack:           "wails",
			PrimaryStackSuggestion: "wails2",
			HasGoToolchain:         true,
			HasPackageJSON:         true,
			LinuxPackages:          []string{"libwebkit2gtk-4.1-dev"},
		}

		plan, err := ComputeSetupPlan(io.Local, dir, cfg, discovery)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]SetupTool{SetupToolGo, SetupToolGarble, SetupToolNode, SetupToolWails}, setupTools(plan)) {
			t.Fatalf("want %v, got %v", []SetupTool{SetupToolGo, SetupToolGarble, SetupToolNode, SetupToolWails}, setupTools(plan))
		}
		if !stdlibAssertEqual([]string{nestedFrontend}, plan.FrontendDirs) {
			t.Fatalf("want %v, got %v", []string{nestedFrontend}, plan.FrontendDirs)
		}
		if !stdlibAssertEqual([]string{"libwebkit2gtk-4.1-dev"}, plan.LinuxPackages) {
			t.Fatalf("want %v, got %v", []string{"libwebkit2gtk-4.1-dev"}, plan.LinuxPackages)
		}

	})

	t.Run("docs plus package json keeps Node and adds Python plus MkDocs", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "mkdocs.yml"), []byte("site_name: Demo\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		discovery := &DiscoveryResult{
			Types:                  []ProjectType{ProjectTypeNode, ProjectTypeDocs},
			PrimaryStack:           "node",
			PrimaryStackSuggestion: "node",
			HasDocsConfig:          true,
			HasPackageJSON:         true,
		}

		plan, err := ComputeSetupPlan(io.Local, dir, DefaultConfig(), discovery)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]SetupTool{SetupToolNode, SetupToolPython, SetupToolMkDocs}, setupTools(plan)) {
			t.Fatalf("want %v, got %v", []SetupTool{SetupToolNode, SetupToolPython, SetupToolMkDocs}, setupTools(plan))
		}
		if !stdlibAssertEqual([]string{dir}, plan.FrontendDirs) {
			t.Fatalf("want %v, got %v", []string{dir}, plan.FrontendDirs)
		}

	})

	t.Run("cpp stack adds Python and Conan", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.20)\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		discovery := &DiscoveryResult{
			Types:                  []ProjectType{ProjectTypeCPP},
			PrimaryStack:           "cpp",
			PrimaryStackSuggestion: "cpp",
			HasRootCMakeLists:      true,
		}

		plan, err := ComputeSetupPlan(io.Local, dir, DefaultConfig(), discovery)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]SetupTool{SetupToolPython, SetupToolConan}, setupTools(plan)) {
			t.Fatalf("want %v, got %v", []SetupTool{SetupToolPython, SetupToolConan}, setupTools(plan))
		}
		if !stdlibAssertEmpty(plan.FrontendDirs) {
			t.Fatalf("expected empty, got %v", plan.FrontendDirs)
		}

	})

	t.Run("python stack adds Python tooling", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "pyproject.toml"), []byte("[project]\nname='demo'\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		discovery := &DiscoveryResult{
			Types:                  []ProjectType{ProjectTypePython},
			PrimaryStack:           "python",
			PrimaryStackSuggestion: "python",
		}

		plan, err := ComputeSetupPlan(io.Local, dir, DefaultConfig(), discovery)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]SetupTool{SetupToolPython}, setupTools(plan)) {
			t.Fatalf("want %v, got %v", []SetupTool{SetupToolPython}, setupTools(plan))
		}

	})

	t.Run("taskfile stack adds Go and Task even without go markers", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "Taskfile.yaml"), []byte("version: '3'\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		discovery := &DiscoveryResult{
			Types:                  []ProjectType{ProjectTypeTaskfile},
			PrimaryStack:           "taskfile",
			PrimaryStackSuggestion: "taskfile",
		}

		plan, err := ComputeSetupPlan(io.Local, dir, DefaultConfig(), discovery)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]SetupTool{SetupToolGo, SetupToolTask}, setupTools(plan)) {
			t.Fatalf("want %v, got %v", []SetupTool{SetupToolGo, SetupToolTask}, setupTools(plan))
		}

	})

	t.Run("configured wails stack adds Go Node and Wails without frontend markers", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.Build.Type = "wails"

		plan, err := ComputeSetupPlan(io.Local, dir, cfg, &DiscoveryResult{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]SetupTool{SetupToolGo, SetupToolNode, SetupToolWails}, setupTools(plan)) {
			t.Fatalf("want %v, got %v", []SetupTool{SetupToolGo, SetupToolNode, SetupToolWails}, setupTools(plan))
		}
		if !stdlibAssertEqual("wails", plan.PrimaryStack) {
			t.Fatalf("want %v, got %v", "wails", plan.PrimaryStack)
		}
		if !stdlibAssertEqual("wails2", plan.PrimaryStackSuggestion) {
			t.Fatalf("want %v, got %v", "wails2", plan.PrimaryStackSuggestion)
		}

	})

	t.Run("configured wails stack derives Linux packages from distro when discovery is partial", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.Build.Type = "wails"

		plan, err := ComputeSetupPlan(io.Local, dir, cfg, &DiscoveryResult{
			Distro: "24.04",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]string{"libwebkit2gtk-4.1-dev"}, plan.LinuxPackages) {
			t.Fatalf("want %v, got %v", []string{"libwebkit2gtk-4.1-dev"}, plan.LinuxPackages)
		}

	})

	t.Run("deno override enables Deno and fallback frontend dir", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.Build.DenoBuild = "deno task bundle"

		plan, err := ComputeSetupPlan(io.Local, dir, cfg, &DiscoveryResult{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]SetupTool{SetupToolDeno}, setupTools(plan)) {
			t.Fatalf("want %v, got %v", []SetupTool{SetupToolDeno}, setupTools(plan))
		}
		if !stdlibAssertEqual([]string{dir}, plan.FrontendDirs) {
			t.Fatalf("want %v, got %v", []string{dir}, plan.FrontendDirs)
		}

	})
}

func TestSetup_ResolveFrontendSetupDirs_Good(t *testing.T) {
	t.Run("returns root frontend and nested manifests in deterministic order", func(t *testing.T) {
		dir := t.TempDir()
		frontendDir := ax.Join(dir, "frontend")
		nestedA := ax.Join(dir, "apps", "alpha")
		nestedB := ax.Join(dir, "apps", "beta")
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.MkdirAll(nestedB, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nestedB, "deno.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.MkdirAll(nestedA, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nestedA, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]string{dir, nestedA, nestedB, frontendDir}, ResolveFrontendSetupDirs(io.Local, dir, false)) {
			t.Fatalf("want %v, got %v", []string{dir, nestedA, nestedB, frontendDir}, ResolveFrontendSetupDirs(io.Local, dir, false))
		}

	})

	t.Run("uses frontend fallback when deno is requested without manifests", func(t *testing.T) {
		dir := t.TempDir()
		frontendDir := ax.Join(dir, "frontend")
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]string{frontendDir}, ResolveFrontendSetupDirs(io.Local, dir, true)) {
			t.Fatalf("want %v, got %v", []string{frontendDir}, ResolveFrontendSetupDirs(io.Local, dir, true))
		}

	})
}

func setupTools(plan *SetupPlan) []SetupTool {
	if plan == nil {
		return nil
	}

	tools := make([]SetupTool, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		tools = append(tools, step.Tool)
	}
	return tools
}

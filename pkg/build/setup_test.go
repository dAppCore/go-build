package build

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetup_ComputeSetupPlan_Good(t *testing.T) {
	t.Run("wails monorepo adds Go Node Wails Garble and Linux packages", func(t *testing.T) {
		dir := t.TempDir()
		nestedFrontend := ax.Join(dir, "apps", "web")

		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/app\n"), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644))
		require.NoError(t, ax.MkdirAll(nestedFrontend, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(nestedFrontend, "package.json"), []byte("{}"), 0o644))

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
		require.NoError(t, err)

		assert.Equal(t, []SetupTool{
			SetupToolGo,
			SetupToolGarble,
			SetupToolNode,
			SetupToolWails,
		}, setupTools(plan))
		assert.Equal(t, []string{nestedFrontend}, plan.FrontendDirs)
		assert.Equal(t, []string{"libwebkit2gtk-4.1-dev"}, plan.LinuxPackages)
	})

	t.Run("docs plus package json keeps Node and adds Python plus MkDocs", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "mkdocs.yml"), []byte("site_name: Demo\n"), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644))

		discovery := &DiscoveryResult{
			Types:                  []ProjectType{ProjectTypeDocs, ProjectTypeNode},
			PrimaryStack:           "docs",
			PrimaryStackSuggestion: "docs",
			HasDocsConfig:          true,
			HasPackageJSON:         true,
		}

		plan, err := ComputeSetupPlan(io.Local, dir, DefaultConfig(), discovery)
		require.NoError(t, err)

		assert.Equal(t, []SetupTool{
			SetupToolNode,
			SetupToolPython,
			SetupToolMkDocs,
		}, setupTools(plan))
		assert.Equal(t, []string{dir}, plan.FrontendDirs)
	})

	t.Run("cpp stack adds Python and Conan", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.20)\n"), 0o644))

		discovery := &DiscoveryResult{
			Types:                  []ProjectType{ProjectTypeCPP},
			PrimaryStack:           "cpp",
			PrimaryStackSuggestion: "cpp",
			HasRootCMakeLists:      true,
		}

		plan, err := ComputeSetupPlan(io.Local, dir, DefaultConfig(), discovery)
		require.NoError(t, err)

		assert.Equal(t, []SetupTool{
			SetupToolPython,
			SetupToolConan,
		}, setupTools(plan))
		assert.Empty(t, plan.FrontendDirs)
	})

	t.Run("taskfile stack adds Go and Task even without go markers", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "Taskfile.yaml"), []byte("version: '3'\n"), 0o644))

		discovery := &DiscoveryResult{
			Types:                  []ProjectType{ProjectTypeTaskfile},
			PrimaryStack:           "taskfile",
			PrimaryStackSuggestion: "taskfile",
		}

		plan, err := ComputeSetupPlan(io.Local, dir, DefaultConfig(), discovery)
		require.NoError(t, err)

		assert.Equal(t, []SetupTool{
			SetupToolGo,
			SetupToolTask,
		}, setupTools(plan))
	})

	t.Run("configured wails stack adds Go Node and Wails without frontend markers", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.Build.Type = "wails"

		plan, err := ComputeSetupPlan(io.Local, dir, cfg, &DiscoveryResult{})
		require.NoError(t, err)

		assert.Equal(t, []SetupTool{
			SetupToolGo,
			SetupToolNode,
			SetupToolWails,
		}, setupTools(plan))
		assert.Equal(t, "wails", plan.PrimaryStack)
		assert.Equal(t, "wails2", plan.PrimaryStackSuggestion)
	})

	t.Run("configured wails stack derives Linux packages from distro when discovery is partial", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.Build.Type = "wails"

		plan, err := ComputeSetupPlan(io.Local, dir, cfg, &DiscoveryResult{
			Distro: "24.04",
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"libwebkit2gtk-4.1-dev"}, plan.LinuxPackages)
	})

	t.Run("deno override enables Deno and fallback frontend dir", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.Build.DenoBuild = "deno task bundle"

		plan, err := ComputeSetupPlan(io.Local, dir, cfg, &DiscoveryResult{})
		require.NoError(t, err)

		assert.Equal(t, []SetupTool{SetupToolDeno}, setupTools(plan))
		assert.Equal(t, []string{dir}, plan.FrontendDirs)
	})
}

func TestSetup_ResolveFrontendSetupDirs_Good(t *testing.T) {
	t.Run("returns root frontend and nested manifests in deterministic order", func(t *testing.T) {
		dir := t.TempDir()
		frontendDir := ax.Join(dir, "frontend")
		nestedA := ax.Join(dir, "apps", "alpha")
		nestedB := ax.Join(dir, "apps", "beta")

		require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644))
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte("{}"), 0o644))
		require.NoError(t, ax.MkdirAll(nestedB, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(nestedB, "deno.json"), []byte("{}"), 0o644))
		require.NoError(t, ax.MkdirAll(nestedA, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(nestedA, "package.json"), []byte("{}"), 0o644))

		assert.Equal(t, []string{dir, nestedA, nestedB, frontendDir}, ResolveFrontendSetupDirs(io.Local, dir, false))
	})

	t.Run("uses frontend fallback when deno is requested without manifests", func(t *testing.T) {
		dir := t.TempDir()
		frontendDir := ax.Join(dir, "frontend")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))

		assert.Equal(t, []string{frontendDir}, ResolveFrontendSetupDirs(io.Local, dir, true))
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

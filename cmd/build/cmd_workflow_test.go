package buildcmd

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
	"dappco.re/go/io"
)

func TestBuildCmd_resolveReleaseWorkflowOutputPathInput_Good(t *testing.T) {
	t.Run("accepts the preferred output path", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the snake_case output path alias", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("", "ci/release.yml", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the legacy output alias", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("", "", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts matching output aliases", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "ci/release.yml", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathInput_Bad(t *testing.T) {
	_, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "ops/release.yml", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "output aliases specify different locations")
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_Good(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "./ci/release.yml", "ci/release.yml", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_CamelCaseGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowCamelCaseGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "ci/release.yml", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowHyphenGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "", "ci/release.yml", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowSnakeGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "", "", "ci/release.yml", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_Bad(t *testing.T) {
	projectDir := t.TempDir()

	_, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "ops/release.yml", "", "", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "workflow output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "workflow output aliases specify different locations")
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_HyphenatedGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "ci/release.yml", "", "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_AbsoluteEquivalent_Good(t *testing.T) {
	projectDir := t.TempDir()
	absolutePath := ax.Join(projectDir, "ci", "release.yml")

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "", "", "", absolutePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(absolutePath, path) {
		t.Fatalf("want %v, got %v", absolutePath, path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_AbsoluteDirectory_Good(t *testing.T) {
	projectDir := t.TempDir()
	absoluteDir := ax.Join(projectDir, "ops")
	if err := io.Local.EnsureDir(absoluteDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", absoluteDir, "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(absoluteDir, "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(absoluteDir, "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_Good(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowInputPathAliases(projectDir, "ci/release.yml", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_WorkflowPathGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowInputPathAliases(projectDir, "", "ci/release.yml", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_Bad(t *testing.T) {
	projectDir := t.TempDir()

	_, err := resolveReleaseWorkflowInputPathAliases(projectDir, "ci/release.yml", "ops/release.yml", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "workflow path aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "workflow path aliases specify different locations")
	}

}

func TestBuildCmd_RunReleaseWorkflow_Good(t *testing.T) {
	projectDir := t.TempDir()

	t.Run("writes to the conventional workflow path by default", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path := build.ReleaseWorkflowPath(projectDir)
		content, err := io.Local.Read(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}
		if !stdlibAssertContains(content, "build:") {
			t.Fatalf("expected %v to contain %v", content, "build:")
		}
		if !stdlibAssertContains(content, "build-name:") {
			t.Fatalf("expected %v to contain %v", content, "build-name:")
		}
		if !stdlibAssertContains(content, "build-platform:") {
			t.Fatalf("expected %v to contain %v", content, "build-platform:")
		}
		if !stdlibAssertContains(content, "version:") {
			t.Fatalf("expected %v to contain %v", content, "version:")
		}
		if !stdlibAssertContains(content, "go-version:") {
			t.Fatalf("expected %v to contain %v", content, "go-version:")
		}
		if !stdlibAssertContains(content, "node-version:") {
			t.Fatalf("expected %v to contain %v", content, "node-version:")
		}
		if !stdlibAssertContains(content, "wails-version:") {
			t.Fatalf("expected %v to contain %v", content, "wails-version:")
		}
		if !stdlibAssertContains(content, "build-tags:") {
			t.Fatalf("expected %v to contain %v", content, "build-tags:")
		}
		if !stdlibAssertContains(content, "build-obfuscate:") {
			t.Fatalf("expected %v to contain %v", content, "build-obfuscate:")
		}
		if !stdlibAssertContains(content, "sign:") {
			t.Fatalf("expected %v to contain %v", content, "sign:")
		}
		if !stdlibAssertContains(content, "package:") {
			t.Fatalf("expected %v to contain %v", content, "package:")
		}
		if !stdlibAssertContains(content, "wails-build-webview2:") {
			t.Fatalf("expected %v to contain %v", content, "wails-build-webview2:")
		}
		if !stdlibAssertContains(content, "npm-build:") {
			t.Fatalf("expected %v to contain %v", content, "npm-build:")
		}
		if !stdlibAssertContains(content, "Discovery") {
			t.Fatalf("expected %v to contain %v", content, "Discovery")
		}
		if !stdlibAssertContains(content, "id: discovery") {
			t.Fatalf("expected %v to contain %v", content, "id: discovery")
		}
		if !stdlibAssertContains(content, "primary_stack_suggestion=wails2") {
			t.Fatalf("expected %v to contain %v", content, "primary_stack_suggestion=wails2")
		}
		if !stdlibAssertContains(content, "echo \"os=$runner_os\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"os=$runner_os\"")
		}
		if !stdlibAssertContains(content, "echo \"arch=$runner_arch\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"arch=$runner_arch\"")
		}
		if !stdlibAssertContains(content, "echo \"has_root_composer_json=$has_root_composer_json\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"has_root_composer_json=$has_root_composer_json\"")
		}
		if !stdlibAssertContains(content, "echo \"has_root_cargo_toml=$has_root_cargo_toml\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"has_root_cargo_toml=$has_root_cargo_toml\"")
		}
		if !stdlibAssertContains(content, "echo \"has_root_go_work=$has_root_go_work\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"has_root_go_work=$has_root_go_work\"")
		}
		if !stdlibAssertContains(content, "echo \"has_root_wails_json=$has_root_wails_json\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"has_root_wails_json=$has_root_wails_json\"")
		}
		if !stdlibAssertContains(content, "echo \"has_subtree_package_json=$has_subtree_package_json\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"has_subtree_package_json=$has_subtree_package_json\"")
		}
		if !stdlibAssertContains(content, "echo \"has_subtree_deno_manifest=$has_subtree_deno_manifest\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"has_subtree_deno_manifest=$has_subtree_deno_manifest\"")
		}
		if !stdlibAssertContains(content, "echo \"has_taskfile=$has_taskfile\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"has_taskfile=$has_taskfile\"")
		}
		if !stdlibAssertContains(content, "configured_build_type=\"\"") {
			t.Fatalf("expected %v to contain %v", content, "configured_build_type=\"\"")
		}
		if !stdlibAssertContains(content, "echo \"configured_build_type=$configured_build_type\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"configured_build_type=$configured_build_type\"")
		}
		if !stdlibAssertContains(content, "Setup Go") {
			t.Fatalf("expected %v to contain %v", content, "Setup Go")
		}
		if !stdlibAssertContains(content, "actions/setup-go@v5") {
			t.Fatalf("expected %v to contain %v", content, "actions/setup-go@v5")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.has_go_toolchain == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.has_go_toolchain == 'true'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.has_taskfile == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.has_taskfile == 'true'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'go'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'go'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'wails'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'wails'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'taskfile'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'taskfile'")
		}
		if !stdlibAssertContains(content, "Install Garble") {
			t.Fatalf("expected %v to contain %v", content, "Install Garble")
		}
		if !stdlibAssertContains(content, "mvdan.cc/garble@latest") {
			t.Fatalf("expected %v to contain %v", content, "mvdan.cc/garble@latest")
		}
		if !stdlibAssertContains(content, "Install Task CLI") {
			t.Fatalf("expected %v to contain %v", content, "Install Task CLI")
		}
		if !stdlibAssertContains(content, "github.com/go-task/task/v3/cmd/task@latest") {
			t.Fatalf("expected %v to contain %v", content, "github.com/go-task/task/v3/cmd/task@latest")
		}
		if !stdlibAssertContains(content, "Setup Node") {
			t.Fatalf("expected %v to contain %v", content, "Setup Node")
		}
		if !stdlibAssertContains(content, "actions/setup-node@v4") {
			t.Fatalf("expected %v to contain %v", content, "actions/setup-node@v4")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.has_package_json == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.has_package_json == 'true'")
		}
		if !stdlibAssertContains(content, "Enable Corepack") {
			t.Fatalf("expected %v to contain %v", content, "Enable Corepack")
		}
		if !stdlibAssertContains(content, "Install frontend dependencies") {
			t.Fatalf("expected %v to contain %v", content, "Install frontend dependencies")
		}
		if !stdlibAssertContains(content, "package_manager_from_manifest()") {
			t.Fatalf("expected %v to contain %v", content, "package_manager_from_manifest()")
		}
		if !stdlibAssertContains(content, "pkg.packageManager") {
			t.Fatalf("expected %v to contain %v", content, "pkg.packageManager")
		}
		if !stdlibAssertContains(content, "declared_manager=\"$(package_manager_from_manifest \"$dir\")\"") {
			t.Fatalf("expected %v to contain %v", content, "declared_manager=\"$(package_manager_from_manifest \"$dir\")\"")
		}
		if !stdlibAssertContains(content, "pnpm install --frozen-lockfile") {
			t.Fatalf("expected %v to contain %v", content, "pnpm install --frozen-lockfile")
		}
		if !stdlibAssertContains(content, "(cd \"$dir\" && pnpm install)") {
			t.Fatalf("expected %v to contain %v", content, "(cd \"$dir\" && pnpm install)")
		}
		if !stdlibAssertContains(content, "yarn install --immutable") {
			t.Fatalf("expected %v to contain %v", content, "yarn install --immutable")
		}
		if !stdlibAssertContains(content, "(cd \"$dir\" && yarn install)") {
			t.Fatalf("expected %v to contain %v", content, "(cd \"$dir\" && yarn install)")
		}
		if !stdlibAssertContains(content, "bun install --frozen-lockfile") {
			t.Fatalf("expected %v to contain %v", content, "bun install --frozen-lockfile")
		}
		if !stdlibAssertContains(content, "(cd \"$dir\" && bun install)") {
			t.Fatalf("expected %v to contain %v", content, "(cd \"$dir\" && bun install)")
		}
		if !stdlibAssertContains(content, "curl -fsSL https://bun.sh/install | bash") {
			t.Fatalf("expected %v to contain %v", content, "curl -fsSL https://bun.sh/install | bash")
		}
		if !stdlibAssertContains(content, "npm ci") {
			t.Fatalf("expected %v to contain %v", content, "npm ci")
		}
		if !stdlibAssertContains(content, "find_visible_files()") {
			t.Fatalf("expected %v to contain %v", content, "find_visible_files()")
		}
		if !stdlibAssertContains(content, "-path './.*'") {
			t.Fatalf("expected %v to contain %v", content, "-path './.*'")
		}
		if !stdlibAssertContains(content, "find_visible_files 3 -name package.json") {
			t.Fatalf("expected %v to contain %v", content, "find_visible_files 3 -name package.json")
		}
		if !stdlibAssertContains(content, "Install Wails CLI") {
			t.Fatalf("expected %v to contain %v", content, "Install Wails CLI")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'wails'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'wails'")
		}
		if !stdlibAssertContains(content, "Install Linux Wails dependencies") {
			t.Fatalf("expected %v to contain %v", content, "Install Linux Wails dependencies")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'cpp'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'cpp'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'docs'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'docs'")
		}
		if !stdlibAssertContains(content, "libwebkit2gtk-4.1-dev") {
			t.Fatalf("expected %v to contain %v", content, "libwebkit2gtk-4.1-dev")
		}
		if !stdlibAssertContains(content, "Setup PHP and Composer") {
			t.Fatalf("expected %v to contain %v", content, "Setup PHP and Composer")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.has_root_composer_json == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.has_root_composer_json == 'true'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'php'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'php'")
		}
		if !stdlibAssertContains(content, "composer-setup.php") {
			t.Fatalf("expected %v to contain %v", content, "composer-setup.php")
		}
		if !stdlibAssertContains(content, "Setup Rust") {
			t.Fatalf("expected %v to contain %v", content, "Setup Rust")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.has_root_cargo_toml == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.has_root_cargo_toml == 'true'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'rust'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'rust'")
		}
		if !stdlibAssertContains(content, "https://sh.rustup.rs") {
			t.Fatalf("expected %v to contain %v", content, "https://sh.rustup.rs")
		}
		if !stdlibAssertContains(content, "Install MkDocs") {
			t.Fatalf("expected %v to contain %v", content, "Install MkDocs")
		}
		if !stdlibAssertContains(content, "Setup Deno") {
			t.Fatalf("expected %v to contain %v", content, "Setup Deno")
		}
		if !stdlibAssertContains(content, "echo \"deno_requested=$deno_requested\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"deno_requested=$deno_requested\"")
		}
		if !stdlibAssertContains(content, "echo \"npm_requested=$npm_requested\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"npm_requested=$npm_requested\"")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.deno_requested == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.deno_requested == 'true'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.has_deno_manifest == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.has_deno_manifest == 'true'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.npm_requested == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.npm_requested == 'true'")
		}
		if !stdlibAssertContains(content, "build-cache:") {
			t.Fatalf("expected %v to contain %v", content, "build-cache:")
		}
		if !stdlibAssertContains(content, "Restore build cache") {
			t.Fatalf("expected %v to contain %v", content, "Restore build cache")
		}
		if !stdlibAssertContains(content, "actions/cache@v4") {
			t.Fatalf("expected %v to contain %v", content, "actions/cache@v4")
		}
		if !stdlibAssertContains(content, "inputs.build-platform == '' || inputs.build-platform == matrix.target") {
			t.Fatalf("expected %v to contain %v", content, "inputs.build-platform == '' || inputs.build-platform == matrix.target")
		}
		if !stdlibAssertContains(content, "--ci") {
			t.Fatalf("expected %v to contain %v", content, "--ci")
		}
		if !stdlibAssertContains(content, "--build-name") {
			t.Fatalf("expected %v to contain %v", content, "--build-name")
		}
		if !stdlibAssertContains(content, "--build-tags") {
			t.Fatalf("expected %v to contain %v", content, "--build-tags")
		}
		if !stdlibAssertContains(content, "--version") {
			t.Fatalf("expected %v to contain %v", content, "--version")
		}
		if !stdlibAssertContains(content, "--build-obfuscate") {
			t.Fatalf("expected %v to contain %v", content, "--build-obfuscate")
		}
		if !stdlibAssertContains(content, "--sign=true") {
			t.Fatalf("expected %v to contain %v", content, "--sign=true")
		}
		if !stdlibAssertContains(content, "--sign=false") {
			t.Fatalf("expected %v to contain %v", content, "--sign=false")
		}
		if !stdlibAssertContains(content, "--package=false") {
			t.Fatalf("expected %v to contain %v", content, "--package=false")
		}
		if !stdlibAssertContains(content, "--build-cache=false") {
			t.Fatalf("expected %v to contain %v", content, "--build-cache=false")
		}
		if !stdlibAssertContains(content, "--npm-build") {
			t.Fatalf("expected %v to contain %v", content, "--npm-build")
		}
		if !stdlibAssertContains(content, "--wails-build-webview2") {
			t.Fatalf("expected %v to contain %v", content, "--wails-build-webview2")
		}
		if !stdlibAssertContains(content, "--archive-format") {
			t.Fatalf("expected %v to contain %v", content, "--archive-format")
		}
		if !stdlibAssertContains(content, "Resolve build name") {
			t.Fatalf("expected %v to contain %v", content, "Resolve build name")
		}
		if !stdlibAssertContains(content, "Compute artifact upload name") {
			t.Fatalf("expected %v to contain %v", content, "Compute artifact upload name")
		}
		if !stdlibAssertContains(content, "build_name=\"${{ inputs.build-name }}\"") {
			t.Fatalf("expected %v to contain %v", content, "build_name=\"${{ inputs.build-name }}\"")
		}
		if !stdlibAssertContains(content, "if [ -z \"$build_name\" ] && [ -f .core/build.yaml ]; then") {
			t.Fatalf("expected %v to contain %v", content, "if [ -z \"$build_name\" ] && [ -f .core/build.yaml ]; then")
		}
		if !stdlibAssertContains(content, "in_project = stripped == \"project:\"") {
			t.Fatalf("expected %v to contain %v", content, "in_project = stripped == \"project:\"")
		}
		if !stdlibAssertContains(content, "build_name=\"${{ steps.build_name.outputs.value }}\"") {
			t.Fatalf("expected %v to contain %v", content, "build_name=\"${{ steps.build_name.outputs.value }}\"")
		}
		if !stdlibAssertContains(content, "build_name=\"${GITHUB_REPOSITORY##*/}\"") {
			t.Fatalf("expected %v to contain %v", content, "build_name=\"${GITHUB_REPOSITORY##*/}\"")
		}
		if !stdlibAssertContains(content, "suffix=\"${{ steps.discovery.outputs.short_sha }}\"") {
			t.Fatalf("expected %v to contain %v", content, "suffix=\"${{ steps.discovery.outputs.short_sha }}\"")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.is_tag") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.is_tag")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.tag") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.tag")
		}
		if !stdlibAssertContains(content, "name: ${{ steps.artifact-name.outputs.value }}") {
			t.Fatalf("expected %v to contain %v", content, "name: ${{ steps.artifact-name.outputs.value }}")
		}
		if !stdlibAssertContains(content, "if: ${{ inputs.package }}") {
			t.Fatalf("expected %v to contain %v", content, "if: ${{ inputs.package }}")
		}
		if !stdlibAssertContains(content, "echo \"value=${artifact_name}\" >> \"${GITHUB_OUTPUT}\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"value=${artifact_name}\" >> \"${GITHUB_OUTPUT}\"")
		}
		if stdlibAssertContains(content, "release-${artifact_name}") {
			t.Fatalf("expected %v not to contain %v", content, "release-${artifact_name}")
		}
		if !stdlibAssertContains(content, "if: ${{ inputs.build && inputs.package && startsWith(github.ref, 'refs/tags/') }}") {
			t.Fatalf("expected %v to contain %v", content, "if: ${{ inputs.build && inputs.package && startsWith(github.ref, 'refs/tags/') }}")
		}
		if !stdlibAssertContains(content, "actions/download-artifact@v4") {
			t.Fatalf("expected %v to contain %v", content, "actions/download-artifact@v4")
		}
		if stdlibAssertContains(content, "pattern: release-*") {
			t.Fatalf("expected %v not to contain %v", content, "pattern: release-*")
		}
		if !stdlibAssertContains(content, "command: ci") {
			t.Fatalf("expected %v to contain %v", content, "command: ci")
		}

	})

	t.Run("registers the build/workflow command", func(t *testing.T) {
		c := core.New()
		AddWorkflowCommand(c)

		result := c.Command("build/workflow")
		if !(result.OK) {
			t.Fatal("expected true")
		}

		command, ok := result.Value.(*core.Command)
		if !(ok) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("build/workflow", command.Path) {
			t.Fatalf("want %v, got %v", "build/workflow", command.Path)
		}
		if !stdlibAssertEqual("cmd.build.workflow.long", command.Description) {
			t.Fatalf("want %v, got %v", "cmd.build.workflow.long", command.Description)
		}

	})

	t.Run("writes to a custom relative path", func(t *testing.T) {
		customPath := "ci/release.yml"
		err := runReleaseWorkflowInDir(projectDir, customPath, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}
		if !stdlibAssertContains(content, "build:") {
			t.Fatalf("expected %v to contain %v", content, "build:")
		}
		if !stdlibAssertContains(content, "build-name:") {
			t.Fatalf("expected %v to contain %v", content, "build-name:")
		}
		if !stdlibAssertContains(content, "build-platform:") {
			t.Fatalf("expected %v to contain %v", content, "build-platform:")
		}
		if !stdlibAssertContains(content, "version:") {
			t.Fatalf("expected %v to contain %v", content, "version:")
		}
		if !stdlibAssertContains(content, "go-version:") {
			t.Fatalf("expected %v to contain %v", content, "go-version:")
		}
		if !stdlibAssertContains(content, "node-version:") {
			t.Fatalf("expected %v to contain %v", content, "node-version:")
		}
		if !stdlibAssertContains(content, "wails-version:") {
			t.Fatalf("expected %v to contain %v", content, "wails-version:")
		}
		if !stdlibAssertContains(content, "build-tags:") {
			t.Fatalf("expected %v to contain %v", content, "build-tags:")
		}
		if !stdlibAssertContains(content, "sign:") {
			t.Fatalf("expected %v to contain %v", content, "sign:")
		}
		if !stdlibAssertContains(content, "package:") {
			t.Fatalf("expected %v to contain %v", content, "package:")
		}
		if !stdlibAssertContains(content, "Discovery") {
			t.Fatalf("expected %v to contain %v", content, "Discovery")
		}
		if !stdlibAssertContains(content, "id: discovery") {
			t.Fatalf("expected %v to contain %v", content, "id: discovery")
		}
		if !stdlibAssertContains(content, "Setup Go") {
			t.Fatalf("expected %v to contain %v", content, "Setup Go")
		}
		if !stdlibAssertContains(content, "Install Garble") {
			t.Fatalf("expected %v to contain %v", content, "Install Garble")
		}
		if !stdlibAssertContains(content, "Install Task CLI") {
			t.Fatalf("expected %v to contain %v", content, "Install Task CLI")
		}
		if !stdlibAssertContains(content, "Setup Node") {
			t.Fatalf("expected %v to contain %v", content, "Setup Node")
		}
		if !stdlibAssertContains(content, "Enable Corepack") {
			t.Fatalf("expected %v to contain %v", content, "Enable Corepack")
		}
		if !stdlibAssertContains(content, "Install frontend dependencies") {
			t.Fatalf("expected %v to contain %v", content, "Install frontend dependencies")
		}
		if !stdlibAssertContains(content, "Install Wails CLI") {
			t.Fatalf("expected %v to contain %v", content, "Install Wails CLI")
		}
		if !stdlibAssertContains(content, "Install Linux Wails dependencies") {
			t.Fatalf("expected %v to contain %v", content, "Install Linux Wails dependencies")
		}
		if !stdlibAssertContains(content, "Setup PHP and Composer") {
			t.Fatalf("expected %v to contain %v", content, "Setup PHP and Composer")
		}
		if !stdlibAssertContains(content, "Setup Rust") {
			t.Fatalf("expected %v to contain %v", content, "Setup Rust")
		}
		if !stdlibAssertContains(content, "Install MkDocs") {
			t.Fatalf("expected %v to contain %v", content, "Install MkDocs")
		}
		if !stdlibAssertContains(content, "Setup Deno") {
			t.Fatalf("expected %v to contain %v", content, "Setup Deno")
		}
		if !stdlibAssertContains(content, "echo \"deno_requested=$deno_requested\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"deno_requested=$deno_requested\"")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.deno_requested == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.deno_requested == 'true'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.has_deno_manifest == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.has_deno_manifest == 'true'")
		}
		if !stdlibAssertContains(content, "build-cache:") {
			t.Fatalf("expected %v to contain %v", content, "build-cache:")
		}
		if !stdlibAssertContains(content, "Restore build cache") {
			t.Fatalf("expected %v to contain %v", content, "Restore build cache")
		}
		if !stdlibAssertContains(content, "--sign=false") {
			t.Fatalf("expected %v to contain %v", content, "--sign=false")
		}
		if !stdlibAssertContains(content, "--package=false") {
			t.Fatalf("expected %v to contain %v", content, "--package=false")
		}
		if !stdlibAssertContains(content, "--build-name") {
			t.Fatalf("expected %v to contain %v", content, "--build-name")
		}
		if !stdlibAssertContains(content, "--build-tags") {
			t.Fatalf("expected %v to contain %v", content, "--build-tags")
		}
		if !stdlibAssertContains(content, "--version") {
			t.Fatalf("expected %v to contain %v", content, "--version")
		}
		if !stdlibAssertContains(content, "--archive-format") {
			t.Fatalf("expected %v to contain %v", content, "--archive-format")
		}
		if !stdlibAssertContains(content, "Resolve build name") {
			t.Fatalf("expected %v to contain %v", content, "Resolve build name")
		}
		if !stdlibAssertContains(content, "Compute artifact upload name") {
			t.Fatalf("expected %v to contain %v", content, "Compute artifact upload name")
		}
		if !stdlibAssertContains(content, "build_name=\"${{ steps.build_name.outputs.value }}\"") {
			t.Fatalf("expected %v to contain %v", content, "build_name=\"${{ steps.build_name.outputs.value }}\"")
		}
		if !stdlibAssertContains(content, "suffix=\"${{ steps.discovery.outputs.short_sha }}\"") {
			t.Fatalf("expected %v to contain %v", content, "suffix=\"${{ steps.discovery.outputs.short_sha }}\"")
		}
		if !stdlibAssertContains(content, "name: ${{ steps.artifact-name.outputs.value }}") {
			t.Fatalf("expected %v to contain %v", content, "name: ${{ steps.artifact-name.outputs.value }}")
		}
		if !stdlibAssertContains(content, "echo \"value=${artifact_name}\" >> \"${GITHUB_OUTPUT}\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"value=${artifact_name}\" >> \"${GITHUB_OUTPUT}\"")
		}
		if stdlibAssertContains(content, "release-${artifact_name}") {
			t.Fatalf("expected %v not to contain %v", content, "release-${artifact_name}")
		}
		if !stdlibAssertContains(content, "actions/download-artifact@v4") {
			t.Fatalf("expected %v to contain %v", content, "actions/download-artifact@v4")
		}
		if stdlibAssertContains(content, "pattern: release-*") {
			t.Fatalf("expected %v not to contain %v", content, "pattern: release-*")
		}
		if !stdlibAssertContains(content, "command: ci") {
			t.Fatalf("expected %v to contain %v", content, "command: ci")
		}

	})

	t.Run("writes release.yml inside a directory-style relative path", func(t *testing.T) {
		customPath := "ci/"
		err := runReleaseWorkflowInDir(projectDir, customPath, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, "ci", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes release.yml inside an existing directory without a trailing slash", func(t *testing.T) {
		if err := io.Local.EnsureDir(ax.Join(projectDir, "ops")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := runReleaseWorkflowInDir(projectDir, "ops", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, "ops", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes release.yml inside a bare directory-style path", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, "ci", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes release.yml inside a current-directory-prefixed directory-style path", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "./ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, "ci", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes release.yml inside the conventional workflows directory", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, ".github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, ".github", "workflows", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes release.yml inside a current-directory-prefixed workflows directory", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "./.github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, ".github", "workflows", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes to the output alias", func(t *testing.T) {
		customPath := "ci/alias-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes to the output-path alias", func(t *testing.T) {
		customPath := "ci/output-path-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes to the output_path alias", func(t *testing.T) {
		customPath := "ci/output_path-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes to the workflow-output alias", func(t *testing.T) {
		customPath := "ci/workflow-output-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})

	t.Run("writes to the workflow_output alias", func(t *testing.T) {
		customPath := "ci/workflow_output-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "workflow_call:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_call:")
		}
		if !stdlibAssertContains(content, "workflow_dispatch:") {
			t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
		}

	})
}

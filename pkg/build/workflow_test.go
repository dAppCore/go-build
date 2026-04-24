package build

import (
	"strings"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core/io"
)

func TestWorkflow_WriteReleaseWorkflow_Good(t *testing.T) {
	t.Run("writes the embedded template to the default path", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read(DefaultReleaseWorkflowPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		template, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(string(template), content) {
			t.Fatalf("want %v, got %v", string(template), content)
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
		if !stdlibAssertEqual(2, strings.Count(content, "sign:\n        description: Enable platform signing after build.\n        required: false\n        type: boolean\n        default: false")) {
			t.Fatalf("want %v, got %v", 2, strings.Count(content, "sign:\n        description: Enable platform signing after build.\n        required: false\n        type: boolean\n        default: false"))
		}
		if !stdlibAssertContains(content, "package:") {
			t.Fatalf("expected %v to contain %v", content, "package:")
		}
		if !stdlibAssertContains(content, "wails-build-webview2:") {
			t.Fatalf("expected %v to contain %v", content, "wails-build-webview2:")
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
		if !stdlibAssertContains(content, "echo \"short_sha=$short_sha\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"short_sha=$short_sha\"")
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
		if !stdlibAssertContains(content, "match = re.match(r\"^\\s*type:\\s*(.+?)\\s*$\", line)") {
			t.Fatalf("expected %v to contain %v", content, "match = re.match(r\"^\\s*type:\\s*(.+?)\\s*$\", line)")
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
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'taskfile'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'taskfile'")
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
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'node'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'node'")
		}
		if !stdlibAssertContains(content, "Enable Corepack") {
			t.Fatalf("expected %v to contain %v", content, "Enable Corepack")
		}
		if !stdlibAssertContains(content, "corepack enable") {
			t.Fatalf("expected %v to contain %v", content, "corepack enable")
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
		if !stdlibAssertContains(content, "github.com/wailsapp/wails/v3/cmd/wails3") {
			t.Fatalf("expected %v to contain %v", content, "github.com/wailsapp/wails/v3/cmd/wails3")
		}
		if !stdlibAssertContains(content, "github.com/wailsapp/wails/v2/cmd/wails") {
			t.Fatalf("expected %v to contain %v", content, "github.com/wailsapp/wails/v2/cmd/wails")
		}
		if !stdlibAssertContains(content, "Setup Python for Conan and MkDocs") {
			t.Fatalf("expected %v to contain %v", content, "Setup Python for Conan and MkDocs")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.has_root_cmakelists == 'true' || steps.discovery.outputs.has_docs_config == 'true'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.has_root_cmakelists == 'true' || steps.discovery.outputs.has_docs_config == 'true'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'cpp'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'cpp'")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.configured_build_type == 'docs'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.configured_build_type == 'docs'")
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
		if !stdlibAssertContains(content, "choco install rustup.install -y") {
			t.Fatalf("expected %v to contain %v", content, "choco install rustup.install -y")
		}
		if !stdlibAssertContains(content, "Install Conan") {
			t.Fatalf("expected %v to contain %v", content, "Install Conan")
		}
		if !stdlibAssertContains(content, "Install MkDocs") {
			t.Fatalf("expected %v to contain %v", content, "Install MkDocs")
		}
		if !stdlibAssertContains(content, "actions/setup-python@v5") {
			t.Fatalf("expected %v to contain %v", content, "actions/setup-python@v5")
		}
		if !stdlibAssertContains(content, "python -m pip install conan") {
			t.Fatalf("expected %v to contain %v", content, "python -m pip install conan")
		}
		if !stdlibAssertContains(content, "python -m pip install mkdocs") {
			t.Fatalf("expected %v to contain %v", content, "python -m pip install mkdocs")
		}
		if !stdlibAssertContains(content, "Setup Deno") {
			t.Fatalf("expected %v to contain %v", content, "Setup Deno")
		}
		if !stdlibAssertContains(content, "denoland/setup-deno@v2") {
			t.Fatalf("expected %v to contain %v", content, "denoland/setup-deno@v2")
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
		if !stdlibAssertContains(content, "actions/cache@v4") {
			t.Fatalf("expected %v to contain %v", content, "actions/cache@v4")
		}
		if !stdlibAssertContains(content, "${{ inputs.working-directory }}/.core/cache") {
			t.Fatalf("expected %v to contain %v", content, "${{ inputs.working-directory }}/.core/cache")
		}
		if !stdlibAssertContains(content, "core-build-${{ runner.os }}-${{ matrix.target }}-") {
			t.Fatalf("expected %v to contain %v", content, "core-build-${{ runner.os }}-${{ matrix.target }}-")
		}
		if !stdlibAssertContains(content, "Install Linux Wails dependencies") {
			t.Fatalf("expected %v to contain %v", content, "Install Linux Wails dependencies")
		}
		if !stdlibAssertContains(content, "steps.discovery.outputs.primary_stack_suggestion == 'wails2'") {
			t.Fatalf("expected %v to contain %v", content, "steps.discovery.outputs.primary_stack_suggestion == 'wails2'")
		}
		if !stdlibAssertContains(content, "runner.os == 'Linux' && (steps.discovery.outputs.primary_stack_suggestion == 'wails2' || steps.discovery.outputs.configured_build_type == 'wails')") {
			t.Fatalf("expected %v to contain %v", content, "runner.os == 'Linux' && (steps.discovery.outputs.primary_stack_suggestion == 'wails2' || steps.discovery.outputs.configured_build_type == 'wails')")
		}
		if !stdlibAssertContains(content, "libwebkit2gtk-4.0-dev") {
			t.Fatalf("expected %v to contain %v", content, "libwebkit2gtk-4.0-dev")
		}
		if !stdlibAssertContains(content, "libwebkit2gtk-4.1-dev") {
			t.Fatalf("expected %v to contain %v", content, "libwebkit2gtk-4.1-dev")
		}
		if !stdlibAssertContains(content, "dpkg --compare-versions") {
			t.Fatalf("expected %v to contain %v", content, "dpkg --compare-versions")
		}
		if !stdlibAssertContains(content, "core build --ci --targets") {
			t.Fatalf("expected %v to contain %v", content, "core build --ci --targets")
		}
		if !stdlibAssertContains(content, "--ci") {
			t.Fatalf("expected %v to contain %v", content, "--ci")
		}
		if !stdlibAssertContains(content, "inputs.build-platform == '' || inputs.build-platform == matrix.target") {
			t.Fatalf("expected %v to contain %v", content, "inputs.build-platform == '' || inputs.build-platform == matrix.target")
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
		if !stdlibAssertContains(content, "echo \"value=${build_name}\" >> \"${GITHUB_OUTPUT}\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"value=${build_name}\" >> \"${GITHUB_OUTPUT}\"")
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
		if !stdlibAssertContains(content, "echo \"value=${artifact_name}\" >> \"${GITHUB_OUTPUT}\"") {
			t.Fatalf("expected %v to contain %v", content, "echo \"value=${artifact_name}\" >> \"${GITHUB_OUTPUT}\"")
		}
		if stdlibAssertContains(content, "release-${artifact_name}") {
			t.Fatalf("expected %v not to contain %v", content, "release-${artifact_name}")
		}
		if !stdlibAssertContains(content, "if: ${{ inputs.package }}") {
			t.Fatalf("expected %v to contain %v", content, "if: ${{ inputs.package }}")
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
		if !stdlibAssertContains(content, "we-are-go-for-launch: true") {
			t.Fatalf("expected %v to contain %v", content, "we-are-go-for-launch: true")
		}
		if !stdlibAssertContains(content, "uses: dAppCore/build@v3") {
			t.Fatalf("expected %v to contain %v", content, "uses: dAppCore/build@v3")
		}

	})

	t.Run("writes to a custom path", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "custom/workflow.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read("custom/workflow.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(content) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("trims surrounding whitespace from the output path", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "  ci  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read("ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(content) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("writes release.yml for a bare directory-style path", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "ci")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read("ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(content) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("writes release.yml inside an existing directory", func(t *testing.T) {
		projectDir := t.TempDir()
		outputDir := ax.Join(projectDir, "ci")
		if err := ax.MkdirAll(outputDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := WriteReleaseWorkflow(io.Local, outputDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(outputDir, DefaultReleaseWorkflowFileName))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		template, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(string(template), content) {
			t.Fatalf("want %v, got %v", string(template), content)
		}

	})

	t.Run("writes release.yml for directory-style output paths", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "ci/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read("ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(content) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("creates parent directories on a real filesystem", func(t *testing.T) {
		projectDir := t.TempDir()
		path := ax.Join(projectDir, ".github", "workflows", "release.yml")

		err := WriteReleaseWorkflow(io.Local, path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		template, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(string(template), content) {
			t.Fatalf("want %v, got %v", string(template), content)
		}

	})
}

func TestWorkflow_WriteReleaseWorkflow_Bad(t *testing.T) {
	t.Run("rejects a nil filesystem medium", func(t *testing.T) {
		err := WriteReleaseWorkflow(nil, "")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "filesystem medium is required") {
			t.Fatalf("expected %v to contain %v", err.Error(), "filesystem medium is required")
		}

	})
}

func TestWorkflow_ReleaseWorkflowPath_Good(t *testing.T) {
	if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", ReleaseWorkflowPath("/tmp/project")) {
		t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", ReleaseWorkflowPath("/tmp/project"))
	}

}

func TestWorkflow_ResolveReleaseWorkflowOutputPathWithMedium_Good(t *testing.T) {
	t.Run("treats an existing directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path := ResolveReleaseWorkflowOutputPathWithMedium(fs, "/tmp/project", "ci")
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("keeps explicit file paths unchanged", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path := ResolveReleaseWorkflowOutputPathWithMedium(fs, "/tmp/project", "ci/release.yml")
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowPath_Good(t *testing.T) {
	t.Run("uses the conventional path when empty", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "")) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ""))
		}

	})

	t.Run("joins relative paths to the project directory", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/release.yml")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/release.yml"))
		}

	})

	t.Run("treats bare relative directory names as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci"))
		}

	})

	t.Run("treats current-directory-prefixed directory names as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./ci")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./ci"))
		}

	})

	t.Run("treats the conventional workflows directory as a directory", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ".github/workflows")) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ".github/workflows"))
		}

	})

	t.Run("treats current-directory-prefixed workflows directories as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./.github/workflows")) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./.github/workflows"))
		}

	})

	t.Run("keeps nested extensionless paths as files", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release", ResolveReleaseWorkflowPath("/tmp/project", "ci/release")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release", ResolveReleaseWorkflowPath("/tmp/project", "ci/release"))
		}

	})

	t.Run("treats the current directory as a workflow directory", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ".")) {
			t.Fatalf("want %v, got %v", "/tmp/project/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "."))
		}

	})

	t.Run("treats trailing-slash relative paths as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/"))
		}

	})

	t.Run("keeps absolute paths unchanged", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/release.yml")) {
			t.Fatalf("want %v, got %v", "/tmp/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/release.yml"))
		}

	})

	t.Run("treats trailing-slash absolute paths as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/workflows/")) {
			t.Fatalf("want %v, got %v", "/tmp/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/workflows/"))
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPath_Good(t *testing.T) {
	t.Run("uses the conventional path when both inputs are empty", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("accepts path as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts bare directory-style path as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts current-directory-prefixed directory-style path as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "./ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts the conventional workflows directory as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", ".github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("accepts current-directory-prefixed workflows directories as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "./.github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("keeps nested extensionless paths as files", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release", path)
		}

	})

	t.Run("accepts the current directory as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", ".", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/release.yml", path)
		}

	})

	t.Run("accepts output as an alias for path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("trims surrounding whitespace from inputs", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "  ci  ", "  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts matching path and output values", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts matching directory-style path and output values", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/", "ci/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPath_Bad(t *testing.T) {
	t.Run("rejects conflicting path and output values", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "ops/release.yml")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(path) {
			t.Fatalf("expected empty, got %v", path)
		}
		if !stdlibAssertContains(err.Error(), "path and output specify different locations") {
			t.Fatalf("expected %v to contain %v", err.Error(), "path and output specify different locations")
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPathWithMedium_Good(t *testing.T) {
	t.Run("treats an existing directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("treats a bare directory-style path as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("treats a current-directory-prefixed directory-style path as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "./ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("treats the conventional workflows directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", ".github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("treats current-directory-prefixed workflows directories as workflow directories", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "./.github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("keeps a file path unchanged when the target is not a directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci/release.yml", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("normalizes matching directory aliases", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci", "ci/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("trims surrounding whitespace before resolving", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "  ci  ", "  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPathAliases_Good(t *testing.T) {
	t.Run("accepts the preferred path input", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "ci", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts the workflowPath alias", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "", "ci", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts the workflow_path alias", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "", "", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts the workflow-path alias", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "", "", "", "ci")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("normalises matching aliases", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "ci/", "./ci", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPathAliases_Bad(t *testing.T) {
	fs := io.NewMemoryMedium()

	path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "ci/release.yml", "ops/release.yml", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(path) {
		t.Fatalf("expected empty, got %v", path)
	}
	if !stdlibAssertContains(err.Error(), "path aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "path aliases specify different locations")
	}

}

func TestWorkflow_ResolveReleaseWorkflowOutputPath_Good(t *testing.T) {
	t.Run("accepts the preferred output path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the snake_case output path alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("", "ci/release.yml", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the hyphenated output path alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "ci/release.yml", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the legacy output alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("", "", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("trims surrounding whitespace from aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("  ci/release.yml  ", "  ", "  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts matching aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "ci/release.yml", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("normalises equivalent path aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "./ci/release.yml", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowOutputPath_Bad(t *testing.T) {
	path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "ops/release.yml", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(path) {
		t.Fatalf("expected empty, got %v", path)
	}
	if !stdlibAssertContains(err.Error(), "output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "output aliases specify different locations")
	}

}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliases_Good(t *testing.T) {
	t.Run("accepts workflowOutputPath aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "ci/release.yml", "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the hyphenated workflowOutputPath alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "", "ci/release.yml", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the workflow_output alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "ci/release.yml", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the workflow-output alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "", "ci/release.yml", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("normalises matching workflow output aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("ci/release.yml", "", "", "./ci/release.yml", "ci/release.yml", "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliasesInProject_Good(t *testing.T) {
	projectDir := t.TempDir()
	absolutePath := ax.Join(projectDir, "ci", "release.yml")
	absoluteDirectory := ax.Join(projectDir, "ops")
	if err := ax.MkdirAll(absoluteDirectory, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("accepts the preferred output path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "ci/release.yml", "", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(absolutePath, path) {
			t.Fatalf("want %v, got %v", absolutePath, path)
		}

	})

	t.Run("accepts an absolute workflow output alias equivalent to the project path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "", "", "", "", absolutePath, "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(absolutePath, path) {
			t.Fatalf("want %v, got %v", absolutePath, path)
		}

	})

	t.Run("accepts matching relative and absolute aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "ci/release.yml", "", "", "", "", "", "", "", absolutePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(absolutePath, path) {
			t.Fatalf("want %v, got %v", absolutePath, path)
		}

	})

	t.Run("treats an existing absolute directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir(absoluteDirectory); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(fs, projectDir, "", "", "", "", absoluteDirectory, "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ax.Join(absoluteDirectory, DefaultReleaseWorkflowFileName), path) {
			t.Fatalf("want %v, got %v", ax.Join(absoluteDirectory, DefaultReleaseWorkflowFileName), path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliasesInProject_Bad(t *testing.T) {
	projectDir := t.TempDir()

	path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "ci/release.yml", "", "", "", "", "", "", "", ax.Join(projectDir, "ops", "release.yml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(path) {
		t.Fatalf("expected empty, got %v", path)
	}
	if !stdlibAssertContains(err.Error(), "output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "output aliases specify different locations")
	}

}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliases_Bad(t *testing.T) {
	path, err := ResolveReleaseWorkflowOutputPathAliases("ci/release.yml", "", "", "", "ops/release.yml", "", "", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(path) {
		t.Fatalf("expected empty, got %v", path)
	}
	if !stdlibAssertContains(err.Error(), "output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "output aliases specify different locations")
	}

}

// Package buildtest contains shared checks for build package tests.
package buildtest

import (
	"strings"
	"testing"

	"dappco.re/go/build/internal/testassert"
)

// AssertReleaseWorkflowContent verifies the release workflow template's core contract.
//
// buildtest.AssertReleaseWorkflowContent(t, generatedWorkflow)
func AssertReleaseWorkflowContent(t testing.TB, content string) {
	t.Helper()

	AssertReleaseWorkflowTriggers(t, content)

	for _, expected := range releaseWorkflowExpectedMarkers {
		if !testassert.Contains(content, expected) {
			t.Fatalf("expected %v to contain %v", content, expected)
		}
	}

	for _, unexpected := range releaseWorkflowForbiddenMarkers {
		if testassert.Contains(content, unexpected) {
			t.Fatalf("expected %v not to contain %v", content, unexpected)
		}
	}

	signInput := "sign:\n        description: Enable platform signing after build.\n        required: false\n        type: boolean\n        default: false"
	if !testassert.Equal(2, strings.Count(content, signInput)) {
		t.Fatalf("want %v, got %v", 2, strings.Count(content, signInput))
	}
}

// AssertReleaseWorkflowTriggers verifies that a generated workflow exposes both trigger modes.
//
// buildtest.AssertReleaseWorkflowTriggers(t, generatedWorkflow)
func AssertReleaseWorkflowTriggers(t testing.TB, content string) {
	t.Helper()

	for _, expected := range []string{"workflow_call:", "workflow_dispatch:"} {
		if !testassert.Contains(content, expected) {
			t.Fatalf("expected %v to contain %v", content, expected)
		}
	}
}

var releaseWorkflowExpectedMarkers = []string{
	"build:",
	"build-name:",
	"build-platform:",
	"version:",
	"go-version:",
	"node-version:",
	"wails-version:",
	"build-tags:",
	"build-obfuscate:",
	"sign:",
	"package:",
	"wails-build-webview2:",
	"npm-build:",
	"Discovery",
	"id: discovery",
	"primary_stack_suggestion=wails2",
	"echo \"os=$runner_os\"",
	"echo \"arch=$runner_arch\"",
	"echo \"short_sha=$short_sha\"",
	"echo \"has_root_composer_json=$has_root_composer_json\"",
	"echo \"has_root_cargo_toml=$has_root_cargo_toml\"",
	"echo \"has_root_go_work=$has_root_go_work\"",
	"echo \"has_root_wails_json=$has_root_wails_json\"",
	"echo \"has_subtree_package_json=$has_subtree_package_json\"",
	"echo \"has_subtree_deno_manifest=$has_subtree_deno_manifest\"",
	"echo \"has_taskfile=$has_taskfile\"",
	"configured_build_type=\"\"",
	"echo \"configured_build_type=$configured_build_type\"",
	"match = re.match(r\"^\\s*type:\\s*(.+?)\\s*$\", line)",
	"Setup Go",
	"actions/setup-go@v5",
	"steps.discovery.outputs.has_go_toolchain == 'true'",
	"steps.discovery.outputs.has_taskfile == 'true'",
	"steps.discovery.outputs.configured_build_type == 'go'",
	"steps.discovery.outputs.configured_build_type == 'wails'",
	"steps.discovery.outputs.configured_build_type == 'taskfile'",
	"Install Garble",
	"mvdan.cc/garble@latest",
	"Install Task CLI",
	"github.com/go-task/task/v3/cmd/task@latest",
	"Setup Node",
	"actions/setup-node@v4",
	"steps.discovery.outputs.has_package_json == 'true'",
	"steps.discovery.outputs.configured_build_type == 'node'",
	"Enable Corepack",
	"corepack enable",
	"Install frontend dependencies",
	"package_manager_from_manifest()",
	"pkg.packageManager",
	"declared_manager=\"$(package_manager_from_manifest \"$dir\")\"",
	"pnpm install --frozen-lockfile",
	"(cd \"$dir\" && pnpm install)",
	"yarn install --immutable",
	"(cd \"$dir\" && yarn install)",
	"bun install --frozen-lockfile",
	"(cd \"$dir\" && bun install)",
	"curl -fsSL https://bun.sh/install | bash",
	"npm ci",
	"find_visible_files()",
	"-path './.*'",
	"find_visible_files 3 -name package.json",
	"Install Wails CLI",
	"github.com/wailsapp/wails/v3/cmd/wails3",
	"github.com/wailsapp/wails/v2/cmd/wails",
	"Setup Python for Conan and MkDocs",
	"steps.discovery.outputs.has_root_cmakelists == 'true' || steps.discovery.outputs.has_docs_config == 'true'",
	"Install Linux Wails dependencies",
	"steps.discovery.outputs.primary_stack_suggestion == 'wails2'",
	"runner.os == 'Linux' && (steps.discovery.outputs.primary_stack_suggestion == 'wails2' || steps.discovery.outputs.configured_build_type == 'wails')",
	"libwebkit2gtk-4.0-dev",
	"libwebkit2gtk-4.1-dev",
	"dpkg --compare-versions",
	"Setup PHP and Composer",
	"steps.discovery.outputs.has_root_composer_json == 'true'",
	"steps.discovery.outputs.configured_build_type == 'php'",
	"composer-setup.php",
	"Setup Rust",
	"steps.discovery.outputs.has_root_cargo_toml == 'true'",
	"steps.discovery.outputs.configured_build_type == 'rust'",
	"https://sh.rustup.rs",
	"choco install rustup.install -y",
	"Install Conan",
	"Install MkDocs",
	"actions/setup-python@v5",
	"python -m pip install conan",
	"python -m pip install mkdocs",
	"Setup Deno",
	"denoland/setup-deno@v2",
	"echo \"deno_requested=$deno_requested\"",
	"echo \"npm_requested=$npm_requested\"",
	"steps.discovery.outputs.deno_requested == 'true'",
	"steps.discovery.outputs.has_deno_manifest == 'true'",
	"steps.discovery.outputs.npm_requested == 'true'",
	"build-cache:",
	"Restore build cache",
	"actions/cache@v4",
	"${{ inputs.working-directory }}/.core/cache",
	"core-build-${{ runner.os }}-${{ matrix.target }}-",
	"inputs.build-platform == '' || inputs.build-platform == matrix.target",
	"core build --ci --targets",
	"--ci",
	"--build-name",
	"--build-tags",
	"--version",
	"--build-obfuscate",
	"--sign=true",
	"--sign=false",
	"--package=false",
	"--build-cache=false",
	"--npm-build",
	"--wails-build-webview2",
	"--archive-format",
	"Resolve build name",
	"Compute artifact upload name",
	"build_name=\"${{ inputs.build-name }}\"",
	"if [ -z \"$build_name\" ] && [ -f .core/build.yaml ]; then",
	"in_project = stripped == \"project:\"",
	"echo \"value=${build_name}\" >> \"${GITHUB_OUTPUT}\"",
	"build_name=\"${{ steps.build_name.outputs.value }}\"",
	"build_name=\"${GITHUB_REPOSITORY##*/}\"",
	"suffix=\"${{ steps.discovery.outputs.short_sha }}\"",
	"steps.discovery.outputs.is_tag",
	"steps.discovery.outputs.tag",
	"name: ${{ steps.artifact-name.outputs.value }}",
	"echo \"value=${artifact_name}\" >> \"${GITHUB_OUTPUT}\"",
	"if: ${{ inputs.package }}",
	"if: ${{ inputs.build && inputs.package && startsWith(github.ref, 'refs/tags/') }}",
	"actions/download-artifact@v4",
	"command: ci",
	"we-are-go-for-launch: true",
	"uses: dAppCore/build@v3",
}

var releaseWorkflowForbiddenMarkers = []string{
	"release-${artifact_name}",
	"pattern: release-*",
}

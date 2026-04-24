package build

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core/io"
)

func TestXcodeCloud_HasXcodeCloudConfig_Good(t *testing.T) {
	if HasXcodeCloudConfig(nil) {
		t.Fatal("expected false")
	}
	if (HasXcodeCloudConfig(&BuildConfig{})) {
		t.Fatal("expected false")
	}
	if !(HasXcodeCloudConfig(&BuildConfig{Apple: AppleConfig{XcodeCloud: XcodeCloudConfig{Workflow: "CoreGUI Release"}}})) {
		t.Fatal("expected true")
	}
	if !(HasXcodeCloudConfig(&BuildConfig{Apple: AppleConfig{XcodeCloud: XcodeCloudConfig{Triggers: []XcodeCloudTrigger{{Branch: "main", Action: "testflight"}}}}})) {
		t.Fatal("expected true")
	}

}

func TestXcodeCloud_GenerateXcodeCloudScripts_Good(t *testing.T) {
	scripts := GenerateXcodeCloudScripts("/tmp/project", &BuildConfig{
		Project: Project{
			Name:   "Core",
			Binary: "Core",
		},
		Apple: AppleConfig{
			BundleID: "ai.lthn.core",
			TeamID:   "ABC123DEF4",
			Arch:     "universal",
			Notarise: boolPtr(false),
			DMG:      boolPtr(true),
			AppStore: boolPtr(true),
		},
	})
	if len(scripts) != 3 {
		t.Fatalf("want len %v, got %v", 3, len(scripts))
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], "go install github.com/wailsapp/wails/v3/cmd/wails3@latest") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], "go install github.com/wailsapp/wails/v3/cmd/wails3@latest")
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], "find_visible_files()") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], "find_visible_files()")
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], "-path './.*'") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], "-path './.*'")
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], "find_visible_files 3 -name package.json") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], "find_visible_files 3 -name package.json")
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], "package_manager_from_manifest()") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], "package_manager_from_manifest()")
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], "pkg.packageManager") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], "pkg.packageManager")
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], `declared_manager="$(package_manager_from_manifest "$dir")"`) {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], `declared_manager="$(package_manager_from_manifest "$dir")"`)
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], `(cd "$dir" && pnpm install)`) {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], `(cd "$dir" && pnpm install)`)
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], `(cd "$dir" && yarn install)`) {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], `(cd "$dir" && yarn install)`)
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], `(cd "$dir" && bun install)`) {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], `(cd "$dir" && bun install)`)
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], "deno_requested()") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], "deno_requested()")
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], "DENO_ENABLE") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], "DENO_ENABLE")
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostCloneScriptName], "DENO_BUILD") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostCloneScriptName], "DENO_BUILD")
	}
	if !stdlibAssertContains(scripts[XcodeCloudPreXcodebuildScriptName], `core build apple --arch 'universal' --config '.core/build.yaml' --notarise=false --dmg --appstore --bundle-id 'ai.lthn.core' --team-id 'ABC123DEF4'`) {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPreXcodebuildScriptName], `core build apple --arch 'universal' --config '.core/build.yaml' --notarise=false --dmg --appstore --bundle-id 'ai.lthn.core' --team-id 'ABC123DEF4'`)
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostXcodebuildScriptName], `BUNDLE_PATH='dist/apple/Core.app'`) {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostXcodebuildScriptName], `BUNDLE_PATH='dist/apple/Core.app'`)
	}
	if !stdlibAssertContains(scripts[XcodeCloudPostXcodebuildScriptName], "codesign --verify --deep --strict") {
		t.Fatalf("expected %v to contain %v", scripts[XcodeCloudPostXcodebuildScriptName], "codesign --verify --deep --strict")
	}

}

func TestXcodeCloud_GenerateXcodeCloudScripts_QuotesShellValues(t *testing.T) {
	scripts := GenerateXcodeCloudScripts("/tmp/project", &BuildConfig{
		Project: Project{
			Name:   "Core",
			Binary: "Core$(touch /tmp/pwned)",
		},
		Apple: AppleConfig{
			BundleID: "ai.lthn.core$(touch /tmp/pwned)",
			TeamID:   "ABC123DEF4$(touch /tmp/pwned)",
			Arch:     "arm64$(touch /tmp/pwned)",
		},
	})

	pre := scripts[XcodeCloudPreXcodebuildScriptName]
	if !stdlibAssertContains(pre, `--arch 'arm64$(touch /tmp/pwned)'`) {
		t.Fatalf("expected %v to contain %v", pre, `--arch 'arm64$(touch /tmp/pwned)'`)
	}
	if !stdlibAssertContains(pre, `--bundle-id 'ai.lthn.core$(touch /tmp/pwned)'`) {
		t.Fatalf("expected %v to contain %v", pre, `--bundle-id 'ai.lthn.core$(touch /tmp/pwned)'`)
	}
	if !stdlibAssertContains(pre, `--team-id 'ABC123DEF4$(touch /tmp/pwned)'`) {
		t.Fatalf("expected %v to contain %v", pre, `--team-id 'ABC123DEF4$(touch /tmp/pwned)'`)
	}
	if stdlibAssertContains(pre, `--arch "arm64$(touch /tmp/pwned)"`) {
		t.Fatalf("expected %v not to contain %v", pre, `--arch "arm64$(touch /tmp/pwned)"`)
	}
	if stdlibAssertContains(pre, `--bundle-id "ai.lthn.core$(touch /tmp/pwned)"`) {
		t.Fatalf("expected %v not to contain %v", pre, `--bundle-id "ai.lthn.core$(touch /tmp/pwned)"`)
	}
	if stdlibAssertContains(pre, `--team-id "ABC123DEF4$(touch /tmp/pwned)"`) {
		t.Fatalf("expected %v not to contain %v", pre, `--team-id "ABC123DEF4$(touch /tmp/pwned)"`)
	}

	post := scripts[XcodeCloudPostXcodebuildScriptName]
	if !stdlibAssertContains(post, `BUNDLE_PATH='dist/apple/Core$(touch /tmp/pwned).app'`) {
		t.Fatalf("expected %v to contain %v", post, `BUNDLE_PATH='dist/apple/Core$(touch /tmp/pwned).app'`)
	}
	if !stdlibAssertContains(post, `EXECUTABLE_PATH='dist/apple/Core$(touch /tmp/pwned).app/Contents/MacOS/Core$(touch /tmp/pwned)'`) {
		t.Fatalf("expected %v to contain %v", post, `EXECUTABLE_PATH='dist/apple/Core$(touch /tmp/pwned).app/Contents/MacOS/Core$(touch /tmp/pwned)'`)
	}
	if stdlibAssertContains(post, `BUNDLE_PATH="dist/apple/Core$(touch /tmp/pwned).app"`) {
		t.Fatalf("expected %v not to contain %v", post, `BUNDLE_PATH="dist/apple/Core$(touch /tmp/pwned).app"`)
	}
	if stdlibAssertContains(post, `EXECUTABLE_PATH="dist/apple/Core$(touch /tmp/pwned).app/Contents/MacOS/Core$(touch /tmp/pwned)"`) {
		t.Fatalf("expected %v not to contain %v", post, `EXECUTABLE_PATH="dist/apple/Core$(touch /tmp/pwned).app/Contents/MacOS/Core$(touch /tmp/pwned)"`)
	}

}

func TestXcodeCloud_WriteXcodeCloudScripts_Good(t *testing.T) {
	projectDir := t.TempDir()

	paths, err := WriteXcodeCloudScripts(io.Local, projectDir, &BuildConfig{
		Project: Project{
			Name:   "Core",
			Binary: "Core",
		},
		Apple: AppleConfig{
			XcodeCloud: XcodeCloudConfig{
				Workflow: "CoreGUI Release",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]string{ax.Join(projectDir, XcodeCloudScriptsDir, XcodeCloudPostCloneScriptName), ax.Join(projectDir, XcodeCloudScriptsDir, XcodeCloudPreXcodebuildScriptName), ax.Join(projectDir, XcodeCloudScriptsDir, XcodeCloudPostXcodebuildScriptName)}, paths) {
		t.Fatalf("want %v, got %v", []string{ax.Join(projectDir, XcodeCloudScriptsDir, XcodeCloudPostCloneScriptName), ax.Join(projectDir, XcodeCloudScriptsDir, XcodeCloudPreXcodebuildScriptName), ax.Join(projectDir, XcodeCloudScriptsDir, XcodeCloudPostXcodebuildScriptName)}, paths)
	}

	for _, path := range paths {
		content, err := io.Local.Read(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(content) {
			t.Fatal("expected non-empty")
		}

		info, err := io.Local.Stat(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(0o755, int(info.Mode().Perm())) {
			t.Fatalf("want %v, got %v", 0o755, int(info.Mode().Perm()))
		}

	}
}

func TestXcodeCloud_WriteXcodeCloudScripts_Bad(t *testing.T) {
	_, err := WriteXcodeCloudScripts(nil, t.TempDir(), DefaultConfig())
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "filesystem medium is required") {
		t.Fatalf("expected %v to contain %v", err.Error(), "filesystem medium is required")
	}

}

func boolPtr(value bool) *bool {
	return &value
}

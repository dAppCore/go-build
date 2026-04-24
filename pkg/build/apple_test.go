package build

import (
	"context"
	"os"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

func TestApple_WriteInfoPlist_Good(t *testing.T) {
	appPath := ax.Join(t.TempDir(), "Core.app")

	path, err := WriteInfoPlist(io.Local, appPath, InfoPlist{
		BundleID:                      "ai.lthn.core",
		BundleName:                    "Core",
		BundleDisplayName:             "Core by Lethean",
		BundleVersion:                 "1.2.3",
		BuildNumber:                   "42",
		MinSystemVersion:              "13.0",
		Category:                      "public.app-category.developer-tools",
		Copyright:                     "Copyright 2026 Lethean CIC. EUPL-1.2.",
		Executable:                    "Core",
		HighResCapable:                true,
		SupportsSecureRestorableState: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := io.Local.Read(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(content, "<key>CFBundleIdentifier</key>") {
		t.Fatalf("expected %v to contain %v", content, "<key>CFBundleIdentifier</key>")
	}
	if !stdlibAssertContains(content, "<string>ai.lthn.core</string>") {
		t.Fatalf("expected %v to contain %v", content, "<string>ai.lthn.core</string>")
	}
	if !stdlibAssertContains(content, "<key>CFBundleExecutable</key>") {
		t.Fatalf("expected %v to contain %v", content, "<key>CFBundleExecutable</key>")
	}
	if !stdlibAssertContains(content, "<string>Core</string>") {
		t.Fatalf("expected %v to contain %v", content, "<string>Core</string>")
	}

}

func TestApple_CreateUniversal_Good(t *testing.T) {
	dir := t.TempDir()
	arm64Path := ax.Join(dir, "arm64", "Core.app")
	amd64Path := ax.Join(dir, "amd64", "Core.app")
	outputPath := ax.Join(dir, "universal", "Core.app")

	writeDummyAppBundle(t, arm64Path, "Core", "arm64")
	writeDummyAppBundle(t, amd64Path, "Core", "amd64")

	oldResolve := appleResolveCommand
	oldCombined := appleCombinedOutput
	t.Cleanup(func() {
		appleResolveCommand = oldResolve
		appleCombinedOutput = oldCombined
	})

	appleResolveCommand = func(name string, fallbackPaths ...string) (string, error) {
		return name, nil
	}
	appleCombinedOutput = func(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
		if !stdlibAssertEqual("lipo", command) {
			t.Fatalf("want %v, got %v", "lipo", command)
		}
		if !stdlibAssertEqual([]string{"-create", "-output", ax.Join(outputPath, "Contents", "MacOS", "Core"), ax.Join(arm64Path, "Contents", "MacOS", "Core"), ax.Join(amd64Path, "Contents", "MacOS", "Core")}, args) {
			t.Fatalf("want %v, got %v", []string{"-create", "-output", ax.Join(outputPath, "Contents", "MacOS", "Core"), ax.Join(arm64Path, "Contents", "MacOS", "Core"), ax.Join(amd64Path, "Contents", "MacOS", "Core")}, args)
		}
		if err := ax.WriteFile(ax.Join(outputPath, "Contents", "MacOS", "Core"), []byte("universal"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		return "", nil
	}

	err := CreateUniversal(arm64Path, amd64Path, outputPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := ax.ReadFile(ax.Join(outputPath, "Contents", "MacOS", "Core"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("universal", string(content)) {
		t.Fatalf("want %v, got %v", "universal", string(content))
	}

}

func TestApple_CreateUniversal_MergesHelpersAndFrameworks_Good(t *testing.T) {
	dir := t.TempDir()
	arm64Path := ax.Join(dir, "arm64", "Core.app")
	amd64Path := ax.Join(dir, "amd64", "Core.app")
	outputPath := ax.Join(dir, "universal", "Core.app")

	writeDummyAppBundle(t, arm64Path, "Core", "arm64-main")
	writeDummyAppBundle(t, amd64Path, "Core", "amd64-main")
	writeDummyExecutable(t, ax.Join(arm64Path, "Contents", "MacOS", "Core Helper"), "arm64-helper")
	writeDummyExecutable(t, ax.Join(amd64Path, "Contents", "MacOS", "Core Helper"), "amd64-helper")
	writeDummyExecutable(t, ax.Join(arm64Path, "Contents", "Frameworks", "Example.framework", "Example"), "arm64-framework")
	writeDummyExecutable(t, ax.Join(amd64Path, "Contents", "Frameworks", "Example.framework", "Example"), "amd64-framework")
	writeDummyExecutable(t, ax.Join(arm64Path, "Contents", "Frameworks", "libSupport.dylib"), "arm64-dylib")
	writeDummyExecutable(t, ax.Join(amd64Path, "Contents", "Frameworks", "libSupport.dylib"), "amd64-dylib")

	oldResolve := appleResolveCommand
	oldCombined := appleCombinedOutput
	t.Cleanup(func() {
		appleResolveCommand = oldResolve
		appleCombinedOutput = oldCombined
	})

	var mergedOutputs []string
	appleResolveCommand = func(name string, fallbackPaths ...string) (string, error) {
		return name, nil
	}
	appleCombinedOutput = func(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
		if !stdlibAssertEqual("lipo", command) {
			t.Fatalf("want %v, got %v", "lipo", command)
		}
		if len(args) != 5 {
			t.Fatalf("want len %v, got %v", 5, len(args))
		}
		if !stdlibAssertEqual("-create", args[0]) {
			t.Fatalf("want %v, got %v", "-create", args[0])
		}
		if !stdlibAssertEqual("-output", args[1]) {
			t.Fatalf("want %v, got %v", "-output", args[1])
		}

		mergedOutputs = append(mergedOutputs, args[2])
		if err := ax.WriteFile(args[2], []byte("universal"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		return "", nil
	}

	err := CreateUniversal(arm64Path, amd64Path, outputPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]string{ax.Join(outputPath, "Contents", "Frameworks", "Example.framework", "Example"), ax.Join(outputPath, "Contents", "Frameworks", "libSupport.dylib"), ax.Join(outputPath, "Contents", "MacOS", "Core"), ax.Join(outputPath, "Contents", "MacOS", "Core Helper")}, mergedOutputs) {
		t.Fatalf("want %v, got %v", []string{ax.Join(outputPath, "Contents", "Frameworks", "Example.framework", "Example"), ax.Join(outputPath, "Contents", "Frameworks", "libSupport.dylib"), ax.Join(outputPath, "Contents", "MacOS", "Core"), ax.Join(outputPath, "Contents", "MacOS", "Core Helper")}, mergedOutputs)
	}

	for _, path := range mergedOutputs {
		content, readErr := ax.ReadFile(path)
		if readErr != nil {
			t.Fatalf("unexpected error: %v", readErr)
		}
		if !stdlibAssertEqual("universal", string(content)) {
			t.Fatalf("want %v, got %v", "universal", string(content))
		}

	}
}

func TestApple_NormaliseDMGConfig_Defaults_Good(t *testing.T) {
	cfg := normaliseDMGConfig(DMGConfig{
		AppPath: ax.Join("/tmp", "Core.app"),
	})
	if !stdlibAssertEqual("Core", cfg.VolumeName) {
		t.Fatalf("want %v, got %v", "Core", cfg.VolumeName)
	}
	if !stdlibAssertEqual(defaultDMGIconSize, cfg.IconSize) {
		t.Fatalf("want %v, got %v", defaultDMGIconSize, cfg.IconSize)
	}
	if !stdlibAssertEqual([2]int{defaultDMGWindowWidth, defaultDMGWindowHeight}, cfg.WindowSize) {
		t.Fatalf("want %v, got %v", [2]int{defaultDMGWindowWidth, defaultDMGWindowHeight}, cfg.WindowSize)
	}

}

func TestApple_BuildDMGAppleScript_UsesConfiguredLayout_Good(t *testing.T) {
	script := buildDMGAppleScript("Core", "Core.app", DMGConfig{
		AppPath:    ax.Join("/tmp", "Core.app"),
		Background: "assets/dmg-background.png",
		IconSize:   144,
		WindowSize: [2]int{800, 520},
	})
	if !stdlibAssertContains(script, `tell disk "Core"`) {
		t.Fatalf("expected %v to contain %v", script, `tell disk "Core"`)
	}
	if !stdlibAssertContains(script, "set bounds of container window to {100, 100, 900, 620}") {
		t.Fatalf("expected %v to contain %v", script, "set bounds of container window to {100, 100, 900, 620}")
	}
	if !stdlibAssertContains(script, "set icon size of opts to 144") {
		t.Fatalf("expected %v to contain %v", script, "set icon size of opts to 144")
	}
	if !stdlibAssertContains(script, `set background picture of opts to file ".background:dmg-background.png"`) {
		t.Fatalf("expected %v to contain %v", script, `set background picture of opts to file ".background:dmg-background.png"`)
	}
	if !stdlibAssertContains(script, `set position of item "Core.app" of container window to {200, 260}`) {
		t.Fatalf("expected %v to contain %v", script, `set position of item "Core.app" of container window to {200, 260}`)
	}
	if !stdlibAssertContains(script, `set position of item "Applications" of container window to {600, 260}`) {
		t.Fatalf("expected %v to contain %v", script, `set position of item "Applications" of container window to {600, 260}`)
	}

}

func TestApple_CreateDMG_ConfiguresFinderLayout_Good(t *testing.T) {
	projectDir := t.TempDir()
	appPath := ax.Join(projectDir, "Core.app")
	backgroundPath := ax.Join(projectDir, "assets", "dmg-background.png")
	outputPath := ax.Join(projectDir, "dist", "Core.dmg")

	writeDummyAppBundle(t, appPath, "Core", "built")
	if err := io.Local.EnsureDir(ax.Dir(backgroundPath)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(backgroundPath, []byte("background"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldResolve := appleResolveCommand
	oldCombined := appleCombinedOutput
	t.Cleanup(func() {
		appleResolveCommand = oldResolve
		appleCombinedOutput = oldCombined
	})

	var commands []struct {
		command string
		args    []string
	}

	appleResolveCommand = func(name string, fallbackPaths ...string) (string, error) {
		return name, nil
	}
	appleCombinedOutput = func(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
		commands = append(commands, struct {
			command string
			args    []string
		}{
			command: command,
			args:    append([]string{}, args...),
		})

		switch command {
		case "hdiutil":
			if stdlibAssertEmpty(args) {
				t.Fatal("expected non-empty")
			}

			switch args[0] {
			case "create":
				srcIndex := indexOf(args, "-srcfolder")
				if srcIndex < 0 {
					t.Fatalf("expected %v to be greater than or equal to %v", srcIndex, 0)
				}

				stageDir := args[srcIndex+1]
				if !(io.Local.Exists(ax.Join(stageDir, "Core.app"))) {
					t.Fatal("expected true")
				}

				linkTarget, err := os.Readlink(ax.Join(stageDir, "Applications"))
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !stdlibAssertEqual("/Applications", linkTarget) {
					t.Fatalf("want %v, got %v", "/Applications", linkTarget)
				}

				backgroundContent, err := io.Local.Read(ax.Join(stageDir, ".background", "dmg-background.png"))
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !stdlibAssertEqual("background", backgroundContent) {
					t.Fatalf("want %v, got %v", "background", backgroundContent)
				}

			case "attach":
				if !stdlibAssertContains(args, "-mountpoint") {
					t.Fatalf("expected %v to contain %v", args, "-mountpoint")
				}

			case "detach":
				if !stdlibAssertEqual("detach", args[0]) {
					t.Fatalf("want %v, got %v", "detach", args[0])
				}

			case "convert":
				if !stdlibAssertEqual(outputPath, args[len(args)-1]) {
					t.Fatalf("want %v, got %v", outputPath, args[len(args)-1])
				}

			default:
				t.Fatalf("unexpected hdiutil command: %v", args)
			}
		case "osascript":
			if len(args) != 1 {
				t.Fatalf("want len %v, got %v", 1, len(args))
			}

			script, err := io.Local.Read(args[0])
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !stdlibAssertContains(script, "set bounds of container window to {100, 100, 740, 580}") {
				t.Fatalf("expected %v to contain %v", script, "set bounds of container window to {100, 100, 740, 580}")
			}
			if !stdlibAssertContains(script, "set icon size of opts to 144") {
				t.Fatalf("expected %v to contain %v", script, "set icon size of opts to 144")
			}
			if !stdlibAssertContains(script, `set background picture of opts to file ".background:dmg-background.png"`) {
				t.Fatalf("expected %v to contain %v", script, `set background picture of opts to file ".background:dmg-background.png"`)
			}
			if !stdlibAssertContains(script, `set position of item "Core.app" of container window to {176, 240}`) {
				t.Fatalf("expected %v to contain %v", script, `set position of item "Core.app" of container window to {176, 240}`)
			}
			if !stdlibAssertContains(script, `set position of item "Applications" of container window to {480, 240}`) {
				t.Fatalf("expected %v to contain %v", script, `set position of item "Applications" of container window to {480, 240}`)
			}

		default:
			t.Fatalf("unexpected command: %s", command)
		}

		return "", nil
	}

	err := CreateDMG(context.Background(), DMGConfig{
		AppPath:    appPath,
		OutputPath: outputPath,
		VolumeName: "Core",
		Background: backgroundPath,
		IconSize:   144,
		WindowSize: [2]int{640, 480},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commands) != 5 {
		t.Fatalf("want len %v, got %v", 5, len(commands))
	}
	if !stdlibAssertEqual("hdiutil", commands[0].command) {
		t.Fatalf("want %v, got %v", "hdiutil", commands[0].command)
	}
	if !stdlibAssertEqual("create", commands[0].args[0]) {
		t.Fatalf("want %v, got %v", "create", commands[0].args[0])
	}
	if !stdlibAssertEqual("hdiutil", commands[1].command) {
		t.Fatalf("want %v, got %v", "hdiutil", commands[1].command)
	}
	if !stdlibAssertEqual("attach", commands[1].args[0]) {
		t.Fatalf("want %v, got %v", "attach", commands[1].args[0])
	}
	if !stdlibAssertEqual("osascript", commands[2].command) {
		t.Fatalf("want %v, got %v", "osascript", commands[2].command)
	}
	if !stdlibAssertEqual("hdiutil", commands[3].command) {
		t.Fatalf("want %v, got %v", "hdiutil", commands[3].command)
	}
	if !stdlibAssertEqual("detach", commands[3].args[0]) {
		t.Fatalf("want %v, got %v", "detach", commands[3].args[0])
	}
	if !stdlibAssertEqual("hdiutil", commands[4].command) {
		t.Fatalf("want %v, got %v", "hdiutil", commands[4].command)
	}
	if !stdlibAssertEqual("convert", commands[4].args[0]) {
		t.Fatalf("want %v, got %v", "convert", commands[4].args[0])
	}

}

func TestApple_BuildWailsApp_AddsMLXBuildTag_Good(t *testing.T) {
	projectDir := t.TempDir()
	bundlePath := ax.Join(projectDir, "build", "bin", "Core.app")
	writeDummyAppBundle(t, bundlePath, "Core", "built")

	oldResolve := appleResolveCommand
	oldCombined := appleCombinedOutput
	t.Cleanup(func() {
		appleResolveCommand = oldResolve
		appleCombinedOutput = oldCombined
	})

	appleResolveCommand = func(name string, fallbackPaths ...string) (string, error) {
		return name, nil
	}
	appleCombinedOutput = func(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
		if !stdlibAssertEqual("wails3", command) {
			t.Fatalf("want %v, got %v", "wails3", command)
		}
		if !stdlibAssertContains(args, "-tags") {
			t.Fatalf("expected %v to contain %v", args, "-tags")
		}

		tagIndex := -1
		for i, arg := range args {
			if arg == "-tags" {
				tagIndex = i + 1
				break
			}
		}
		if tagIndex < 1 {
			t.Fatalf("expected %v to be greater than or equal to %v", tagIndex, 1)
		}
		if !stdlibAssertEqual("integration,mlx", args[tagIndex]) {
			t.Fatalf("want %v, got %v", "integration,mlx", args[tagIndex])
		}

		return "", nil
	}

	result, err := BuildWailsApp(context.Background(), WailsBuildConfig{
		ProjectDir: projectDir,
		Name:       "Core",
		Arch:       "arm64",
		BuildTags:  []string{"integration"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(bundlePath, result) {
		t.Fatalf("want %v, got %v", bundlePath, result)
	}

}

func TestApple_BuildWailsApp_PreBuildsFrontendAndForcesCGO_Good(t *testing.T) {
	projectDir := t.TempDir()
	frontendDir := ax.Join(projectDir, "frontend")
	bundlePath := ax.Join(projectDir, "build", "bin", "Core.app")
	if err := io.Local.EnsureDir(frontendDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldResolve := appleResolveCommand
	oldCombined := appleCombinedOutput
	t.Cleanup(func() {
		appleResolveCommand = oldResolve
		appleCombinedOutput = oldCombined
	})

	var calls []struct {
		dir     string
		command string
		args    []string
		env     []string
	}

	appleResolveCommand = func(name string, fallbackPaths ...string) (string, error) {
		return name, nil
	}
	appleCombinedOutput = func(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
		calls = append(calls, struct {
			dir     string
			command string
			args    []string
			env     []string
		}{
			dir:     dir,
			command: command,
			args:    append([]string{}, args...),
			env:     append([]string{}, env...),
		})

		switch command {
		case "deno-build":
			if !stdlibAssertEqual(frontendDir, dir) {
				t.Fatalf("want %v, got %v", frontendDir, dir)
			}
			if !stdlibAssertEqual([]string{"--target", "release"}, args) {
				t.Fatalf("want %v, got %v", []string{"--target", "release"}, args)
			}

		case "wails3":
			if !stdlibAssertEqual(projectDir, dir) {
				t.Fatalf("want %v, got %v", projectDir, dir)
			}
			if !stdlibAssertContains(env, "CGO_ENABLED=1") {
				t.Fatalf("expected %v to contain %v", env, "CGO_ENABLED=1")
			}

			writeDummyAppBundle(t, bundlePath, "Core", "built")
		default:
			t.Fatalf("unexpected command: %s", command)
		}

		return "", nil
	}

	result, err := BuildWailsApp(context.Background(), WailsBuildConfig{
		ProjectDir: projectDir,
		Name:       "Core",
		Arch:       "arm64",
		OutputDir:  ax.Join(projectDir, "dist"),
		DenoBuild:  "deno-build --target release",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "dist", "Core.app"), result) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", "Core.app"), result)
	}
	if len(calls) != 2 {
		t.Fatalf("want len %v, got %v", 2, len(calls))
	}
	if !stdlibAssertEqual("deno-build", calls[0].command) {
		t.Fatalf("want %v, got %v", "deno-build", calls[0].command)
	}
	if !stdlibAssertEqual("wails3", calls[1].command) {
		t.Fatalf("want %v, got %v", "wails3", calls[1].command)
	}

}

func TestApple_BuildWailsApp_UsesDenoWhenEnabledWithoutManifest_Good(t *testing.T) {
	projectDir := t.TempDir()
	bundlePath := ax.Join(projectDir, "build", "bin", "Core.app")
	if err := ax.WriteFile(ax.Join(projectDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("DENO_ENABLE", "true")

	oldResolve := appleResolveCommand
	oldCombined := appleCombinedOutput
	t.Cleanup(func() {
		appleResolveCommand = oldResolve
		appleCombinedOutput = oldCombined
	})

	var calls []struct {
		dir     string
		command string
		args    []string
	}

	appleResolveCommand = func(name string, fallbackPaths ...string) (string, error) {
		return name, nil
	}
	appleCombinedOutput = func(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
		calls = append(calls, struct {
			dir     string
			command string
			args    []string
		}{
			dir:     dir,
			command: command,
			args:    append([]string{}, args...),
		})

		switch command {
		case "deno":
			if !stdlibAssertEqual(projectDir, dir) {
				t.Fatalf("want %v, got %v", projectDir, dir)
			}
			if !stdlibAssertEqual([]string{"task", "build"}, args) {
				t.Fatalf("want %v, got %v", []string{"task", "build"}, args)
			}

		case "wails3":
			writeDummyAppBundle(t, bundlePath, "Core", "built")
		default:
			t.Fatalf("unexpected command: %s", command)
		}

		return "", nil
	}

	result, err := BuildWailsApp(context.Background(), WailsBuildConfig{
		ProjectDir: projectDir,
		Name:       "Core",
		Arch:       "arm64",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(bundlePath, result) {
		t.Fatalf("want %v, got %v", bundlePath, result)
	}
	if len(calls) != 2 {
		t.Fatalf("want len %v, got %v", 2, len(calls))
	}
	if !stdlibAssertEqual("deno", calls[0].command) {
		t.Fatalf("want %v, got %v", "deno", calls[0].command)
	}
	if !stdlibAssertEqual("wails3", calls[1].command) {
		t.Fatalf("want %v, got %v", "wails3", calls[1].command)
	}

}

func TestApple_BuildApple_Good(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist", "apple")

	oldBuildWails := appleBuildWailsAppFn
	oldUniversal := appleCreateUniversalFn
	oldSign := appleSignFn
	oldNotarise := appleNotariseFn
	oldDMG := appleCreateDMGFn
	t.Cleanup(func() {
		appleBuildWailsAppFn = oldBuildWails
		appleCreateUniversalFn = oldUniversal
		appleSignFn = oldSign
		appleNotariseFn = oldNotarise
		appleCreateDMGFn = oldDMG
	})

	var builtArches []string
	var buildEnvs [][]string
	appleBuildWailsAppFn = func(ctx context.Context, cfg WailsBuildConfig) (string, error) {
		builtArches = append(builtArches, cfg.Arch)
		buildEnvs = append(buildEnvs, append([]string{}, cfg.Env...))
		appPath := ax.Join(cfg.OutputDir, cfg.Name+".app")
		writeDummyAppBundle(t, appPath, cfg.Name, cfg.Arch)
		return appPath, nil
	}
	appleCreateUniversalFn = func(arm64Path, amd64Path, outputPath string) error {
		if err := copyPath(io.Local, arm64Path, outputPath); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		return ax.WriteFile(ax.Join(outputPath, "Contents", "MacOS", "Core"), []byte("universal"), 0o755)
	}

	var signCalls []SignConfig
	appleSignFn = func(ctx context.Context, cfg SignConfig) error {
		signCalls = append(signCalls, cfg)
		return nil
	}

	var notarisedPath string
	appleNotariseFn = func(ctx context.Context, cfg NotariseConfig) error {
		notarisedPath = cfg.AppPath
		return nil
	}

	var dmgCall DMGConfig
	appleCreateDMGFn = func(ctx context.Context, cfg DMGConfig) error {
		dmgCall = cfg
		return ax.WriteFile(cfg.OutputPath, []byte("dmg"), 0o644)
	}

	result, err := BuildApple(context.Background(), &Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "Core",
		Version:    "v1.2.3",
		BuildTags:  []string{"integration"},
		LDFlags:    []string{"-s", "-w"},
		Cache: CacheConfig{
			Enabled: true,
			Paths: []string{
				ax.Join(outputDir, "cache", "go-build"),
				ax.Join(outputDir, "cache", "go-mod"),
			},
		},
	}, AppleOptions{
		BundleID:     "ai.lthn.core",
		Arch:         "universal",
		Sign:         true,
		Notarise:     true,
		DMG:          true,
		CertIdentity: "Developer ID Application: Lethean CIC (ABC123DEF4)",
		TeamID:       "ABC123DEF4",
		AppleID:      "dev@example.com",
		Password:     "app-password",
	}, "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]string{"arm64", "amd64"}, builtArches) {
		t.Fatalf("want %v, got %v", []string{"arm64", "amd64"}, builtArches)
	}
	if len(buildEnvs) != 2 {
		t.Fatalf("want len %v, got %v", 2, len(buildEnvs))
	}
	if !stdlibAssertContains(buildEnvs[0], "GOCACHE="+ax.Join(outputDir, "cache", "go-build")) {
		t.Fatalf("expected %v to contain %v", buildEnvs[0], "GOCACHE="+ax.Join(outputDir, "cache", "go-build"))
	}
	if !stdlibAssertContains(buildEnvs[0], "GOMODCACHE="+ax.Join(outputDir, "cache", "go-mod")) {
		t.Fatalf("expected %v to contain %v", buildEnvs[0], "GOMODCACHE="+ax.Join(outputDir, "cache", "go-mod"))
	}
	if !stdlibAssertEqual(ax.Join(outputDir, "Core.app"), result.BundlePath) {
		t.Fatalf("want %v, got %v", ax.Join(outputDir, "Core.app"), result.BundlePath)
	}
	if !stdlibAssertEqual(ax.Join(outputDir, "Core-1.2.3.dmg"), result.DMGPath) {
		t.Fatalf("want %v, got %v", ax.Join(outputDir, "Core-1.2.3.dmg"), result.DMGPath)
	}
	if !stdlibAssertEqual(result.DMGPath, notarisedPath) {
		t.Fatalf("want %v, got %v", result.DMGPath, notarisedPath)
	}
	if len(signCalls) != 2 {
		t.Fatalf("want len %v, got %v", 2, len(signCalls))
	}
	if !stdlibAssertEqual(result.BundlePath, signCalls[0].AppPath) {
		t.Fatalf("want %v, got %v", result.BundlePath, signCalls[0].AppPath)
	}
	if !stdlibAssertEqual(result.EntitlementsPath, signCalls[0].Entitlements) {
		t.Fatalf("want %v, got %v", result.EntitlementsPath, signCalls[0].Entitlements)
	}
	if !stdlibAssertEqual(result.DMGPath, signCalls[1].AppPath) {
		t.Fatalf("want %v, got %v", result.DMGPath, signCalls[1].AppPath)
	}
	if !stdlibAssertEmpty(signCalls[1].Entitlements) {
		t.Fatalf("expected empty, got %v", signCalls[1].Entitlements)
	}
	if signCalls[1].Hardened {
		t.Fatal("expected false")
	}
	if !stdlibAssertEqual(result.DMGPath, dmgCall.OutputPath) {
		t.Fatalf("want %v, got %v", result.DMGPath, dmgCall.OutputPath)
	}

	plistContent, err := io.Local.Read(result.InfoPlistPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(plistContent, "<string>ai.lthn.core</string>") {
		t.Fatalf("expected %v to contain %v", plistContent, "<string>ai.lthn.core</string>")
	}
	if !stdlibAssertContains(plistContent, "<string>42</string>") {
		t.Fatalf("expected %v to contain %v", plistContent, "<string>42</string>")
	}

	entitlementsContent, err := io.Local.Read(result.EntitlementsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(entitlementsContent, "<key>com.apple.security.app-sandbox</key>") {
		t.Fatalf("expected %v to contain %v", entitlementsContent, "<key>com.apple.security.app-sandbox</key>")
	}
	if !stdlibAssertContains(entitlementsContent, "<false/>") {
		t.Fatalf("expected %v to contain %v", entitlementsContent, "<false/>")
	}

}

func TestApple_NotariseAuthArgs_Good(t *testing.T) {
	args, err := notariseAuthArgs(NotariseConfig{
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
		APIKeyPath:     "/tmp/AuthKey_KEY123.p8",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]string{"--key", "/tmp/AuthKey_KEY123.p8", "--key-id", "KEY123", "--issuer", "ISSUER456"}, args) {
		t.Fatalf("want %v, got %v", []string{"--key", "/tmp/AuthKey_KEY123.p8", "--key-id", "KEY123", "--issuer", "ISSUER456"}, args)
	}

	args, err = notariseAuthArgs(NotariseConfig{
		TeamID:   "ABC123DEF4",
		AppleID:  "dev@example.com",
		Password: "app-password",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]string{"--apple-id", "dev@example.com", "--password", "app-password", "--team-id", "ABC123DEF4"}, args) {
		t.Fatalf("want %v, got %v", []string{"--apple-id", "dev@example.com", "--password", "app-password", "--team-id", "ABC123DEF4"}, args)
	}

}

func TestApple_Notarise_AppendsNotaryLogOnRejectedStatus_Bad(t *testing.T) {
	oldResolve := appleResolveCommand
	oldCombined := appleCombinedOutput
	t.Cleanup(func() {
		appleResolveCommand = oldResolve
		appleCombinedOutput = oldCombined
	})

	appleResolveCommand = func(name string, fallbackPaths ...string) (string, error) {
		return name, nil
	}
	appleCombinedOutput = func(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
		switch command {
		case "ditto":
			return "", nil
		case "xcrun":
			if len(args) < 2 {
				t.Fatalf("expected %v to be greater than or equal to %v", len(args), 2)
			}
			if !stdlibAssertEqual("notarytool", args[0]) {
				t.Fatalf("want %v, got %v", "notarytool", args[0])
			}

			switch args[1] {
			case "submit":
				return `{"id":"request-123","status":"Invalid"}`, nil
			case "log":
				return "notary log details", nil
			default:
				t.Fatalf("unexpected xcrun invocation: %v", args)
			}
		default:
			t.Fatalf("unexpected command: %s", command)
		}

		return "", nil
	}

	err := Notarise(context.Background(), NotariseConfig{
		AppPath:        ax.Join(t.TempDir(), "Core.app"),
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
		APIKeyPath:     "/tmp/AuthKey_KEY123.p8",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "status Invalid") {
		t.Fatalf("expected error %v to contain %v", err, "status Invalid")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "notary log details") {
		t.Fatalf("expected error %v to contain %v", err, "notary log details")
	}

}

func TestApple_BuildApple_AppStorePreflight_Bad(t *testing.T) {
	_, err := BuildApple(context.Background(), &Config{
		FS:         io.Local,
		ProjectDir: t.TempDir(),
		OutputDir:  ax.Join(t.TempDir(), "dist", "apple"),
		Name:       "Core",
		Version:    "v1.2.3",
	}, AppleOptions{
		BundleID:       "ai.lthn.core",
		Arch:           "arm64",
		Sign:           true,
		AppStore:       true,
		CertIdentity:   "Developer ID Application: Lethean CIC (ABC123DEF4)",
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
		APIKeyPath:     "/tmp/AuthKey_KEY123.p8",
		ProfilePath:    "/tmp/Core.provisionprofile",
		Category:       "public.app-category.developer-tools",
		Copyright:      "Copyright 2026 Lethean CIC. EUPL-1.2.",
	}, "42")
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "distribution certificate") {
		t.Fatalf("expected error %v to contain %v", err, "distribution certificate")
	}

}

func TestApple_BuildApple_TestFlightRequiresDistributionCertificate_Bad(t *testing.T) {
	_, err := BuildApple(context.Background(), &Config{
		FS:         io.Local,
		ProjectDir: t.TempDir(),
		OutputDir:  ax.Join(t.TempDir(), "dist", "apple"),
		Name:       "Core",
		Version:    "v1.2.3",
	}, AppleOptions{
		BundleID:       "ai.lthn.core",
		Arch:           "arm64",
		Sign:           true,
		TestFlight:     true,
		CertIdentity:   "Developer ID Application: Lethean CIC (ABC123DEF4)",
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
		APIKeyPath:     "/tmp/AuthKey_KEY123.p8",
		ProfilePath:    "/tmp/Core.provisionprofile",
	}, "42")
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "distribution certificate") {
		t.Fatalf("expected error %v to contain %v", err, "distribution certificate")
	}

}

func TestApple_BuildApple_AppStorePreflight_Good(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist", "apple")
	profilePath := ax.Join(projectDir, "Core.provisionprofile")
	if err := ax.WriteFile(profilePath, []byte("profile"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metadataPath := writeAppStoreMetadata(t, projectDir)

	oldBuildWails := appleBuildWailsAppFn
	oldSign := appleSignFn
	oldSubmit := appleSubmitAppStoreFn
	t.Cleanup(func() {
		appleBuildWailsAppFn = oldBuildWails
		appleSignFn = oldSign
		appleSubmitAppStoreFn = oldSubmit
	})

	appleBuildWailsAppFn = func(ctx context.Context, cfg WailsBuildConfig) (string, error) {
		appPath := ax.Join(cfg.OutputDir, cfg.Name+".app")
		writeDummyAppBundle(t, appPath, cfg.Name, "safe")
		return appPath, nil
	}
	appleSignFn = func(ctx context.Context, cfg SignConfig) error {
		return nil
	}

	var submitCfg AppStoreConfig
	var submitCalled bool
	appleSubmitAppStoreFn = func(ctx context.Context, cfg AppStoreConfig) error {
		submitCalled = true
		submitCfg = cfg
		return nil
	}

	result, err := BuildApple(context.Background(), &Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "Core",
		Version:    "v1.2.3",
	}, AppleOptions{
		BundleID:         "ai.lthn.core",
		Arch:             "arm64",
		Sign:             true,
		AppStore:         true,
		CertIdentity:     "Apple Distribution: Lethean CIC (ABC123DEF4)",
		APIKeyID:         "KEY123",
		APIKeyIssuerID:   "ISSUER456",
		APIKeyPath:       "/tmp/AuthKey_KEY123.p8",
		ProfilePath:      profilePath,
		MetadataPath:     metadataPath,
		PrivacyPolicyURL: "https://lthn.ai/privacy",
		Category:         "public.app-category.developer-tools",
		Copyright:        "Copyright 2026 Lethean CIC. EUPL-1.2.",
	}, "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdlibAssertNil(result) {
		t.Fatal("expected non-nil")
	}
	if !(submitCalled) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(result.BundlePath, submitCfg.AppPath) {
		t.Fatalf("want %v, got %v", result.BundlePath, submitCfg.AppPath)
	}
	if !stdlibAssertEqual("1.2.3", submitCfg.Version) {
		t.Fatalf("want %v, got %v", "1.2.3", submitCfg.Version)
	}
	if !stdlibAssertEqual("manual", submitCfg.ReleaseType) {
		t.Fatalf("want %v, got %v", "manual", submitCfg.ReleaseType)
	}

}

func TestApple_ValidatePrivacyPolicyURL_Bad(t *testing.T) {
	err := validatePrivacyPolicyURL("")
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "privacy_policy_url") {
		t.Fatalf("expected error %v to contain %v", err, "privacy_policy_url")
	}

	err = validatePrivacyPolicyURL("https://example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "non-root path") {
		t.Fatalf("expected error %v to contain %v", err, "non-root path")
	}

}

func TestApple_ValidateAppStoreMetadata_Bad(t *testing.T) {
	projectDir := t.TempDir()
	metadataPath := ax.Join(projectDir, ".core", "apple", "appstore")
	if err := io.Local.EnsureDir(ax.Join(metadataPath, "screenshots")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(metadataPath, "screenshots", "shot.png"), []byte("png"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := validateAppStoreMetadata(io.Local, projectDir, metadataPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "description") {
		t.Fatalf("expected error %v to contain %v", err, "description")
	}

}

func TestApple_ScanBundleForPrivateAPIUsage_Bad(t *testing.T) {
	appPath := ax.Join(t.TempDir(), "Core.app")
	writeDummyAppBundle(t, appPath, "Core", "/System/Library/PrivateFrameworks/Example.framework")

	err := scanBundleForPrivateAPIUsage(io.Local, appPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "private API usage detected") {
		t.Fatalf("expected error %v to contain %v", err, "private API usage detected")
	}

}

func TestApple_UploadTestFlight_Bad(t *testing.T) {
	err := UploadTestFlight(context.Background(), TestFlightConfig{
		AppPath:        "build/Core.app",
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "api_key_path") {
		t.Fatalf("expected error %v to contain %v", err, "api_key_path")
	}

}

func TestApple_SubmitAppStore_Bad(t *testing.T) {
	err := SubmitAppStore(context.Background(), AppStoreConfig{
		AppPath:        "build/Core.app",
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "api_key_path") {
		t.Fatalf("expected error %v to contain %v", err, "api_key_path")
	}

}

func TestApple_PackageForASCUpload_StagesAPIKeyWithCanonicalName_Good(t *testing.T) {
	keyPath := ax.Join(t.TempDir(), "lethean-app-store-key.p8")
	if err := ax.WriteFile(keyPath, []byte("private-key"), 0o600); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pkgPath := ax.Join(t.TempDir(), "Core.pkg")

	uploadPath, env, cleanup, err := packageForASCUpload(context.Background(), pkgPath, "", "KEY123", keyPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdlibAssertNil(cleanup) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual(pkgPath, uploadPath) {
		t.Fatalf("want %v, got %v", pkgPath, uploadPath)
	}
	if len(env) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(env))
	}

	stagedDir := envDirValue(t, env, "API_PRIVATE_KEYS_DIR")
	stagedPath := ax.Join(stagedDir, "AuthKey_KEY123.p8")
	content, err := io.Local.Read(stagedPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("private-key", content) {
		t.Fatalf("want %v, got %v", "private-key", content)
	}

	cleanup()
	if io.Local.Exists(stagedDir) {
		t.Fatal("expected false")
	}

}

func TestApple_PackageForASCUpload_UsesExistingCanonicalKeyPath_Good(t *testing.T) {
	keyDir := t.TempDir()
	keyPath := ax.Join(keyDir, "AuthKey_KEY123.p8")
	if err := ax.WriteFile(keyPath, []byte("private-key"), 0o600); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pkgPath := ax.Join(t.TempDir(), "Core.pkg")

	uploadPath, env, cleanup, err := packageForASCUpload(context.Background(), pkgPath, "", "KEY123", keyPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdlibAssertNil(cleanup) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual(pkgPath, uploadPath) {
		t.Fatalf("want %v, got %v", pkgPath, uploadPath)
	}
	if len(env) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(env))
	}
	if !stdlibAssertEqual(keyDir, envDirValue(t, env, "API_PRIVATE_KEYS_DIR")) {
		t.Fatalf("want %v, got %v", keyDir, envDirValue(t, env, "API_PRIVATE_KEYS_DIR"))
	}

	cleanup()
	if !(io.Local.Exists(keyDir)) {
		t.Fatal("expected true")
	}
	if !(io.Local.Exists(keyPath)) {
		t.Fatal("expected true")
	}

}

func writeDummyAppBundle(t *testing.T, appPath, executableName, marker string) {
	t.Helper()
	if err := io.Local.EnsureDir(ax.Join(appPath, "Contents", "MacOS")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := WriteInfoPlist(io.Local, appPath, InfoPlist{
		BundleID:                      "ai.lthn.core",
		BundleName:                    executableName,
		BundleDisplayName:             executableName,
		BundleVersion:                 "1.0.0",
		BuildNumber:                   "1",
		MinSystemVersion:              "13.0",
		Category:                      "public.app-category.developer-tools",
		Executable:                    executableName,
		HighResCapable:                true,
		SupportsSecureRestorableState: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(appPath, "Contents", "MacOS", executableName), []byte(marker), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func writeDummyExecutable(t *testing.T, path, marker string) {
	t.Helper()
	if err := io.Local.EnsureDir(ax.Dir(path)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(path, []byte(marker), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func writeAppStoreMetadata(t *testing.T, projectDir string) string {
	t.Helper()

	metadataPath := ax.Join(projectDir, ".core", "apple", "appstore")
	if err := io.Local.EnsureDir(ax.Join(metadataPath, "screenshots")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(metadataPath, "description.txt"), []byte("Core App Store description"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(metadataPath, "screenshots", "shot-1.png"), []byte("png"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return metadataPath
}

func envDirValue(t *testing.T, env []string, key string) string {
	t.Helper()

	prefix := key + "="
	for _, entry := range env {
		if value, ok := assertEnvEntry(entry, prefix); ok {
			return value
		}
	}

	t.Fatalf("environment variable %s not found", key)
	return ""
}

func assertEnvEntry(entry, prefix string) (string, bool) {
	if len(entry) <= len(prefix) || entry[:len(prefix)] != prefix {
		return "", false
	}
	return entry[len(prefix):], true
}

func indexOf(values []string, needle string) int {
	for i, value := range values {
		if value == needle {
			return i
		}
	}
	return -1
}

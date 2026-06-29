package buildcmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/testassert"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/signing"
	storage "dappco.re/go/build/pkg/storage"
)

// --- loadAppleBuildConfig (cmd_apple.go) ---

func TestCmdApple_loadAppleBuildConfig_Good(t *core.T) {
	// With no explicit config path, the project's .core/build.yaml is loaded.
	projectDir := t.TempDir()
	requireBuildCmdOK(t, ax.MkdirAll(ax.Join(projectDir, ".core"), 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(`version: 1
project:
  name: Demo
  binary: demo
`), 0o644))

	result := loadAppleBuildConfig(storage.Local, projectDir, "")
	core.AssertTrue(t, result.OK)
	cfg := result.Value.(*build.BuildConfig)
	core.AssertEqual(t, "demo", cfg.Project.Binary)
}

func TestCmdApple_loadAppleBuildConfig_Bad(t *core.T) {
	// An explicit but non-existent config path is reported as not found.
	projectDir := t.TempDir()
	result := loadAppleBuildConfig(storage.Local, projectDir, "missing.yaml")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "build config not found")
}

func TestCmdApple_loadAppleBuildConfig_Ugly(t *core.T) {
	// Edge case: an explicit (relative) config path that exists is loaded from
	// the project directory.
	projectDir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "custom.yaml"), []byte(`version: 1
project:
  name: Custom
  binary: custom
`), 0o644))

	result := loadAppleBuildConfig(storage.Local, projectDir, "custom.yaml")
	core.AssertTrue(t, result.OK)
	cfg := result.Value.(*build.BuildConfig)
	core.AssertEqual(t, "custom", cfg.Project.Binary)
}

// --- validateAppleBuildNumber (cmd_apple.go) ---

func TestCmdApple_validateAppleBuildNumber_Good(t *core.T) {
	core.AssertTrue(t, validateAppleBuildNumber("42").OK)
}

func TestCmdApple_validateAppleBuildNumber_Bad(t *core.T) {
	// Non-numeric build numbers are rejected.
	result := validateAppleBuildNumber("1.2.3")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "positive integer")
}

func TestCmdApple_validateAppleBuildNumber_Ugly(t *core.T) {
	// Edge case: an empty string and a value with whitespace are both invalid.
	core.AssertFalse(t, validateAppleBuildNumber("").OK)
	core.AssertFalse(t, validateAppleBuildNumber("12 ").OK)
}

func TestBuildCmd_resolveAppleCommandOptions_Good(t *testing.T) {
	cfg := &build.BuildConfig{
		Apple: build.AppleConfig{
			BundleID: "ai.lthn.core",
			Arch:     "arm64",
			Sign:     boolPtr(false),
		},
		Sign: signing.SignConfig{
			MacOS: signing.MacOSConfig{
				Identity:    "Developer ID Application: Lethean CIC (ABC123DEF4)",
				TeamID:      "ABC123DEF4",
				AppleID:     "dev@example.com",
				AppPassword: "secret",
			},
		},
	}

	options := resolveAppleCommandOptions(cfg, appleCLIOptions{})
	if !stdlibAssertEqual("ai.lthn.core", options.BundleID) {
		t.Fatalf("want %v, got %v", "ai.lthn.core", options.BundleID)
	}
	if !stdlibAssertEqual("arm64", options.Arch) {
		t.Fatalf("want %v, got %v", "arm64", options.Arch)
	}
	if options.Sign {
		t.Fatal("expected false")
	}
	if !stdlibAssertEqual("Developer ID Application: Lethean CIC (ABC123DEF4)", options.CertIdentity) {
		t.Fatalf("want %v, got %v", "Developer ID Application: Lethean CIC (ABC123DEF4)", options.CertIdentity)
	}
	if !stdlibAssertEqual("ABC123DEF4", options.TeamID) {
		t.Fatalf("want %v, got %v", "ABC123DEF4", options.TeamID)
	}
	if !stdlibAssertEqual("dev@example.com", options.AppleID) {
		t.Fatalf("want %v, got %v", "dev@example.com", options.AppleID)
	}
	if !stdlibAssertEqual("secret", options.Password) {
		t.Fatalf("want %v, got %v", "secret", options.Password)
	}

	options = resolveAppleCommandOptions(cfg, appleCLIOptions{
		Arch:              "universal",
		ArchChanged:       true,
		Sign:              true,
		SignChanged:       true,
		BundleID:          "ai.lthn.core.preview",
		BundleIDChanged:   true,
		TeamID:            "ZZZ9876543",
		TeamIDChanged:     true,
		TestFlight:        true,
		TestFlightChanged: true,
	})
	if !stdlibAssertEqual("universal", options.Arch) {
		t.Fatalf("want %v, got %v", "universal", options.Arch)
	}
	if !(options.Sign) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("ai.lthn.core.preview", options.BundleID) {
		t.Fatalf("want %v, got %v", "ai.lthn.core.preview", options.BundleID)
	}
	if !stdlibAssertEqual("ZZZ9876543", options.TeamID) {
		t.Fatalf("want %v, got %v", "ZZZ9876543", options.TeamID)
	}
	if !(options.TestFlight) {
		t.Fatal("expected true")
	}

}

func TestBuildCmd_resolveAppleBuildNumber_Good(t *testing.T) {
	t.Run("prefers github run number when valid", func(t *testing.T) {
		t.Setenv("GITHUB_RUN_NUMBER", "77")
		value := requireBuildCmdString(t, resolveAppleBuildNumber(context.Background(), t.TempDir()))
		if !stdlibAssertEqual("77", value) {
			t.Fatalf("want %v, got %v", "77", value)
		}

	})

	t.Run("falls back to git commit count", func(t *testing.T) {
		dir := t.TempDir()
		runGit(t, dir, "init")
		runGit(t, dir, "config", "user.email", "test@example.com")
		runGit(t, dir, "config", "user.name", "Test User")
		requireBuildCmdOK(t, ax.WriteFile(ax.Join(dir, "README.md"), []byte("hello\n"), 0o644))

		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "feat: initial commit")

		t.Setenv("GITHUB_RUN_NUMBER", "")
		value := requireBuildCmdString(t, resolveAppleBuildNumber(context.Background(), dir))
		if !stdlibAssertEqual("1", value) {
			t.Fatalf("want %v, got %v", "1", value)
		}

	})
}

func TestBuildCmd_AddAppleCommand_Good(t *testing.T) {
	c := core.New()
	AddAppleCommand(c)

	result := c.Command("build/apple")
	if !(result.OK) {
		t.Fatal("expected true")
	}

	command, ok := result.Value.(*core.Command)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("build/apple", command.Path) {
		t.Fatalf("want %v, got %v", "build/apple", command.Path)
	}
	if !stdlibAssertEqual("cmd.build.apple.long", command.Description) {
		t.Fatalf("want %v, got %v", "cmd.build.apple.long", command.Description)
	}

}

func TestBuildCmd_runAppleBuildInDir_Good(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	requireBuildCmdOK(t, ax.MkdirAll(coreDir, 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
  sign: false
sign:
  macos:
    identity: "Developer ID Application: Lethean CIC (ABC123DEF4)"
    team_id: ABC123DEF4
    apple_id: dev@example.com
    app_password: secret
`), 0o644))

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	var called bool
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) core.Result {
		called = true
		if !stdlibAssertEqual(ax.Join(projectDir, "out"), cfg.OutputDir) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "out"), cfg.OutputDir)
		}
		if !stdlibAssertEqual("Core", cfg.Name) {
			t.Fatalf("want %v, got %v", "Core", cfg.Name)
		}
		if !stdlibAssertEqual("v1.2.3", cfg.Version) {
			t.Fatalf("want %v, got %v", "v1.2.3", cfg.Version)
		}
		if !stdlibAssertEqual("42", buildNumber) {
			t.Fatalf("want %v, got %v", "42", buildNumber)
		}
		if !stdlibAssertEqual("ai.lthn.core", options.BundleID) {
			t.Fatalf("want %v, got %v", "ai.lthn.core", options.BundleID)
		}
		if !(options.Sign) {
			t.Fatal("expected true")
		}

		return core.Ok(&build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		})
	}

	requireBuildCmdOK(t, runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Sign:        true,
		SignChanged: true,
		Version:     "v1.2.3",
		BuildNumber: "42",
		OutputDir:   "out",
	}))
	if !(called) {
		t.Fatal("expected true")
	}

}

func TestBuildCmd_runAppleBuildInDir_RejectsUnsafeVersion_Bad(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	requireBuildCmdOK(t, ax.MkdirAll(coreDir, 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
  sign: false
`), 0o644))

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) core.Result {
		t.Fatal("buildAppleFn must not be called for unsafe versions")
		return core.Ok(nil)
	}

	message := requireBuildCmdError(t, runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Version:     "v1.2.3 --bad",
		BuildNumber: "42",
	}))
	if !stdlibAssertContains(message, "invalid build version") {
		t.Fatalf("expected %v to contain %v", message, "invalid build version")
	}

}

func TestBuildCmd_runAppleBuildInDir_SetsUpBuildCache_Good(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	requireBuildCmdOK(t, ax.MkdirAll(coreDir, 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
build:
  cache:
    enabled: true
    paths:
      - cache/go-build
      - cache/go-mod
apple:
  bundle_id: ai.lthn.core
  sign: false
`), 0o644))

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) core.Result {
		if !stdlibAssertEqual([]string{ax.Join(projectDir, "cache", "go-build"), ax.Join(projectDir, "cache", "go-mod")}, cfg.Cache.Paths) {
			t.Fatalf("want %v, got %v", []string{ax.Join(projectDir, "cache", "go-build"), ax.Join(projectDir, "cache", "go-mod")}, cfg.Cache.Paths)
		}
		if !(cfg.Cache.Enabled) {
			t.Fatal("expected true")
		}
		if !(cfg.FS.Exists(ax.Join(projectDir, ".core", "cache"))) {
			t.Fatal("expected true")
		}
		if !(cfg.FS.Exists(ax.Join(projectDir, "cache", "go-build"))) {
			t.Fatal("expected true")
		}
		if !(cfg.FS.Exists(ax.Join(projectDir, "cache", "go-mod"))) {
			t.Fatal("expected true")
		}

		return core.Ok(&build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		})
	}

	requireBuildCmdOK(t, runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Version:     "v1.2.3",
		BuildNumber: "42",
	}))

}

func TestBuildCmd_runAppleBuildInDir_WritesXcodeCloudScripts_Good(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	requireBuildCmdOK(t, ax.MkdirAll(coreDir, 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
  sign: false
  xcode_cloud:
    workflow: CoreGUI Release
`), 0o644))

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) core.Result {
		return core.Ok(&build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		})
	}

	requireBuildCmdOK(t, runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Version:     "v1.2.3",
		BuildNumber: "42",
	}))

	preScriptPath := ax.Join(projectDir, build.XcodeCloudScriptsDir, build.XcodeCloudPreXcodebuildScriptName)
	preScript := requireBuildCmdBytes(t, ax.ReadFile(preScriptPath))
	if !stdlibAssertContains(string(preScript), `core build apple --arch 'universal' --config '.core/build.yaml'`) {
		t.Fatalf("expected %v to contain %v", string(preScript), `core build apple --arch 'universal' --config '.core/build.yaml'`)
	}

}

func boolPtr(value bool) *bool {
	return &value
}

var (
	stdlibAssertEqual         = testassert.Equal
	stdlibAssertNil           = testassert.Nil
	stdlibAssertEmpty         = testassert.Empty
	stdlibAssertContains      = testassert.Contains
	stdlibAssertElementsMatch = testassert.ElementsMatch
)

// --- AddAppleCommand (meaningful) ---

func TestCmdApple_AddAppleCommand_Good(t *core.T) {
	c := core.New()
	result := AddAppleCommand(c)
	core.AssertTrue(t, result.OK)
	registered := c.Command("build/apple")
	core.AssertTrue(t, registered.OK)
	cmd := registered.Value.(*core.Command)
	core.AssertEqual(t, "cmd.build.apple.long", cmd.Description)
	core.AssertNotNil(t, cmd.Action)
}

func TestCmdApple_AddAppleCommand_Bad(t *core.T) {
	// Re-registering the same executable command path is rejected.
	c := core.New()
	core.AssertTrue(t, AddAppleCommand(c).OK)
	result := AddAppleCommand(c)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "already registered")
}

func TestCmdApple_AddAppleCommand_Ugly(t *core.T) {
	// Edge case: build/apple coexists with an unrelated pre-registered command.
	c := core.New()
	core.AssertTrue(t, c.Command("build/other", core.Command{
		Action: func(core.Options) core.Result { return core.Ok(nil) },
	}).OK)
	core.AssertTrue(t, AddAppleCommand(c).OK)
	core.AssertTrue(t, c.Command("build/apple").OK)
	core.AssertTrue(t, c.Command("build/other").OK)
}

// TestCmdApple_AddAppleCommand_ActionWired drives the registered build/apple
// action (and thus runAppleBuild) through to the BuildApple call. The test
// working directory has no Apple configuration, so the build fails fast with a
// configuration error rather than invoking the macOS toolchain.
func TestCmdApple_AddAppleCommand_ActionWired(t *core.T) {
	c := core.New()
	core.AssertTrue(t, AddAppleCommand(c).OK)
	captureBuildStdout(t)

	result := c.Command("build/apple").Value.(*core.Command).Run(core.NewOptions(
		core.Option{Key: "version", Value: "v1.0.0"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "bundle_id is required")
}

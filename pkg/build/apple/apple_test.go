package apple

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/testassert"
	build "dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/signing"
	coreio "dappco.re/go/io"
)

func TestAppleBuilder_New_Good(t *testing.T) {
	builder := New(
		WithArch("arm64"),
		WithSign(false),
		WithNotarise(false),
		WithDMG(true),
		WithTestFlight(true),
		WithAppStore(true),
	)
	if stdlibAssertNil(builder) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("apple", builder.Name()) {
		t.Fatalf("want %v, got %v", "apple", builder.Name())
	}
	if stdlibAssertNil(builder.ServiceRuntime) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("arm64", builder.options.Arch) {
		t.Fatalf("want %v, got %v", "arm64", builder.options.Arch)
	}
	if !stdlibAssertEqual("arm64", builder.Options().Arch) {
		t.Fatalf("want %v, got %v", "arm64", builder.Options().Arch)
	}
	if builder.options.Sign {
		t.Fatal("expected false")
	}
	if builder.options.Notarise {
		t.Fatal("expected false")
	}
	if !(builder.options.DMG) {
		t.Fatal("expected true")
	}
	if !(builder.options.TestFlight) {
		t.Fatal("expected true")
	}
	if !(builder.options.AppStore) {
		t.Fatal("expected true")
	}
	if !(builder.explicit.arch) {
		t.Fatal("expected true")
	}
	if !(builder.explicit.sign) {
		t.Fatal("expected true")
	}
	if !(builder.explicit.notarise) {
		t.Fatal("expected true")
	}
	if !(builder.explicit.dmg) {
		t.Fatal("expected true")
	}
	if !(builder.explicit.testFlight) {
		t.Fatal("expected true")
	}
	if !(builder.explicit.appStore) {
		t.Fatal("expected true")
	}

}

func TestAppleBuilder_New_PreservesExplicitDefaultValuedOptions_Good(t *testing.T) {
	builder := New(
		WithArch("universal"),
		WithSign(true),
		WithNotarise(true),
	)
	if stdlibAssertNil(builder) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("universal", builder.options.Arch) {
		t.Fatalf("want %v, got %v", "universal", builder.options.Arch)
	}
	if !(builder.options.Sign) {
		t.Fatal("expected true")
	}
	if !(builder.options.Notarise) {
		t.Fatal("expected true")
	}
	if !(builder.explicit.arch) {
		t.Fatal("expected true")
	}
	if !(builder.explicit.sign) {
		t.Fatal("expected true")
	}
	if !(builder.explicit.notarise) {
		t.Fatal("expected true")
	}

}

func TestAppleBuilder_Register_Good(t *testing.T) {
	c := core.New()

	result := Register(c)
	if !(result.OK) {
		t.Fatal("expected true")
	}

	builder, ok := result.Value.(*AppleBuilder)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("apple", builder.Name()) {
		t.Fatalf("want %v, got %v", "apple", builder.Name())
	}
	if stdlibAssertNil(builder.ServiceRuntime) {
		t.Fatal("expected non-nil")
	}
	if c != builder.Core() {
		t.Fatalf("expected %v and %v to be the same", c, builder.Core())
	}
	if !(c.Service("apple").OK) {
		t.Fatal("expected true")
	}
	if !(c.RegistryOf("services").Has("apple")) {
		t.Fatal("expected true")
	}

}

func TestAppleBuilder_Detect_Good(t *testing.T) {
	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := New().Detect(coreio.Local, dir)
	if !(result.OK) {
		t.Fatal("expected true")
	}

	detected, ok := result.Value.(bool)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !(detected) {
		t.Fatal("expected true")
	}

}

func TestAppleBuilder_Build_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldLoadConfig := loadConfigFn
	oldBuildApple := buildAppleFn
	oldDetermineVersion := determineVersion
	oldGetwd := getwdFn
	oldRunDir := runDirFn
	oldWriteXcodeCloudScripts := writeXcodeCloudScriptsFn
	t.Cleanup(func() {
		loadConfigFn = oldLoadConfig
		buildAppleFn = oldBuildApple
		determineVersion = oldDetermineVersion
		getwdFn = oldGetwd
		runDirFn = oldRunDir
		writeXcodeCloudScriptsFn = oldWriteXcodeCloudScripts
	})

	loadConfigFn = func(fs coreio.Medium, dir string) (*build.BuildConfig, error) {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return &build.BuildConfig{
			Project: build.Project{
				Name:   "Core",
				Binary: "Core",
			},
			Build: build.Build{
				LDFlags: []string{"-s", "-w"},
			},
			Apple: build.AppleConfig{
				BundleID: "ai.lthn.core",
				Sign:     boolPtr(false),
			},
			Sign: signing.SignConfig{
				MacOS: signing.MacOSConfig{
					Identity: "Developer ID Application: Lethean CIC (ABC123DEF4)",
					TeamID:   "ABC123DEF4",
				},
			},
		}, nil
	}
	determineVersion = func(ctx context.Context, dir string) (string, error) {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return "v1.2.3", nil
	}
	getwdFn = func() (string, error) {
		return projectDir, nil
	}
	runDirFn = func(ctx context.Context, dir, command string, args ...string) (string, error) {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}
		if !stdlibAssertEqual("git", command) {
			t.Fatalf("want %v, got %v", "git", command)
		}
		if !stdlibAssertEqual([]string{"rev-list", "--count", "HEAD"}, args) {
			t.Fatalf("want %v, got %v", []string{"rev-list", "--count", "HEAD"}, args)
		}

		return "42", nil
	}
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		if !stdlibAssertEqual(ax.Join(projectDir, "dist", "apple"), cfg.OutputDir) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", "apple"), cfg.OutputDir)
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
		if !stdlibAssertEqual("arm64", options.Arch) {
			t.Fatalf("want %v, got %v", "arm64", options.Arch)
		}

		return &build.AppleBuildResult{
			BundlePath: ax.Join(cfg.OutputDir, "Core.app"),
		}, nil
	}

	result := New(WithArch("arm64"), WithSign(true)).Build(context.Background(), nil)
	if !(result.OK) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value)
	}

}

func TestAppleBuilder_Build_PartialRuntimeOptionsPreservePipelineDefaults_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldLoadConfig := loadConfigFn
	oldBuildApple := buildAppleFn
	oldDetermineVersion := determineVersion
	oldGetwd := getwdFn
	oldRunDir := runDirFn
	t.Cleanup(func() {
		loadConfigFn = oldLoadConfig
		buildAppleFn = oldBuildApple
		determineVersion = oldDetermineVersion
		getwdFn = oldGetwd
		runDirFn = oldRunDir
	})

	loadConfigFn = func(fs coreio.Medium, dir string) (*build.BuildConfig, error) {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return &build.BuildConfig{
			Project: build.Project{
				Name:   "Core",
				Binary: "Core",
			},
			Apple: build.AppleConfig{
				BundleID: "ai.lthn.core",
				DMG:      boolPtr(true),
			},
			Sign: signing.SignConfig{
				MacOS: signing.MacOSConfig{
					Identity: "Developer ID Application: Lethean CIC (ABC123DEF4)",
					TeamID:   "ABC123DEF4",
				},
			},
		}, nil
	}
	determineVersion = func(ctx context.Context, dir string) (string, error) {
		return "v1.2.3", nil
	}
	getwdFn = func() (string, error) {
		return projectDir, nil
	}
	runDirFn = func(ctx context.Context, dir, command string, args ...string) (string, error) {
		return "42", nil
	}
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		if !stdlibAssertEqual("ai.lthn.override", options.BundleID) {
			t.Fatalf("want %v, got %v", "ai.lthn.override", options.BundleID)
		}
		if !(options.Sign) {
			t.Fatal("expected true")
		}
		if !(options.Notarise) {
			t.Fatal("expected true")
		}
		if !(options.DMG) {
			t.Fatal("expected true")
		}
		if options.TestFlight {
			t.Fatal("expected false")
		}
		if options.AppStore {
			t.Fatal("expected false")
		}

		return &build.AppleBuildResult{
			BundlePath: ax.Join(cfg.OutputDir, "Core.app"),
		}, nil
	}

	result := New().Build(context.Background(), &AppleOptions{
		BundleID: "ai.lthn.override",
	})
	if !(result.OK) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value)
	}

}

func TestAppleBuilder_Build_SetsUpBuildCache_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldLoadConfig := loadConfigFn
	oldBuildApple := buildAppleFn
	oldDetermineVersion := determineVersion
	oldGetwd := getwdFn
	oldRunDir := runDirFn
	t.Cleanup(func() {
		loadConfigFn = oldLoadConfig
		buildAppleFn = oldBuildApple
		determineVersion = oldDetermineVersion
		getwdFn = oldGetwd
		runDirFn = oldRunDir
	})

	loadConfigFn = func(fs coreio.Medium, dir string) (*build.BuildConfig, error) {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return &build.BuildConfig{
			Project: build.Project{
				Name:   "Core",
				Binary: "Core",
			},
			Build: build.Build{
				Cache: build.CacheConfig{
					Enabled: true,
					Paths: []string{
						"cache/go-build",
						"cache/go-mod",
					},
				},
			},
			Apple: build.AppleConfig{
				BundleID: "ai.lthn.core",
				Sign:     boolPtr(false),
			},
		}, nil
	}
	determineVersion = func(ctx context.Context, dir string) (string, error) {
		return "v1.2.3", nil
	}
	getwdFn = func() (string, error) {
		return projectDir, nil
	}
	runDirFn = func(ctx context.Context, dir, command string, args ...string) (string, error) {
		return "42", nil
	}
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		if !stdlibAssertEqual([]string{ax.Join(projectDir, "cache", "go-build"), ax.Join(projectDir, "cache", "go-mod")}, cfg.Cache.Paths) {
			t.Fatalf("want %v, got %v", []string{ax.Join(projectDir, "cache", "go-build"), ax.Join(projectDir, "cache", "go-mod")}, cfg.Cache.Paths)
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

		return &build.AppleBuildResult{
			BundlePath: ax.Join(cfg.OutputDir, "Core.app"),
		}, nil
	}

	result := New().Build(context.Background(), nil)
	if !(result.OK) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value)
	}

}

func TestAppleBuilder_Build_WritesXcodeCloudScripts_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldLoadConfig := loadConfigFn
	oldBuildApple := buildAppleFn
	oldDetermineVersion := determineVersion
	oldGetwd := getwdFn
	oldRunDir := runDirFn
	oldWriteXcodeCloudScripts := writeXcodeCloudScriptsFn
	t.Cleanup(func() {
		loadConfigFn = oldLoadConfig
		buildAppleFn = oldBuildApple
		determineVersion = oldDetermineVersion
		getwdFn = oldGetwd
		runDirFn = oldRunDir
		writeXcodeCloudScriptsFn = oldWriteXcodeCloudScripts
	})

	loadConfigFn = func(fs coreio.Medium, dir string) (*build.BuildConfig, error) {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return &build.BuildConfig{
			Project: build.Project{
				Name:   "Core",
				Binary: "Core",
			},
			Apple: build.AppleConfig{
				BundleID: "ai.lthn.core",
				Sign:     boolPtr(false),
				XcodeCloud: build.XcodeCloudConfig{
					Workflow: "CoreGUI Release",
				},
			},
		}, nil
	}
	determineVersion = func(ctx context.Context, dir string) (string, error) {
		return "v1.2.3", nil
	}
	getwdFn = func() (string, error) {
		return projectDir, nil
	}
	runDirFn = func(ctx context.Context, dir, command string, args ...string) (string, error) {
		return "42", nil
	}

	var scriptsWritten bool
	writeXcodeCloudScriptsFn = func(fs coreio.Medium, dir string, cfg *build.BuildConfig) ([]string, error) {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}
		if !stdlibAssertEqual("CoreGUI Release", cfg.Apple.XcodeCloud.Workflow) {
			t.Fatalf("want %v, got %v", "CoreGUI Release", cfg.Apple.XcodeCloud.Workflow)
		}

		scriptsWritten = true
		return []string{ax.Join(dir, build.XcodeCloudScriptsDir, build.XcodeCloudPreXcodebuildScriptName)}, nil
	}
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		return &build.AppleBuildResult{
			BundlePath: ax.Join(cfg.OutputDir, "Core.app"),
		}, nil
	}

	result := New().Build(context.Background(), nil)
	if !(result.OK) {
		t.Fatal("expected true")
	}
	if !(scriptsWritten) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value)
	}

}

func TestAppleBuilder_resolveOptions_BoolOnlyRuntimeOverride_Good(t *testing.T) {
	builder := New()

	options := builder.resolveOptions(&build.BuildConfig{
		Apple: build.AppleConfig{
			BundleID: "ai.lthn.core",
			DMG:      boolPtr(true),
		},
	}, &AppleOptions{
		Sign:     false,
		Notarise: false,
		DMG:      false,
		AppStore: true,
	})
	if options.Sign {
		t.Fatal("expected false")
	}
	if options.Notarise {
		t.Fatal("expected false")
	}
	if options.DMG {
		t.Fatal("expected false")
	}
	if !(options.AppStore) {
		t.Fatal("expected true")
	}

}

func TestApple_BuildWailsApp_UsesCurrentDirectoryAndStringLDFlags_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldBuildWails := buildWailsAppFn
	oldGetwd := getwdFn
	t.Cleanup(func() {
		buildWailsAppFn = oldBuildWails
		getwdFn = oldGetwd
	})

	getwdFn = func() (string, error) {
		return projectDir, nil
	}

	buildWailsAppFn = func(ctx context.Context, cfg build.WailsBuildConfig) (string, error) {
		if !stdlibAssertEqual(projectDir, cfg.ProjectDir) {
			t.Fatalf("want %v, got %v", projectDir, cfg.ProjectDir)
		}
		if !stdlibAssertEqual("Core", cfg.Name) {
			t.Fatalf("want %v, got %v", "Core", cfg.Name)
		}
		if !stdlibAssertEqual("arm64", cfg.Arch) {
			t.Fatalf("want %v, got %v", "arm64", cfg.Arch)
		}
		if !stdlibAssertEqual([]string{"integration"}, cfg.BuildTags) {
			t.Fatalf("want %v, got %v", []string{"integration"}, cfg.BuildTags)
		}
		if !stdlibAssertEqual([]string{"-s -w -X main.version=1.2.3"}, cfg.LDFlags) {
			t.Fatalf("want %v, got %v", []string{"-s -w -X main.version=1.2.3"}, cfg.LDFlags)
		}
		if !stdlibAssertEqual("1.2.3", cfg.Version) {
			t.Fatalf("want %v, got %v", "1.2.3", cfg.Version)
		}
		if !stdlibAssertEqual([]string{"FOO=bar"}, cfg.Env) {
			t.Fatalf("want %v, got %v", []string{"FOO=bar"}, cfg.Env)
		}

		return ax.Join(projectDir, "dist", "Core.app"), nil
	}

	result := BuildWailsApp(context.Background(), WailsBuildConfig{
		Name:      "Core",
		Arch:      "arm64",
		BuildTags: []string{"integration"},
		LDFlags:   "-s -w -X main.version=1.2.3",
		OutputDir: ax.Join(projectDir, "dist"),
		Version:   "1.2.3",
		Env:       []string{"FOO=bar"},
	})
	if !(result.OK) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "dist", "Core.app"), result.Value) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", "Core.app"), result.Value)
	}

}

func boolPtr(value bool) *bool {
	return &value
}

var (
	stdlibAssertEqual         = testassert.Equal
	stdlibAssertNil           = testassert.Nil
	stdlibAssertEmpty         = testassert.Empty
	stdlibAssertZero          = testassert.Zero
	stdlibAssertContains      = testassert.Contains
	stdlibAssertElementsMatch = testassert.ElementsMatch
)

// --- v0.9.0 generated compliance triplets ---
func TestApple_Register_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Register(core.New())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_Register_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Register(core.New())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_Register_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Register(core.New())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_New_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = New()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_New_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = New()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_New_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = New()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithArch_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithArch("amd64")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithArch_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithArch("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithArch_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithArch("amd64")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithSign_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithSign(true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithSign_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithSign(false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithSign_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithSign(true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithNotarise_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithNotarise(true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithNotarise_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithNotarise(false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithNotarise_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithNotarise(true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithDMG_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithDMG(true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithDMG_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithDMG(false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithDMG_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithDMG(true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithTestFlight_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithTestFlight(true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithTestFlight_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithTestFlight(false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithTestFlight_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithTestFlight(true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithAppStore_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppStore(true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithAppStore_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppStore(false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithAppStore_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppStore(true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_AppleBuilder_Name_Good(t *core.T) {
	subject := &AppleBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_AppleBuilder_Name_Bad(t *core.T) {
	subject := &AppleBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_AppleBuilder_Name_Ugly(t *core.T) {
	subject := &AppleBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_AppleBuilder_Detect_Good(t *core.T) {
	subject := &AppleBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_AppleBuilder_Detect_Bad(t *core.T) {
	subject := &AppleBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(coreio.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_AppleBuilder_Detect_Ugly(t *core.T) {
	subject := &AppleBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_AppleBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AppleBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, &AppleOptions{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_AppleBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AppleBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_AppleBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AppleBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, &AppleOptions{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_BuildWailsApp_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = BuildWailsApp(ctx, WailsBuildConfig{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_BuildWailsApp_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = BuildWailsApp(ctx, WailsBuildConfig{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_BuildWailsApp_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = BuildWailsApp(ctx, WailsBuildConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_CreateUniversal_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CreateUniversal(core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_CreateUniversal_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CreateUniversal("", "", "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_CreateUniversal_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CreateUniversal(core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_Sign_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Sign(ctx, SignConfig{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_Sign_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Sign(ctx, SignConfig{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_Sign_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Sign(ctx, SignConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_Notarise_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Notarise(ctx, NotariseConfig{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_Notarise_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Notarise(ctx, NotariseConfig{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_Notarise_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Notarise(ctx, NotariseConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_CreateDMG_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CreateDMG(ctx, DMGConfig{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_CreateDMG_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CreateDMG(ctx, DMGConfig{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_CreateDMG_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CreateDMG(ctx, DMGConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_UploadTestFlight_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = UploadTestFlight(ctx, TestFlightConfig{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_UploadTestFlight_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = UploadTestFlight(ctx, TestFlightConfig{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_UploadTestFlight_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = UploadTestFlight(ctx, TestFlightConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_SubmitAppStore_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = SubmitAppStore(ctx, AppStoreConfig{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_SubmitAppStore_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = SubmitAppStore(ctx, AppStoreConfig{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_SubmitAppStore_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = SubmitAppStore(ctx, AppStoreConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

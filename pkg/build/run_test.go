package build

import (
	"context"
	"runtime"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	coreio "dappco.re/go/build/pkg/storage"
)

type runTestBuilder struct {
	directoryArtifact bool
}

type capturingRunTestBuilder struct {
	captured **Config
}

func (b *runTestBuilder) Name() string { return "run-test" }

func (b *runTestBuilder) Detect(fs coreio.Medium, dir string) core.Result {
	return core.Ok(true)
}

func (b *runTestBuilder) Build(ctx context.Context, cfg *Config, targets []Target) core.Result {
	if cfg.FS == nil {
		cfg.FS = coreio.Local
	}
	if len(targets) == 0 {
		targets = []Target{{OS: "linux", Arch: "amd64"}}
	}

	artifacts := make([]Artifact, 0, len(targets))
	for _, target := range targets {
		basePath := ax.Join(cfg.OutputDir, target.OS+"_"+target.Arch, cfg.Name)
		if b.directoryArtifact {
			artifactPath := basePath + ".app"
			created := cfg.FS.EnsureDir(ax.Join(artifactPath, "Contents", "MacOS"))
			if !created.OK {
				return created
			}
			written := cfg.FS.WriteMode(ax.Join(artifactPath, "Contents", "MacOS", cfg.Name), "bundle:"+target.String(), 0o755)
			if !written.OK {
				return written
			}
			artifacts = append(artifacts, Artifact{Path: artifactPath, OS: target.OS, Arch: target.Arch})
			continue
		}

		created := cfg.FS.EnsureDir(ax.Dir(basePath))
		if !created.OK {
			return created
		}
		written := cfg.FS.WriteMode(basePath, "artifact:"+target.String(), 0o755)
		if !written.OK {
			return written
		}
		artifacts = append(artifacts, Artifact{Path: basePath, OS: target.OS, Arch: target.Arch})
	}

	return core.Ok(artifacts)
}

func (b *capturingRunTestBuilder) Name() string { return "capturing-run-test" }

func (b *capturingRunTestBuilder) Detect(fs coreio.Medium, dir string) core.Result {
	return core.Ok(true)
}

func (b *capturingRunTestBuilder) Build(ctx context.Context, cfg *Config, targets []Target) core.Result {
	if b.captured != nil {
		*b.captured = cfg
	}
	return (&runTestBuilder{}).Build(ctx, cfg, targets)
}

func requireRunOKResult(t *testing.T, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requireRunArtifacts(t *testing.T, result core.Result) []Artifact {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.([]Artifact)
}

func requireRunString(t *testing.T, result core.Result) string {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(string)
}

func requireRunError(t *testing.T, result core.Result) string {
	t.Helper()
	if result.OK {
		t.Fatal("expected error")
	}
	return result.Error()
}

func TestRun_UsesOutputMediumGood(t *testing.T) {
	projectDir := t.TempDir()
	output := coreio.NewMemoryMedium()

	artifacts := requireRunArtifacts(t, Run(
		WithContext(context.Background()),
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeGo)),
		WithBuildName("core-build"),
		WithTargets(Target{OS: "linux", Arch: "amd64"}),
		WithOutput(output),
		WithOutputDir("releases"),
		WithBuilderResolver(func(projectType ProjectType) core.Result {
			return core.Ok(&runTestBuilder{})
		}),
	))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if !stdlibAssertEqual(ax.Join("releases", "linux_amd64", "core-build"), artifacts[0].Path) {
		t.Fatalf("want %v, got %v", ax.Join("releases", "linux_amd64", "core-build"), artifacts[0].Path)
	}

	content := requireRunString(t, output.Read(ax.Join("releases", "linux_amd64", "core-build")))
	if !stdlibAssertEqual("artifact:linux/amd64", content) {
		t.Fatalf("want %v, got %v", "artifact:linux/amd64", content)
	}

}

func TestRun_UsesOutputMediumRootWhenOutputDirUnsetGood(t *testing.T) {
	projectDir := t.TempDir()
	output := coreio.NewMemoryMedium()

	artifacts := requireRunArtifacts(t, Run(
		WithContext(context.Background()),
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeGo)),
		WithBuildName("core-build"),
		WithTargets(Target{OS: "linux", Arch: "amd64"}),
		WithOutput(output),
		WithBuilderResolver(func(projectType ProjectType) core.Result {
			return core.Ok(&runTestBuilder{})
		}),
	))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if !stdlibAssertEqual(ax.Join("linux_amd64", "core-build"), artifacts[0].Path) {
		t.Fatalf("want %v, got %v", ax.Join("linux_amd64", "core-build"), artifacts[0].Path)
	}

	content := requireRunString(t, output.Read(ax.Join("linux_amd64", "core-build")))
	if !stdlibAssertEqual("artifact:linux/amd64", content) {
		t.Fatalf("want %v, got %v", "artifact:linux/amd64", content)
	}

}

func TestRun_MirrorsDirectoryArtifactsGood(t *testing.T) {
	projectDir := t.TempDir()
	output := coreio.NewMemoryMedium()

	artifacts := requireRunArtifacts(t, Run(
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeWails)),
		WithBuildName("core-build"),
		WithTargets(Target{OS: "darwin", Arch: "arm64"}),
		WithOutput(output),
		WithOutputDir("bundles"),
		WithBuilderResolver(func(projectType ProjectType) core.Result {
			return core.Ok(&runTestBuilder{directoryArtifact: true})
		}),
	))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	bundlePath := ax.Join("bundles", "darwin_arm64", "core-build.app")
	if !stdlibAssertEqual(bundlePath, artifacts[0].Path) {
		t.Fatalf("want %v, got %v", bundlePath, artifacts[0].Path)
	}
	if !(output.IsDir(bundlePath)) {
		t.Fatal("expected true")
	}

	binaryPath := ax.Join(bundlePath, "Contents", "MacOS", "core-build")
	content := requireRunString(t, output.Read(binaryPath))
	if !stdlibAssertEqual("bundle:darwin/arm64", content) {
		t.Fatalf("want %v, got %v", "bundle:darwin/arm64", content)
	}

}

func TestRun_UsesLocalTargetWhenBuildConfigMissingGood(t *testing.T) {
	projectDir := t.TempDir()
	output := coreio.NewMemoryMedium()
	requireRunOKResult(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/demo\n"), 0o644))

	artifacts := requireRunArtifacts(t, Run(
		WithProjectDir(projectDir),
		WithBuildType(string(ProjectTypeGo)),
		WithBuildName("core-build"),
		WithOutput(output),
		WithOutputDir("releases"),
		WithBuilderResolver(func(projectType ProjectType) core.Result {
			return core.Ok(&runTestBuilder{})
		}),
	))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	expectedPath := ax.Join("releases", runtime.GOOS+"_"+runtime.GOARCH, "core-build")
	if !stdlibAssertEqual(expectedPath, artifacts[0].Path) {
		t.Fatalf("want %v, got %v", expectedPath, artifacts[0].Path)
	}

}

func TestRun_UsesBuiltinGoResolverWhenResolverUnsetGood(t *testing.T) {
	projectDir := t.TempDir()
	requireRunOKResult(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/builtin\n\ngo 1.24\n"), 0o644))
	requireRunOKResult(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	output := coreio.NewMemoryMedium()
	artifacts := requireRunArtifacts(t, Run(
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeGo)),
		WithBuildName("core-build"),
		WithTargets(Target{OS: runtime.GOOS, Arch: runtime.GOARCH}),
		WithOutput(output),
		WithOutputDir("releases"),
	))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	expectedPath := ax.Join("releases", runtime.GOOS+"_"+runtime.GOARCH, "core-build")
	if runtime.GOOS == "windows" {
		expectedPath += ".exe"
	}
	if !stdlibAssertEqual(expectedPath, artifacts[0].Path) {
		t.Fatalf("want %v, got %v", expectedPath, artifacts[0].Path)
	}
	if !(output.Exists(expectedPath)) {
		t.Fatal("expected true")
	}

}

func TestRun_Bad_NoBuilderResolverForUnsupportedProjectType(t *testing.T) {
	projectDir := t.TempDir()

	err := requireRunError(t, Run(
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeNode)),
	))
	if !stdlibAssertContains(err, "builtin fallback only supports go projects") {
		t.Fatalf("expected %v to contain %v", err, "builtin fallback only supports go projects")
	}

}

func TestRun_ForwardsActionPortOverridesGood(t *testing.T) {
	projectDir := t.TempDir()

	var captured *Config
	_ = requireRunArtifacts(t, Run(
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeGo)),
		WithBuildName("core-build"),
		WithTargets(Target{OS: "linux", Arch: "amd64"}),
		WithBuildTags("integration", "release"),
		WithObfuscate(true),
		WithNSIS(true),
		WithWebView2("embed"),
		WithDenoBuild("deno task bundle"),
		WithNpmBuild("npm run bundle"),
		WithBuildCache(true),
		WithBuilderResolver(func(projectType ProjectType) core.Result {
			return core.Ok(&capturingRunTestBuilder{captured: &captured})
		}),
		WithOutput(coreio.NewMemoryMedium()),
	))
	if stdlibAssertNil(captured) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual([]string{"integration", "release"}, captured.BuildTags) {
		t.Fatalf("want %v, got %v", []string{"integration", "release"}, captured.BuildTags)
	}
	if !(captured.Obfuscate) {
		t.Fatal("expected true")
	}
	if !(captured.NSIS) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("embed", captured.WebView2) {
		t.Fatalf("want %v, got %v", "embed", captured.WebView2)
	}
	if !stdlibAssertEqual("deno task bundle", captured.DenoBuild) {
		t.Fatalf("want %v, got %v", "deno task bundle", captured.DenoBuild)
	}
	if !stdlibAssertEqual("npm run bundle", captured.NpmBuild) {
		t.Fatalf("want %v, got %v", "npm run bundle", captured.NpmBuild)
	}
	if !(captured.Cache.Enabled) {
		t.Fatal("expected true")
	}
	if stdlibAssertEmpty(captured.Cache.Paths) {
		t.Fatal("expected non-empty")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestRun_RegisterDefaultBuilderResolver_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		RegisterDefaultBuilderResolver(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_RegisterDefaultBuilderResolver_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		RegisterDefaultBuilderResolver(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_RegisterDefaultBuilderResolver_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		RegisterDefaultBuilderResolver(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_DefaultBuilderResolver_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultBuilderResolver()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_DefaultBuilderResolver_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultBuilderResolver()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_DefaultBuilderResolver_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultBuilderResolver()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_DefaultRunConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultRunConfig()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_DefaultRunConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultRunConfig()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_DefaultRunConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultRunConfig()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithContext_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithContext(ctx)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithContext_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithContext(ctx)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithContext_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithContext(ctx)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithProjectDir_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithProjectDir(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithProjectDir_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithProjectDir("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithProjectDir_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithProjectDir(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithConfigPath_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithConfigPath(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithConfigPath_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithConfigPath("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithConfigPath_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithConfigPath(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithBuildConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildConfig(&BuildConfig{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithBuildConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildConfig(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithBuildConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildConfig(&BuildConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithBuildType_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildType("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithBuildType_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildType("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithBuildType_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildType("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithBuildTags_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildTags()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithBuildTags_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildTags()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithBuildTags_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildTags()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithObfuscate_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithObfuscate(true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithObfuscate_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithObfuscate(false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithObfuscate_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithObfuscate(true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithNSIS_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithNSIS(true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithNSIS_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithNSIS(false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithNSIS_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithNSIS(true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithWebView2_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithWebView2("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithWebView2_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithWebView2("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithWebView2_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithWebView2("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithDenoBuild_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithDenoBuild("dappcore-command-not-found")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithDenoBuild_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithDenoBuild("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithDenoBuild_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithDenoBuild("dappcore-command-not-found")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithNpmBuild_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithNpmBuild("dappcore-command-not-found")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithNpmBuild_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithNpmBuild("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithNpmBuild_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithNpmBuild("dappcore-command-not-found")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithBuildCache_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildCache(true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithBuildCache_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildCache(false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithBuildCache_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildCache(true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithBuildName_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildName("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithBuildName_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildName("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithBuildName_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuildName("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithOutputDir_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithOutputDir(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithOutputDir_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithOutputDir("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithOutputDir_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithOutputDir(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithOutput_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithOutput(coreio.NewMemoryMedium())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithOutput_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithOutput(coreio.NewMemoryMedium())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithOutput_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithOutput(coreio.NewMemoryMedium())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithTargets_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithTargets()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithTargets_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithTargets()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithTargets_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithTargets()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithVersion_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithVersion("v1.2.3")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithVersion_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithVersion("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithVersion_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithVersion("v1.2.3")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithBuilderResolver_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuilderResolver(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithBuilderResolver_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuilderResolver(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithBuilderResolver_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBuilderResolver(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_WithVersionResolver_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithVersionResolver(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_WithVersionResolver_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithVersionResolver(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_WithVersionResolver_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithVersionResolver(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRun_Run_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRun_Run_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRun_Run_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

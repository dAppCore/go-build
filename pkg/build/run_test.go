package build

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runTestBuilder struct {
	directoryArtifact bool
}

type capturingRunTestBuilder struct {
	captured **Config
}

func (b *runTestBuilder) Name() string { return "run-test" }

func (b *runTestBuilder) Detect(fs coreio.Medium, dir string) (bool, error) {
	return true, nil
}

func (b *runTestBuilder) Build(ctx context.Context, cfg *Config, targets []Target) ([]Artifact, error) {
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
			if err := cfg.FS.EnsureDir(ax.Join(artifactPath, "Contents", "MacOS")); err != nil {
				return nil, err
			}
			if err := cfg.FS.WriteMode(ax.Join(artifactPath, "Contents", "MacOS", cfg.Name), "bundle:"+target.String(), 0o755); err != nil {
				return nil, err
			}
			artifacts = append(artifacts, Artifact{Path: artifactPath, OS: target.OS, Arch: target.Arch})
			continue
		}

		if err := cfg.FS.EnsureDir(ax.Dir(basePath)); err != nil {
			return nil, err
		}
		if err := cfg.FS.WriteMode(basePath, "artifact:"+target.String(), 0o755); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{Path: basePath, OS: target.OS, Arch: target.Arch})
	}

	return artifacts, nil
}

func (b *capturingRunTestBuilder) Name() string { return "capturing-run-test" }

func (b *capturingRunTestBuilder) Detect(fs coreio.Medium, dir string) (bool, error) {
	return true, nil
}

func (b *capturingRunTestBuilder) Build(ctx context.Context, cfg *Config, targets []Target) ([]Artifact, error) {
	if b.captured != nil {
		*b.captured = cfg
	}
	return (&runTestBuilder{}).Build(ctx, cfg, targets)
}

func TestRun_UsesOutputMedium_Good(t *testing.T) {
	projectDir := t.TempDir()
	output := coreio.NewMemoryMedium()

	artifacts, err := Run(
		WithContext(context.Background()),
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeGo)),
		WithBuildName("core-build"),
		WithTargets(Target{OS: "linux", Arch: "amd64"}),
		WithOutput(output),
		WithOutputDir("releases"),
		WithBuilderResolver(func(projectType ProjectType) (Builder, error) {
			return &runTestBuilder{}, nil
		}),
	)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	assert.Equal(t, ax.Join("releases", "linux_amd64", "core-build"), artifacts[0].Path)

	content, err := output.Read(ax.Join("releases", "linux_amd64", "core-build"))
	require.NoError(t, err)
	assert.Equal(t, "artifact:linux/amd64", content)
}

func TestRun_UsesOutputMediumRootWhenOutputDirUnset_Good(t *testing.T) {
	projectDir := t.TempDir()
	output := coreio.NewMemoryMedium()

	artifacts, err := Run(
		WithContext(context.Background()),
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeGo)),
		WithBuildName("core-build"),
		WithTargets(Target{OS: "linux", Arch: "amd64"}),
		WithOutput(output),
		WithBuilderResolver(func(projectType ProjectType) (Builder, error) {
			return &runTestBuilder{}, nil
		}),
	)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	assert.Equal(t, ax.Join("linux_amd64", "core-build"), artifacts[0].Path)

	content, err := output.Read(ax.Join("linux_amd64", "core-build"))
	require.NoError(t, err)
	assert.Equal(t, "artifact:linux/amd64", content)
}

func TestRun_MirrorsDirectoryArtifacts_Good(t *testing.T) {
	projectDir := t.TempDir()
	output := coreio.NewMemoryMedium()

	artifacts, err := Run(
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeWails)),
		WithBuildName("core-build"),
		WithTargets(Target{OS: "darwin", Arch: "arm64"}),
		WithOutput(output),
		WithOutputDir("bundles"),
		WithBuilderResolver(func(projectType ProjectType) (Builder, error) {
			return &runTestBuilder{directoryArtifact: true}, nil
		}),
	)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	bundlePath := ax.Join("bundles", "darwin_arm64", "core-build.app")
	assert.Equal(t, bundlePath, artifacts[0].Path)
	assert.True(t, output.IsDir(bundlePath))

	binaryPath := ax.Join(bundlePath, "Contents", "MacOS", "core-build")
	content, err := output.Read(binaryPath)
	require.NoError(t, err)
	assert.Equal(t, "bundle:darwin/arm64", content)
}

func TestRun_UsesLocalTargetWhenBuildConfigMissing_Good(t *testing.T) {
	projectDir := t.TempDir()
	output := coreio.NewMemoryMedium()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/demo\n"), 0o644))

	artifacts, err := Run(
		WithProjectDir(projectDir),
		WithBuildType(string(ProjectTypeGo)),
		WithBuildName("core-build"),
		WithOutput(output),
		WithOutputDir("releases"),
		WithBuilderResolver(func(projectType ProjectType) (Builder, error) {
			return &runTestBuilder{}, nil
		}),
	)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	expectedPath := ax.Join("releases", runtime.GOOS+"_"+runtime.GOARCH, "core-build")
	assert.Equal(t, expectedPath, artifacts[0].Path)
}

func TestRun_UsesBuiltinGoResolverWhenResolverUnset_Good(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/builtin\n\ngo 1.24\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	output := coreio.NewMemoryMedium()
	artifacts, err := Run(
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeGo)),
		WithBuildName("core-build"),
		WithTargets(Target{OS: runtime.GOOS, Arch: runtime.GOARCH}),
		WithOutput(output),
		WithOutputDir("releases"),
	)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	expectedPath := ax.Join("releases", runtime.GOOS+"_"+runtime.GOARCH, "core-build")
	if runtime.GOOS == "windows" {
		expectedPath += ".exe"
	}
	assert.Equal(t, expectedPath, artifacts[0].Path)
	assert.True(t, output.Exists(expectedPath))
}

func TestRun_Bad_NoBuilderResolverForUnsupportedProjectType(t *testing.T) {
	projectDir := t.TempDir()

	_, err := Run(
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeNode)),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "builtin fallback only supports go projects")
}

func TestRun_ForwardsActionPortOverrides_Good(t *testing.T) {
	projectDir := t.TempDir()

	var captured *Config
	_, err := Run(
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
		WithBuildCache(true),
		WithBuilderResolver(func(projectType ProjectType) (Builder, error) {
			return &capturingRunTestBuilder{captured: &captured}, nil
		}),
		WithOutput(coreio.NewMemoryMedium()),
	)
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, []string{"integration", "release"}, captured.BuildTags)
	assert.True(t, captured.Obfuscate)
	assert.True(t, captured.NSIS)
	assert.Equal(t, "embed", captured.WebView2)
	assert.Equal(t, "deno task bundle", captured.DenoBuild)
	assert.True(t, captured.Cache.Enabled)
	assert.NotEmpty(t, captured.Cache.Paths)
}

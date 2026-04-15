package build

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runTestBuilder struct {
	directoryArtifact bool
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

func TestRun_Bad_NoBuilderResolver(t *testing.T) {
	projectDir := t.TempDir()

	_, err := Run(
		WithProjectDir(projectDir),
		WithBuildConfig(DefaultConfig()),
		WithBuildType(string(ProjectTypeGo)),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "builder resolver is required")
}

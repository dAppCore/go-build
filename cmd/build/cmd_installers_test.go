package buildcmd

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/core"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCmd_AddInstallersCommand_Good(t *testing.T) {
	c := core.New()

	AddInstallersCommand(c)

	assert.True(t, c.Command("build/installers").OK)
}

func TestBuildCmd_runBuildInstallersInDir_GeneratesAll_Good(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, io.Local.EnsureDir(ax.Join(projectDir, ".core")))
	require.NoError(t, io.Local.Write(ax.Join(projectDir, ".core", "build.yaml"), `version: 1
project:
  binary: corex
`))
	require.NoError(t, io.Local.Write(ax.Join(projectDir, ".core", "release.yaml"), `version: 1
project:
  repository: dappcore/core
`))

	err := runBuildInstallersInDir(context.Background(), projectDir, "", "v1.2.3", "", "", "")
	require.NoError(t, err)

	outputDir := ax.Join(projectDir, "dist", "installers")
	expected := []string{"setup.sh", "ci.sh", "php.sh", "go.sh", "agent.sh", "dev.sh"}
	for _, name := range expected {
		assert.FileExists(t, ax.Join(outputDir, name))
	}

	content, err := io.Local.Read(ax.Join(outputDir, "setup.sh"))
	require.NoError(t, err)
	assert.Contains(t, content, "corex")
	assert.Contains(t, content, "v1.2.3")
	assert.Contains(t, content, "dappcore/core")
	assert.Contains(t, content, "https://lthn.sh/setup.sh")

	devContent, err := io.Local.Read(ax.Join(outputDir, "dev.sh"))
	require.NoError(t, err)
	assert.Contains(t, devContent, `DEV_IMAGE_VERSION="${VERSION#v}"`)
	assert.Contains(t, devContent, `DEV_IMAGE="ghcr.io/dappcore/core-dev:${DEV_IMAGE_VERSION}"`)
}

func TestBuildCmd_runBuildInstallersInDir_GeneratesSingleVariant_Good(t *testing.T) {
	projectDir := t.TempDir()

	err := runBuildInstallersInDir(context.Background(), projectDir, "ci", "v1.2.3", "out/installers", "dappcore/core", "core")
	require.NoError(t, err)

	assert.FileExists(t, ax.Join(projectDir, "out", "installers", "ci.sh"))
	assert.NoFileExists(t, ax.Join(projectDir, "out", "installers", "setup.sh"))
}

func TestBuildCmd_runBuildInstallersInDir_UsesResolvedVersion_Good(t *testing.T) {
	projectDir := t.TempDir()

	originalVersionResolver := resolveInstallersVersion
	t.Cleanup(func() {
		resolveInstallersVersion = originalVersionResolver
	})
	resolveInstallersVersion = func(ctx context.Context, dir string) (string, error) {
		assert.Equal(t, projectDir, dir)
		return "v9.9.9", nil
	}

	err := runBuildInstallersInDir(context.Background(), projectDir, "setup.sh", "", "", "dappcore/core", "core")
	require.NoError(t, err)

	content, err := io.Local.Read(ax.Join(projectDir, "dist", "installers", "setup.sh"))
	require.NoError(t, err)
	assert.Contains(t, content, "v9.9.9")
}

func TestBuildCmd_runBuildInstallersInDir_UsesGitRemoteWhenReleaseConfigMissing_Good(t *testing.T) {
	projectDir := t.TempDir()

	originalLoadReleaseConfig := loadInstallersReleaseConfig
	originalDetectRepository := detectInstallersRepository
	t.Cleanup(func() {
		loadInstallersReleaseConfig = originalLoadReleaseConfig
		detectInstallersRepository = originalDetectRepository
	})

	loadInstallersReleaseConfig = func(dir string) (*release.Config, error) {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		return cfg, nil
	}
	detectInstallersRepository = func(ctx context.Context, dir string) (string, error) {
		assert.Equal(t, projectDir, dir)
		return "host-uk/core-build", nil
	}

	err := runBuildInstallersInDir(context.Background(), projectDir, "agentic", "v1.2.3", "", "", "core")
	require.NoError(t, err)

	content, err := io.Local.Read(ax.Join(projectDir, "dist", "installers", "agent.sh"))
	require.NoError(t, err)
	assert.Contains(t, content, "host-uk/core-build")
}

func TestBuildCmd_runBuildInstallersInDir_UnknownVariant_Bad(t *testing.T) {
	projectDir := t.TempDir()

	err := runBuildInstallersInDir(context.Background(), projectDir, "bogus", "v1.2.3", "", "dappcore/core", "core")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown installer variant")
}

func TestBuildCmd_runBuildInstallersInDir_MissingRepository_Bad(t *testing.T) {
	projectDir := t.TempDir()

	originalLoadReleaseConfig := loadInstallersReleaseConfig
	originalDetectRepository := detectInstallersRepository
	t.Cleanup(func() {
		loadInstallersReleaseConfig = originalLoadReleaseConfig
		detectInstallersRepository = originalDetectRepository
	})

	loadInstallersReleaseConfig = func(dir string) (*release.Config, error) {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		return cfg, nil
	}
	detectInstallersRepository = func(ctx context.Context, dir string) (string, error) {
		return "", assert.AnError
	}

	err := runBuildInstallersInDir(context.Background(), projectDir, "ci", "v1.2.3", "", "", "core")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use --repo")
}

func TestBuild_GenerateInstallerWrappers_Good(t *testing.T) {
	script, err := build.GenerateInstaller(build.VariantCI, build.InstallerConfig{
		Version:    "v1.2.3",
		Repo:       "dappcore/core",
		BinaryName: "core",
	})
	require.NoError(t, err)
	assert.Contains(t, script, "dappcore/core")
	assert.Equal(t, []build.InstallerVariant{
		build.VariantFull,
		build.VariantCI,
		build.VariantPHP,
		build.VariantGo,
		build.VariantAgent,
		build.VariantDev,
	}, build.InstallerVariants())
	assert.Equal(t, "ci.sh", build.InstallerOutputName(build.VariantCI))
	assert.Equal(t, build.VariantAgent, build.VariantAgentic)
	agenticScript, err := build.GenerateInstaller(build.VariantAgentic, build.InstallerConfig{
		Version:    "v1.2.3",
		Repo:       "dappcore/core",
		BinaryName: "core",
	})
	require.NoError(t, err)
	assert.Contains(t, agenticScript, "dappcore/core")
}

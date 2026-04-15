package buildcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	require.NoError(t, ax.ExecDir(context.Background(), dir, "git", args...))
}

func setupFakeGPG(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

output=""
while [ $# -gt 0 ]; do
	case "$1" in
	--output)
		shift
		output="${1:-}"
		;;
	esac
	shift
done

: "${output:?missing --output}"
mkdir -p "$(dirname "$output")"
printf 'signature\n' > "$output"
`

	require.NoError(t, ax.WriteFile(ax.Join(binDir, "gpg"), []byte(script), 0o755))
}

func TestBuildCmd_GetBuilder_Good(t *testing.T) {
	t.Run("returns Python builder for python project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypePython)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "python", builder.Name())
	})
}

func TestBuildCmd_buildRuntimeConfig_Good(t *testing.T) {
	buildConfig := &build.BuildConfig{
		Project: build.Project{
			Name: "sample",
		},
		Build: build.Build{
			LDFlags:        []string{"-s", "-w"},
			Flags:          []string{"-trimpath"},
			BuildTags:      []string{"integration"},
			Env:            []string{"FOO=bar"},
			CGO:            true,
			Obfuscate:      true,
			DenoBuild:      "deno task bundle",
			NSIS:           true,
			WebView2:       "embed",
			Dockerfile:     "Dockerfile.custom",
			Registry:       "ghcr.io",
			Image:          "owner/repo",
			Tags:           []string{"latest", "{{.Version}}"},
			BuildArgs:      map[string]string{"VERSION": "{{.Version}}"},
			Push:           true,
			Load:           true,
			LinuxKitConfig: ".core/linuxkit/server.yml",
			Formats:        []string{"iso", "qcow2"},
		},
	}

	cfg := buildRuntimeConfig(io.Local, "/project", "/project/dist", "binary", buildConfig, false, "", "v1.2.3")

	assert.Equal(t, []string{"-s", "-w"}, cfg.LDFlags)
	assert.Equal(t, []string{"-trimpath"}, cfg.Flags)
	assert.Equal(t, []string{"integration"}, cfg.BuildTags)
	assert.Equal(t, []string{"FOO=bar"}, cfg.Env)
	assert.True(t, cfg.CGO)
	assert.True(t, cfg.Obfuscate)
	assert.Equal(t, "deno task bundle", cfg.DenoBuild)
	assert.True(t, cfg.NSIS)
	assert.Equal(t, "embed", cfg.WebView2)
	assert.Equal(t, "Dockerfile.custom", cfg.Dockerfile)
	assert.Equal(t, "ghcr.io", cfg.Registry)
	assert.Equal(t, "owner/repo", cfg.Image)
	assert.Equal(t, []string{"latest", "{{.Version}}"}, cfg.Tags)
	assert.Equal(t, map[string]string{"VERSION": "{{.Version}}"}, cfg.BuildArgs)
	assert.True(t, cfg.Push)
	assert.True(t, cfg.Load)
	assert.Equal(t, ".core/linuxkit/server.yml", cfg.LinuxKitConfig)
	assert.Equal(t, []string{"iso", "qcow2"}, cfg.Formats)
	assert.Equal(t, "v1.2.3", cfg.Version)
}

func TestBuildCmd_buildRuntimeConfig_ImageOverride_Good(t *testing.T) {
	buildConfig := &build.BuildConfig{
		Build: build.Build{
			Image: "owner/repo",
		},
	}

	cfg := buildRuntimeConfig(io.Local, "/project", "/project/dist", "binary", buildConfig, true, "cli/image", "v2.0.0")

	assert.Equal(t, "cli/image", cfg.Image)
	assert.True(t, cfg.Push)
	assert.Equal(t, "v2.0.0", cfg.Version)
}

func TestBuildCmd_buildRuntimeConfig_ClonesBuildArgs_Good(t *testing.T) {
	buildConfig := &build.BuildConfig{
		Build: build.Build{
			BuildArgs: map[string]string{"VERSION": "v1.2.3"},
		},
	}

	cfg := buildRuntimeConfig(io.Local, "/project", "/project/dist", "binary", buildConfig, false, "", "v1.2.3")
	require.NotNil(t, cfg.BuildArgs)

	cfg.BuildArgs["VERSION"] = "mutated"
	assert.Equal(t, "v1.2.3", buildConfig.Build.BuildArgs["VERSION"])
}

func TestBuildCmd_resolveNoSign_Good(t *testing.T) {
	t.Run("keeps signing enabled by default", func(t *testing.T) {
		assert.False(t, resolveNoSign(false, true, false))
	})

	t.Run("disables signing when no-sign is set", func(t *testing.T) {
		assert.True(t, resolveNoSign(true, true, false))
	})

	t.Run("disables signing when sign=false is set", func(t *testing.T) {
		assert.True(t, resolveNoSign(false, false, true))
	})

	t.Run("keeps signing enabled when sign=true is set", func(t *testing.T) {
		assert.False(t, resolveNoSign(false, true, true))
	})
}

func TestBuildCmd_resolveBuildSignConfig_Good(t *testing.T) {
	t.Run("enables signing when notarize overrides disabled config", func(t *testing.T) {
		signCfg := resolveBuildSignConfig(build.DefaultConfig().Sign, ProjectBuildRequest{
			Notarize: true,
		})

		assert.True(t, signCfg.Enabled)
		assert.True(t, signCfg.MacOS.Notarize)
	})

	t.Run("preserves explicit no-sign over notarize", func(t *testing.T) {
		signCfg := resolveBuildSignConfig(build.DefaultConfig().Sign, ProjectBuildRequest{
			NoSign:   true,
			Notarize: true,
		})

		assert.False(t, signCfg.Enabled)
		assert.True(t, signCfg.MacOS.Notarize)
	})

	t.Run("re-enables signing when config disabled but notarize requested", func(t *testing.T) {
		base := build.DefaultConfig().Sign
		base.Enabled = false

		signCfg := resolveBuildSignConfig(base, ProjectBuildRequest{
			Notarize: true,
		})

		assert.True(t, signCfg.Enabled)
		assert.True(t, signCfg.MacOS.Notarize)
	})
}

func TestBuildCmd_resolvePackageOutputs_Good(t *testing.T) {
	t.Run("leaves archive and checksum defaults alone when package is unset", func(t *testing.T) {
		archiveOutput, checksumOutput := resolvePackageOutputs(false, false, false, false, false, false)
		assert.False(t, archiveOutput)
		assert.False(t, checksumOutput)
	})

	t.Run("disables archive and checksum when package=false and neither output flag is explicit", func(t *testing.T) {
		archiveOutput, checksumOutput := resolvePackageOutputs(false, true, true, false, true, false)
		assert.False(t, archiveOutput)
		assert.False(t, checksumOutput)
	})

	t.Run("enables archive and checksum when package=true and neither output flag is explicit", func(t *testing.T) {
		archiveOutput, checksumOutput := resolvePackageOutputs(true, true, false, false, false, false)
		assert.True(t, archiveOutput)
		assert.True(t, checksumOutput)
	})

	t.Run("preserves explicit archive and checksum overrides over package=false", func(t *testing.T) {
		archiveOutput, checksumOutput := resolvePackageOutputs(false, true, true, true, false, true)
		assert.True(t, archiveOutput)
		assert.False(t, checksumOutput)
	})
}

func TestBuildCmd_runProjectBuild_CIModeEmitsGitHubAnnotationOnError_Bad(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/demo\n"), 0o644))

	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
		cli.SetStdout(nil)
		cli.SetStderr(nil)
	})
	getProjectBuildWorkingDir = func() (string, error) { return projectDir, nil }

	var stdout bytes.Buffer
	cli.SetStdout(&stdout)
	cli.SetStderr(&stdout)

	err := runProjectBuild(ProjectBuildRequest{
		Context:     context.Background(),
		CIMode:      true,
		TargetsFlag: "linux",
	})
	require.Error(t, err)
	assert.Contains(t, stdout.String(), emitCIAnnotationForTest(err))
}

func TestBuildCmd_applyProjectBuildOverrides_Good(t *testing.T) {
	t.Run("applies action-style build overrides and enables default cache", func(t *testing.T) {
		cfg := build.DefaultConfig()

		applyProjectBuildOverrides(cfg, ProjectBuildRequest{
			BuildTagsFlag: "mlx, debug release,mlx",
			Obfuscate:     true,
			ObfuscateSet:  true,
			NSIS:          true,
			NSISSet:       true,
			WebView2:      "download",
			WebView2Set:   true,
			DenoBuild:     "deno task bundle",
			DenoBuildSet:  true,
			BuildCache:    true,
			BuildCacheSet: true,
			Sign:          false,
			SignSet:       true,
		})

		assert.Equal(t, []string{"mlx", "debug", "release"}, cfg.Build.BuildTags)
		assert.True(t, cfg.Build.Obfuscate)
		assert.True(t, cfg.Build.NSIS)
		assert.Equal(t, "download", cfg.Build.WebView2)
		assert.Equal(t, "deno task bundle", cfg.Build.DenoBuild)
		assert.True(t, cfg.Build.Cache.Enabled)
		assert.Equal(t, ax.Join(build.ConfigDir, "cache"), cfg.Build.Cache.Directory)
		assert.Equal(t, []string{ax.Join("cache", "go-build"), ax.Join("cache", "go-mod")}, cfg.Build.Cache.Paths)
		assert.False(t, cfg.Sign.Enabled)
	})

	t.Run("preserves configured cache paths when enabling cache from the CLI", func(t *testing.T) {
		cfg := build.DefaultConfig()
		cfg.Build.Cache = build.CacheConfig{
			Directory: "custom/cache",
			Paths:     []string{"custom/go-build"},
		}

		applyProjectBuildOverrides(cfg, ProjectBuildRequest{
			BuildCache:    true,
			BuildCacheSet: true,
		})

		assert.True(t, cfg.Build.Cache.Enabled)
		assert.Equal(t, "custom/cache", cfg.Build.Cache.Directory)
		assert.Equal(t, []string{"custom/go-build"}, cfg.Build.Cache.Paths)
	})

	t.Run("can disable build cache without discarding the configured paths", func(t *testing.T) {
		cfg := build.DefaultConfig()
		cfg.Build.Cache = build.CacheConfig{
			Enabled:   true,
			Directory: "custom/cache",
			Paths:     []string{"custom/go-build", "custom/go-mod"},
		}

		applyProjectBuildOverrides(cfg, ProjectBuildRequest{
			BuildCache:    false,
			BuildCacheSet: true,
		})

		assert.False(t, cfg.Build.Cache.Enabled)
		assert.Equal(t, "custom/cache", cfg.Build.Cache.Directory)
		assert.Equal(t, []string{"custom/go-build", "custom/go-mod"}, cfg.Build.Cache.Paths)
	})

	t.Run("can force signing back on when config disabled it", func(t *testing.T) {
		cfg := build.DefaultConfig()
		cfg.Sign.Enabled = false

		applyProjectBuildOverrides(cfg, ProjectBuildRequest{
			Sign:    true,
			SignSet: true,
		})

		assert.True(t, cfg.Sign.Enabled)
	})
}

func TestBuildCmd_resolveProjectBuildName_Good(t *testing.T) {
	t.Run("prefers the CLI build name override", func(t *testing.T) {
		cfg := &build.BuildConfig{
			Project: build.Project{
				Name:   "project-name",
				Binary: "project-binary",
			},
		}

		assert.Equal(t, "cli-name", resolveProjectBuildName("/tmp/project", cfg, "cli-name"))
	})

	t.Run("falls back to project binary, then project name, then directory name", func(t *testing.T) {
		cfg := &build.BuildConfig{
			Project: build.Project{
				Name:   "project-name",
				Binary: "project-binary",
			},
		}
		assert.Equal(t, "project-binary", resolveProjectBuildName("/tmp/project", cfg, ""))

		cfg.Project.Binary = ""
		assert.Equal(t, "project-name", resolveProjectBuildName("/tmp/project", cfg, ""))

		cfg.Project.Name = ""
		assert.Equal(t, "project", resolveProjectBuildName("/tmp/project", cfg, ""))
	})
}

func TestBuildCmd_resolveArchiveFormat_Good(t *testing.T) {
	t.Run("uses cli override when present", func(t *testing.T) {
		format, err := resolveArchiveFormat("gz", "xz")
		require.NoError(t, err)
		assert.Equal(t, build.ArchiveFormatXZ, format)
	})

	t.Run("falls back to config when cli override is empty", func(t *testing.T) {
		format, err := resolveArchiveFormat("zip", "")
		require.NoError(t, err)
		assert.Equal(t, build.ArchiveFormatZip, format)
	})
}

func TestBuildCmd_resolveBuildVersion_Good(t *testing.T) {
	dir := t.TempDir()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	require.NoError(t, ax.WriteFile(ax.Join(dir, "README.md"), []byte("hello\n"), 0644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feat: initial commit")
	runGit(t, dir, "tag", "v1.4.2")

	version, err := resolveBuildVersion(context.Background(), dir)
	require.NoError(t, err)
	assert.Equal(t, "v1.4.2", version)
}

func TestBuildCmd_writeArtifactMetadata_Good(t *testing.T) {
	t.Setenv("GITHUB_SHA", "abc1234def5678")
	t.Setenv("GITHUB_REF", "refs/tags/v1.2.3")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")

	fs := io.Local
	dir := t.TempDir()

	linuxDir := ax.Join(dir, "linux_amd64")
	windowsDir := ax.Join(dir, "windows_amd64")
	require.NoError(t, ax.MkdirAll(linuxDir, 0755))
	require.NoError(t, ax.MkdirAll(windowsDir, 0755))

	artifacts := []build.Artifact{
		{Path: ax.Join(linuxDir, "sample"), OS: "linux", Arch: "amd64"},
		{Path: ax.Join(windowsDir, "sample.exe"), OS: "windows", Arch: "amd64"},
	}

	err := writeArtifactMetadata(fs, "sample", artifacts)
	require.NoError(t, err)

	verifyArtifactMeta := func(path string, expectedOS string, expectedArch string) {
		content, readErr := ax.ReadFile(path)
		require.NoError(t, readErr)

		var meta map[string]any
		require.NoError(t, json.Unmarshal(content, &meta))

		assert.Equal(t, "sample", meta["name"])
		assert.Equal(t, expectedOS, meta["os"])
		assert.Equal(t, expectedArch, meta["arch"])
		assert.Equal(t, "v1.2.3", meta["tag"])
		assert.Equal(t, "owner/repo", meta["repo"])
	}

	verifyArtifactMeta(ax.Join(linuxDir, "artifact_meta.json"), "linux", "amd64")
	verifyArtifactMeta(ax.Join(windowsDir, "artifact_meta.json"), "windows", "amd64")
}

func TestBuildCmd_writeArtifactMetadata_SkipsChecksumArtifacts_Good(t *testing.T) {
	t.Setenv("GITHUB_SHA", "abc1234def5678")
	t.Setenv("GITHUB_REF", "refs/tags/v1.2.3")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")

	fs := io.Local
	dir := t.TempDir()
	distDir := ax.Join(dir, "dist")
	require.NoError(t, ax.MkdirAll(distDir, 0o755))

	checksumPath := ax.Join(distDir, "CHECKSUMS.txt")
	signaturePath := checksumPath + ".asc"
	require.NoError(t, ax.WriteFile(checksumPath, []byte("checksums"), 0o644))
	require.NoError(t, ax.WriteFile(signaturePath, []byte("signature"), 0o644))

	err := writeArtifactMetadata(fs, "sample", []build.Artifact{
		{Path: checksumPath},
		{Path: signaturePath},
	})
	require.NoError(t, err)

	assert.NoFileExists(t, ax.Join(distDir, "artifact_meta.json"))
}

func TestBuildCmd_computeAndWriteChecksums_IncludesChecksumArtifacts_Good(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist")
	artifactPath := ax.Join(outputDir, "sample_linux_amd64.tar.gz")
	require.NoError(t, ax.MkdirAll(outputDir, 0o755))
	require.NoError(t, ax.WriteFile(artifactPath, []byte("archive"), 0o644))

	signCfg := build.DefaultConfig().Sign
	signCfg.Enabled = false

	artifacts, err := computeAndWriteChecksums(
		context.Background(),
		io.Local,
		projectDir,
		outputDir,
		[]build.Artifact{{Path: artifactPath, OS: "linux", Arch: "amd64"}},
		signCfg,
		false,
		false,
	)
	require.NoError(t, err)

	paths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		paths = append(paths, artifact.Path)
	}

	assert.Contains(t, paths, artifactPath)
	assert.Contains(t, paths, ax.Join(outputDir, "CHECKSUMS.txt"))
	assert.NotContains(t, paths, ax.Join(outputDir, "CHECKSUMS.txt.asc"))
	assert.FileExists(t, ax.Join(outputDir, "CHECKSUMS.txt"))
}

func TestBuildCmd_computeAndWriteChecksums_IncludesSignatureArtifact_Good(t *testing.T) {
	binDir := t.TempDir()
	setupFakeGPG(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist")
	artifactPath := ax.Join(outputDir, "sample_linux_amd64.tar.gz")
	require.NoError(t, ax.MkdirAll(outputDir, 0o755))
	require.NoError(t, ax.WriteFile(artifactPath, []byte("archive"), 0o644))

	signCfg := build.DefaultConfig().Sign
	signCfg.Enabled = true
	signCfg.GPG.Key = "ABCD1234"

	artifacts, err := computeAndWriteChecksums(
		context.Background(),
		io.Local,
		projectDir,
		outputDir,
		[]build.Artifact{{Path: artifactPath, OS: "linux", Arch: "amd64"}},
		signCfg,
		false,
		false,
	)
	require.NoError(t, err)

	paths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		paths = append(paths, artifact.Path)
	}

	assert.Contains(t, paths, ax.Join(outputDir, "CHECKSUMS.txt"))
	assert.Contains(t, paths, ax.Join(outputDir, "CHECKSUMS.txt.asc"))
	assert.FileExists(t, ax.Join(outputDir, "CHECKSUMS.txt.asc"))
}

func TestBuildCmd_selectOutputArtifacts_Good(t *testing.T) {
	rawArtifacts := []build.Artifact{{Path: "dist/raw"}}
	archivedArtifacts := []build.Artifact{{Path: "dist/raw.tar.gz"}}
	checksummedArtifacts := []build.Artifact{{Path: "dist/raw.tar.gz", Checksum: "abc123"}}

	t.Run("prefers checksummed artifacts", func(t *testing.T) {
		selected := selectOutputArtifacts(rawArtifacts, archivedArtifacts, checksummedArtifacts)
		assert.Equal(t, checksummedArtifacts, selected)
	})

	t.Run("falls back to archived artifacts", func(t *testing.T) {
		selected := selectOutputArtifacts(rawArtifacts, archivedArtifacts, nil)
		assert.Equal(t, archivedArtifacts, selected)
	})

	t.Run("falls back to raw artifacts", func(t *testing.T) {
		selected := selectOutputArtifacts(rawArtifacts, nil, nil)
		assert.Equal(t, rawArtifacts, selected)
	})
}

func TestBuildCmd_runProjectBuild_PwaOverride_Good(t *testing.T) {
	expectedWD, err := ax.Getwd()
	require.NoError(t, err)

	original := runLocalPwaBuild
	t.Cleanup(func() {
		runLocalPwaBuild = original
	})

	called := false
	runLocalPwaBuild = func(ctx context.Context, projectDir string) error {
		called = true
		assert.Equal(t, expectedWD, projectDir)
		return nil
	}

	err = runProjectBuild(ProjectBuildRequest{
		Context:   context.Background(),
		BuildType: "pwa",
	})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestBuildCmd_runProjectBuild_NoConfigGoPassthrough_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() (string, error) {
		return projectDir, nil
	}

	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	err := runProjectBuild(ProjectBuildRequest{
		Context:       context.Background(),
		ArchiveOutput: true,
	})
	require.NoError(t, err)

	assert.FileExists(t, ax.Join(projectDir, "passthrough"))
	assert.NoFileExists(t, ax.Join(projectDir, "dist"))
}

func TestBuildCmd_runProjectBuild_ConfiguredBuildDefaultsToRawArtifacts_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() (string, error) {
		return projectDir, nil
	}

	require.NoError(t, ax.MkdirAll(ax.Join(projectDir, ".core"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/configured\n\ngo 1.24\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(
		"version: 1\n"+
			"project:\n"+
			"  name: configured\n"+
			"  binary: configured\n"+
			"targets:\n"+
			"  - os: "+runtime.GOOS+"\n"+
			"    arch: "+runtime.GOARCH+"\n"+
			"sign:\n"+
			"  enabled: false\n",
	), 0o644))

	err := runProjectBuild(ProjectBuildRequest{
		Context: context.Background(),
	})
	require.NoError(t, err)

	expectedBinary := ax.Join(projectDir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "configured")
	if runtime.GOOS == "windows" {
		expectedBinary += ".exe"
	}

	assert.FileExists(t, expectedBinary)
	assert.NoFileExists(t, ax.Join(projectDir, "dist", "CHECKSUMS.txt"))
	assert.NoFileExists(t, ax.Join(projectDir, "dist", "configured_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.gz"))
	assert.NoFileExists(t, ax.Join(projectDir, "dist", "configured_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.xz"))
	assert.NoFileExists(t, ax.Join(projectDir, "dist", "configured_"+runtime.GOOS+"_"+runtime.GOARCH+".zip"))
}

func TestBuildCmd_shouldUseGoBuildPassthrough_Good(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	t.Run("keeps simple no-config go builds on passthrough", func(t *testing.T) {
		assert.True(t, shouldUseGoBuildPassthrough(io.Local, projectDir, ProjectBuildRequest{}))
	})

	t.Run("uses the pipeline for ci mode", func(t *testing.T) {
		assert.False(t, shouldUseGoBuildPassthrough(io.Local, projectDir, ProjectBuildRequest{
			CIMode: true,
		}))
	})

	t.Run("uses the pipeline for explicit archive requests", func(t *testing.T) {
		assert.False(t, shouldUseGoBuildPassthrough(io.Local, projectDir, ProjectBuildRequest{
			ArchiveOutput:    true,
			ArchiveOutputSet: true,
		}))
	})

	t.Run("uses the pipeline for explicit package requests", func(t *testing.T) {
		assert.False(t, shouldUseGoBuildPassthrough(io.Local, projectDir, ProjectBuildRequest{
			ArchiveOutput:  true,
			ChecksumOutput: true,
			PackageSet:     true,
		}))
	})

	t.Run("uses the pipeline for explicit versioning", func(t *testing.T) {
		assert.False(t, shouldUseGoBuildPassthrough(io.Local, projectDir, ProjectBuildRequest{
			Version: "v1.2.3",
		}))
	})

	t.Run("uses the pipeline for Wails projects even without config", func(t *testing.T) {
		wailsDir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(wailsDir, "go.mod"), []byte("module example.com/wails\n\ngo 1.24\n"), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(wailsDir, "wails.json"), []byte(`{"name":"demo"}`), 0o644))

		assert.False(t, shouldUseGoBuildPassthrough(io.Local, wailsDir, ProjectBuildRequest{}))
	})

	t.Run("uses the pipeline for multi-type Go and Node projects", func(t *testing.T) {
		stackDir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(stackDir, "go.mod"), []byte("module example.com/fullstack\n\ngo 1.24\n"), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(stackDir, "package.json"), []byte(`{"name":"fullstack"}`), 0o644))

		assert.False(t, shouldUseGoBuildPassthrough(io.Local, stackDir, ProjectBuildRequest{}))
	})
}

func TestBuildCmd_runProjectBuild_NoConfigGoPassthroughTargetAndOutput_Good(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "bin")
	outputPath := ax.Join(outputDir, "custom-binary")
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() (string, error) {
		return projectDir, nil
	}

	require.NoError(t, ax.MkdirAll(outputDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	err := runProjectBuild(ProjectBuildRequest{
		Context:     context.Background(),
		TargetsFlag: "linux/amd64",
		OutputDir:   outputDir,
		BuildName:   "custom-binary",
	})
	require.NoError(t, err)

	assert.FileExists(t, outputPath)
}

func TestBuildCmd_runProjectBuild_NoConfigGoCIModeUsesPipeline_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() (string, error) {
		return projectDir, nil
	}

	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	buildName := ax.Base(projectDir)

	err := runProjectBuild(ProjectBuildRequest{
		Context:        context.Background(),
		CIMode:         true,
		TargetsFlag:    "linux/amd64",
		ArchiveOutput:  false,
		ChecksumOutput: false,
	})
	require.NoError(t, err)

	assert.NoFileExists(t, ax.Join(projectDir, "passthrough"))
	assert.FileExists(t, ax.Join(projectDir, "dist", "linux_amd64", buildName))
}

func TestBuildCmd_runProjectBuild_CIModeCopiesCIStampedArtifacts_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() (string, error) {
		return projectDir, nil
	}

	t.Setenv("GITHUB_SHA", "abc1234def5678901234567890123456789012345")
	t.Setenv("GITHUB_REF", "refs/tags/v1.2.3")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")

	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	err := runProjectBuild(ProjectBuildRequest{
		Context:     context.Background(),
		CIMode:      true,
		TargetsFlag: "linux/amd64",
	})
	require.NoError(t, err)

	ciArtifactPath := ax.Join(projectDir, "dist", "linux_amd64", ax.Base(projectDir)+"_linux_amd64_v1.2.3")
	assert.FileExists(t, ciArtifactPath)
	assert.FileExists(t, ax.Join(projectDir, "dist", "linux_amd64", ax.Base(projectDir)))
	assert.FileExists(t, ax.Join(projectDir, "dist", "linux_amd64", "artifact_meta.json"))
}

func TestBuildCmd_runProjectBuild_NoConfigGoArchiveRequestUsesPipeline_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() (string, error) {
		return projectDir, nil
	}

	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	buildName := ax.Base(projectDir)

	err := runProjectBuild(ProjectBuildRequest{
		Context:          context.Background(),
		TargetsFlag:      "linux/amd64",
		ArchiveOutput:    true,
		ArchiveOutputSet: true,
		ChecksumOutput:   false,
	})
	require.NoError(t, err)

	assert.NoFileExists(t, ax.Join(projectDir, "passthrough"))
	assert.FileExists(t, ax.Join(projectDir, "dist", "linux_amd64", buildName))
	assert.FileExists(t, ax.Join(projectDir, "dist", buildName+"_linux_amd64.tar.gz"))
}

package buildcmd

import (
	"context"
	core "dappco.re/go"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

const cmdProjectOSField = "o" + "s"

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	requireBuildCmdOK(t, ax.ExecDir(context.Background(), dir, "git", args...))

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
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(binDir, "gpg"), []byte(script), 0o755))

}

func TestBuildCmd_GetBuilderGood(t *testing.T) {
	t.Run("returns Python builder for python project type", func(t *testing.T) {
		builder := requireBuildCmdBuilder(t, getBuilder(build.ProjectTypePython))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("python", builder.Name()) {
			t.Fatalf("want %v, got %v", "python", builder.Name())
		}

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

	cfg := buildRuntimeConfig(storage.Local, "/project", "/project/dist", "binary", buildConfig, false, "", "v1.2.3")
	if !stdlibAssertEqual([]string{"-s", "-w"}, cfg.LDFlags) {
		t.Fatalf("want %v, got %v", []string{"-s", "-w"}, cfg.LDFlags)
	}
	if !stdlibAssertEqual([]string{"-trimpath"}, cfg.Flags) {
		t.Fatalf("want %v, got %v", []string{"-trimpath"}, cfg.Flags)
	}
	if !stdlibAssertEqual([]string{"integration"}, cfg.BuildTags) {
		t.Fatalf("want %v, got %v", []string{"integration"}, cfg.BuildTags)
	}
	if !stdlibAssertEqual([]string{"FOO=bar"}, cfg.Env) {
		t.Fatalf("want %v, got %v", []string{"FOO=bar"}, cfg.Env)
	}
	if !(cfg.CGO) {
		t.Fatal("expected true")
	}
	if !(cfg.Obfuscate) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("deno task bundle", cfg.DenoBuild) {
		t.Fatalf("want %v, got %v", "deno task bundle", cfg.DenoBuild)
	}
	if !(cfg.NSIS) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("embed", cfg.WebView2) {
		t.Fatalf("want %v, got %v", "embed", cfg.WebView2)
	}
	if !stdlibAssertEqual("Dockerfile.custom", cfg.Dockerfile) {
		t.Fatalf("want %v, got %v", "Dockerfile.custom", cfg.Dockerfile)
	}
	if !stdlibAssertEqual("ghcr.io", cfg.Registry) {
		t.Fatalf("want %v, got %v", "ghcr.io", cfg.Registry)
	}
	if !stdlibAssertEqual("owner/repo", cfg.Image) {
		t.Fatalf("want %v, got %v", "owner/repo", cfg.Image)
	}
	if !stdlibAssertEqual([]string{"latest", "{{.Version}}"}, cfg.Tags) {
		t.Fatalf("want %v, got %v", []string{"latest", "{{.Version}}"}, cfg.Tags)
	}
	if !stdlibAssertEqual(map[string]string{"VERSION": "{{.Version}}"}, cfg.BuildArgs) {
		t.Fatalf("want %v, got %v", map[string]string{"VERSION": "{{.Version}}"}, cfg.BuildArgs)
	}
	if !(cfg.Push) {
		t.Fatal("expected true")
	}
	if !(cfg.Load) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(".core/linuxkit/server.yml", cfg.LinuxKitConfig) {
		t.Fatalf("want %v, got %v", ".core/linuxkit/server.yml", cfg.LinuxKitConfig)
	}
	if !stdlibAssertEqual([]string{"iso", "qcow2"}, cfg.Formats) {
		t.Fatalf("want %v, got %v", []string{"iso", "qcow2"}, cfg.Formats)
	}
	if !stdlibAssertEqual("v1.2.3", cfg.Version) {
		t.Fatalf("want %v, got %v", "v1.2.3", cfg.Version)
	}

}

func TestBuildCmd_buildRuntimeConfig_ImageOverride_Good(t *testing.T) {
	buildConfig := &build.BuildConfig{
		Build: build.Build{
			Image: "owner/repo",
		},
	}

	cfg := buildRuntimeConfig(storage.Local, "/project", "/project/dist", "binary", buildConfig, true, "cli/image", "v2.0.0")
	if !stdlibAssertEqual("cli/image", cfg.Image) {
		t.Fatalf("want %v, got %v", "cli/image", cfg.Image)
	}
	if !(cfg.Push) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("v2.0.0", cfg.Version) {
		t.Fatalf("want %v, got %v", "v2.0.0", cfg.Version)
	}

}

func TestBuildCmd_buildRuntimeConfig_ClonesBuildArgs_Good(t *testing.T) {
	buildConfig := &build.BuildConfig{
		Build: build.Build{
			BuildArgs: map[string]string{"VERSION": "v1.2.3"},
		},
	}

	cfg := buildRuntimeConfig(storage.Local, "/project", "/project/dist", "binary", buildConfig, false, "", "v1.2.3")
	if stdlibAssertNil(cfg.BuildArgs) {
		t.Fatal("expected non-nil")
	}

	cfg.BuildArgs["VERSION"] = "mutated"
	if !stdlibAssertEqual("v1.2.3", buildConfig.Build.BuildArgs["VERSION"]) {
		t.Fatalf("want %v, got %v", "v1.2.3", buildConfig.Build.BuildArgs["VERSION"])
	}

}

func TestBuildCmd_resolveNoSign_Good(t *testing.T) {
	t.Run("keeps signing enabled by default", func(t *testing.T) {
		if resolveNoSign(false, true, false) {
			t.Fatal("expected false")
		}

	})

	t.Run("disables signing when no-sign is set", func(t *testing.T) {
		if !(resolveNoSign(true, true, false)) {
			t.Fatal("expected true")
		}

	})

	t.Run("disables signing when sign=false is set", func(t *testing.T) {
		if !(resolveNoSign(false, false, true)) {
			t.Fatal("expected true")
		}

	})

	t.Run("keeps signing enabled when sign=true is set", func(t *testing.T) {
		if resolveNoSign(false, true, true) {
			t.Fatal("expected false")
		}

	})
}

func TestBuildCmd_resolveBuildSignConfig_Good(t *testing.T) {
	t.Run("enables signing when notarize overrides disabled config", func(t *testing.T) {
		signCfg := resolveBuildSignConfig(build.DefaultConfig().Sign, ProjectBuildRequest{
			Notarize: true,
		})
		if !(signCfg.Enabled) {
			t.Fatal("expected true")
		}
		if !(signCfg.MacOS.Notarize) {
			t.Fatal("expected true")
		}

	})

	t.Run("preserves explicit no-sign over notarize", func(t *testing.T) {
		signCfg := resolveBuildSignConfig(build.DefaultConfig().Sign, ProjectBuildRequest{
			NoSign:   true,
			Notarize: true,
		})
		if signCfg.Enabled {
			t.Fatal("expected false")
		}
		if !(signCfg.MacOS.Notarize) {
			t.Fatal("expected true")
		}

	})

	t.Run("re-enables signing when config disabled but notarize requested", func(t *testing.T) {
		base := build.DefaultConfig().Sign
		base.Enabled = false

		signCfg := resolveBuildSignConfig(base, ProjectBuildRequest{
			Notarize: true,
		})
		if !(signCfg.Enabled) {
			t.Fatal("expected true")
		}
		if !(signCfg.MacOS.Notarize) {
			t.Fatal("expected true")
		}

	})
}

func TestBuildCmd_resolvePackageOutputs_Good(t *testing.T) {
	t.Run("leaves archive and checksum defaults alone when package is unset", func(t *testing.T) {
		archiveOutput, checksumOutput := resolvePackageOutputs(false, false, false, false, false, false)
		if archiveOutput {
			t.Fatal("expected false")
		}
		if checksumOutput {
			t.Fatal("expected false")
		}

	})

	t.Run("disables archive and checksum when package=false and neither output flag is explicit", func(t *testing.T) {
		archiveOutput, checksumOutput := resolvePackageOutputs(false, true, true, false, true, false)
		if archiveOutput {
			t.Fatal("expected false")
		}
		if checksumOutput {
			t.Fatal("expected false")
		}

	})

	t.Run("enables archive and checksum when package=true and neither output flag is explicit", func(t *testing.T) {
		archiveOutput, checksumOutput := resolvePackageOutputs(true, true, false, false, false, false)
		if !(archiveOutput) {
			t.Fatal("expected true")
		}
		if !(checksumOutput) {
			t.Fatal("expected true")
		}

	})

	t.Run("preserves explicit archive and checksum overrides over package=false", func(t *testing.T) {
		archiveOutput, checksumOutput := resolvePackageOutputs(false, true, true, true, false, true)
		if !(archiveOutput) {
			t.Fatal("expected true")
		}
		if checksumOutput {
			t.Fatal("expected false")
		}

	})
}

func TestBuildCmd_runProjectBuild_CIModeEmitsGitHubAnnotationOnError_Bad(t *testing.T) {
	projectDir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/demo\n"), 0o644))

	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
		cli.SetStdout(nil)
		cli.SetStderr(nil)
	})
	getProjectBuildWorkingDir = func() core.Result { return core.Ok(projectDir) }

	stdout := core.NewBuffer()
	cli.SetStdout(stdout)
	cli.SetStderr(stdout)

	result := runProjectBuild(ProjectBuildRequest{
		Context:     context.Background(),
		CIMode:      true,
		TargetsFlag: "linux",
	})
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(stdout.String(), emitCIAnnotationForTest(result)) {
		t.Fatalf("expected %v to contain %v", stdout.String(), emitCIAnnotationForTest(result))
	}

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
		if !stdlibAssertEqual([]string{"mlx", "debug", "release"}, cfg.Build.BuildTags) {
			t.Fatalf("want %v, got %v", []string{"mlx", "debug", "release"}, cfg.Build.BuildTags)
		}
		if !(cfg.Build.Obfuscate) {
			t.Fatal("expected true")
		}
		if !(cfg.Build.NSIS) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("download", cfg.Build.WebView2) {
			t.Fatalf("want %v, got %v", "download", cfg.Build.WebView2)
		}
		if !stdlibAssertEqual("deno task bundle", cfg.Build.DenoBuild) {
			t.Fatalf("want %v, got %v", "deno task bundle", cfg.Build.DenoBuild)
		}
		if !(cfg.Build.Cache.Enabled) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual(ax.Join(build.ConfigDir, "cache"), cfg.Build.Cache.Directory) {
			t.Fatalf("want %v, got %v", ax.Join(build.ConfigDir, "cache"), cfg.Build.Cache.Directory)
		}
		if !stdlibAssertEqual([]string{ax.Join("cache", "go-build"), ax.Join("cache", "go-mod")}, cfg.Build.Cache.Paths) {
			t.Fatalf("want %v, got %v", []string{ax.Join("cache", "go-build"), ax.Join("cache", "go-mod")}, cfg.Build.Cache.Paths)
		}
		if cfg.Sign.Enabled {
			t.Fatal("expected false")
		}

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
		if !(cfg.Build.Cache.Enabled) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("custom/cache", cfg.Build.Cache.Directory) {
			t.Fatalf("want %v, got %v", "custom/cache", cfg.Build.Cache.Directory)
		}
		if !stdlibAssertEqual([]string{"custom/go-build"}, cfg.Build.Cache.Paths) {
			t.Fatalf("want %v, got %v", []string{"custom/go-build"}, cfg.Build.Cache.Paths)
		}

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
		if cfg.Build.Cache.Enabled {
			t.Fatal("expected false")
		}
		if !stdlibAssertEqual("custom/cache", cfg.Build.Cache.Directory) {
			t.Fatalf("want %v, got %v", "custom/cache", cfg.Build.Cache.Directory)
		}
		if !stdlibAssertEqual([]string{"custom/go-build", "custom/go-mod"}, cfg.Build.Cache.Paths) {
			t.Fatalf("want %v, got %v", []string{"custom/go-build", "custom/go-mod"}, cfg.Build.Cache.Paths)
		}

	})

	t.Run("can force signing back on when config disabled it", func(t *testing.T) {
		cfg := build.DefaultConfig()
		cfg.Sign.Enabled = false

		applyProjectBuildOverrides(cfg, ProjectBuildRequest{
			Sign:    true,
			SignSet: true,
		})
		if !(cfg.Sign.Enabled) {
			t.Fatal("expected true")
		}

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
		if !stdlibAssertEqual("cli-name", resolveProjectBuildName("/tmp/project", cfg, "cli-name")) {
			t.Fatalf("want %v, got %v", "cli-name", resolveProjectBuildName("/tmp/project", cfg, "cli-name"))
		}

	})

	t.Run("falls back to project binary, then project name, then directory name", func(t *testing.T) {
		cfg := &build.BuildConfig{
			Project: build.Project{
				Name:   "project-name",
				Binary: "project-binary",
			},
		}
		if !stdlibAssertEqual("project-binary", resolveProjectBuildName("/tmp/project", cfg, "")) {
			t.Fatalf("want %v, got %v", "project-binary", resolveProjectBuildName("/tmp/project", cfg, ""))
		}

		cfg.Project.Binary = ""
		if !stdlibAssertEqual("project-name", resolveProjectBuildName("/tmp/project", cfg, "")) {
			t.Fatalf("want %v, got %v", "project-name", resolveProjectBuildName("/tmp/project", cfg, ""))
		}

		cfg.Project.Name = ""
		if !stdlibAssertEqual("project", resolveProjectBuildName("/tmp/project", cfg, "")) {
			t.Fatalf("want %v, got %v", "project", resolveProjectBuildName("/tmp/project", cfg, ""))
		}

	})
}

func TestBuildCmd_resolveArchiveFormat_Good(t *testing.T) {
	t.Run("uses cli override when present", func(t *testing.T) {
		format := requireBuildCmdArchiveFormat(t, resolveArchiveFormat("gz", "xz"))
		if !stdlibAssertEqual(build.ArchiveFormatXZ, format) {
			t.Fatalf("want %v, got %v", build.ArchiveFormatXZ, format)
		}

	})

	t.Run("falls back to config when cli override is empty", func(t *testing.T) {
		format := requireBuildCmdArchiveFormat(t, resolveArchiveFormat("zip", ""))
		if !stdlibAssertEqual(build.ArchiveFormatZip, format) {
			t.Fatalf("want %v, got %v", build.ArchiveFormatZip, format)
		}

	})
}

func TestBuildCmd_resolveBuildVersion_Good(t *testing.T) {
	dir := t.TempDir()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(dir, "README.md"), []byte("hello\n"), 0644))

	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feat: initial commit")
	runGit(t, dir, "tag", "v1.4.2")

	version := requireBuildCmdString(t, resolveBuildVersion(context.Background(), dir))
	if !stdlibAssertEqual("v1.4.2", version) {
		t.Fatalf("want %v, got %v", "v1.4.2", version)
	}

}

func TestBuildCmd_writeArtifactMetadata_Good(t *testing.T) {
	t.Setenv("GITHUB_SHA", "abc1234def5678")
	t.Setenv("GITHUB_REF", "refs/tags/v1.2.3")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")

	fs := storage.Local
	dir := t.TempDir()

	linuxDir := ax.Join(dir, "linux_amd64")
	windowsDir := ax.Join(dir, "windows_amd64")
	requireBuildCmdOK(t, ax.MkdirAll(linuxDir, 0755))
	requireBuildCmdOK(t, ax.MkdirAll(windowsDir, 0755))

	artifacts := []build.Artifact{
		{Path: ax.Join(linuxDir, "sample"), OS: "linux", Arch: "amd64"},
		{Path: ax.Join(windowsDir, "sample.exe"), OS: "windows", Arch: "amd64"},
	}

	requireBuildCmdOK(t, writeArtifactMetadata(fs, "sample", artifacts))

	verifyArtifactMeta := func(path string, expectedOS string, expectedArch string) {
		content := requireBuildCmdBytes(t, ax.ReadFile(path))

		var meta map[string]any
		requireBuildCmdOK(t, ax.JSONUnmarshal(content, &meta))
		if !stdlibAssertEqual("sample", meta["name"]) {
			t.Fatalf("want %v, got %v", "sample", meta["name"])
		}
		if !stdlibAssertEqual(expectedOS, meta[cmdProjectOSField]) {
			t.Fatalf("want %v, got %v", expectedOS, meta[cmdProjectOSField])
		}
		if !stdlibAssertEqual(expectedArch, meta["arch"]) {
			t.Fatalf("want %v, got %v", expectedArch, meta["arch"])
		}
		if !stdlibAssertEqual("v1.2.3", meta["tag"]) {
			t.Fatalf("want %v, got %v", "v1.2.3", meta["tag"])
		}
		if !stdlibAssertEqual("owner/repo", meta["repo"]) {
			t.Fatalf("want %v, got %v", "owner/repo", meta["repo"])
		}

	}

	verifyArtifactMeta(ax.Join(linuxDir, "artifact_meta.json"), "linux", "amd64")
	verifyArtifactMeta(ax.Join(windowsDir, "artifact_meta.json"), "windows", "amd64")
}

func TestBuildCmd_writeArtifactMetadata_SkipsChecksumArtifacts_Good(t *testing.T) {
	t.Setenv("GITHUB_SHA", "abc1234def5678")
	t.Setenv("GITHUB_REF", "refs/tags/v1.2.3")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")

	fs := storage.Local
	dir := t.TempDir()
	distDir := ax.Join(dir, "dist")
	requireBuildCmdOK(t, ax.MkdirAll(distDir, 0o755))

	checksumPath := ax.Join(distDir, "CHECKSUMS.txt")
	signaturePath := checksumPath + ".asc"
	requireBuildCmdOK(t, ax.WriteFile(checksumPath, []byte("checksums"), 0o644))
	requireBuildCmdOK(t, ax.WriteFile(signaturePath, []byte("signature"), 0o644))

	requireBuildCmdOK(t, writeArtifactMetadata(fs, "sample", []build.Artifact{
		{Path: checksumPath},
		{Path: signaturePath},
	}))
	if ax.Exists(ax.Join(distDir, "artifact_meta.json")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(distDir, "artifact_meta.json"))
	}

}

func TestBuildCmd_computeAndWriteChecksums_IncludesChecksumArtifacts_Good(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist")
	artifactPath := ax.Join(outputDir, "sample_linux_amd64.tar.gz")
	requireBuildCmdOK(t, ax.MkdirAll(outputDir, 0o755))
	requireBuildCmdOK(t, ax.WriteFile(artifactPath, []byte("archive"), 0o644))

	signCfg := build.DefaultConfig().Sign
	signCfg.Enabled = false

	artifacts := requireBuildCmdArtifacts(t, computeAndWriteChecksums(
		context.Background(),
		storage.Local,
		projectDir,
		outputDir,
		[]build.Artifact{{Path: artifactPath, OS: "linux", Arch: "amd64"}},
		signCfg,
		false,
		false,
	))

	paths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		paths = append(paths, artifact.Path)
	}
	if !stdlibAssertContains(paths, artifactPath) {
		t.Fatalf("expected %v to contain %v", paths, artifactPath)
	}
	if !stdlibAssertContains(paths, ax.Join(outputDir, "CHECKSUMS.txt")) {
		t.Fatalf("expected %v to contain %v", paths, ax.Join(outputDir, "CHECKSUMS.txt"))
	}
	if stdlibAssertContains(paths, ax.Join(outputDir, "CHECKSUMS.txt.asc")) {
		t.Fatalf("expected %v not to contain %v", paths, ax.Join(outputDir, "CHECKSUMS.txt.asc"))
	}
	requireBuildCmdOK(t, ax.Stat(ax.Join(outputDir, "CHECKSUMS.txt")))

}

func TestBuildCmd_computeAndWriteChecksums_IncludesSignatureArtifact_Good(t *testing.T) {
	binDir := t.TempDir()
	setupFakeGPG(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist")
	artifactPath := ax.Join(outputDir, "sample_linux_amd64.tar.gz")
	requireBuildCmdOK(t, ax.MkdirAll(outputDir, 0o755))
	requireBuildCmdOK(t, ax.WriteFile(artifactPath, []byte("archive"), 0o644))

	signCfg := build.DefaultConfig().Sign
	signCfg.Enabled = true
	signCfg.GPG.Key = "ABCD1234"

	artifacts := requireBuildCmdArtifacts(t, computeAndWriteChecksums(
		context.Background(),
		storage.Local,
		projectDir,
		outputDir,
		[]build.Artifact{{Path: artifactPath, OS: "linux", Arch: "amd64"}},
		signCfg,
		false,
		false,
	))

	paths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		paths = append(paths, artifact.Path)
	}
	if !stdlibAssertContains(paths, ax.Join(outputDir, "CHECKSUMS.txt")) {
		t.Fatalf("expected %v to contain %v", paths, ax.Join(outputDir, "CHECKSUMS.txt"))
	}
	if !stdlibAssertContains(paths, ax.Join(outputDir, "CHECKSUMS.txt.asc")) {
		t.Fatalf("expected %v to contain %v", paths, ax.Join(outputDir, "CHECKSUMS.txt.asc"))
	}
	requireBuildCmdOK(t, ax.Stat(ax.Join(outputDir, "CHECKSUMS.txt.asc")))

}

func TestBuildCmd_selectOutputArtifacts_Good(t *testing.T) {
	rawArtifacts := []build.Artifact{{Path: "dist/raw"}}
	archivedArtifacts := []build.Artifact{{Path: "dist/raw.tar.gz"}}
	checksummedArtifacts := []build.Artifact{{Path: "dist/raw.tar.gz", Checksum: "abc123"}}

	t.Run("prefers checksummed artifacts", func(t *testing.T) {
		selected := selectOutputArtifacts(rawArtifacts, archivedArtifacts, checksummedArtifacts)
		if !stdlibAssertEqual(checksummedArtifacts, selected) {
			t.Fatalf("want %v, got %v", checksummedArtifacts, selected)
		}

	})

	t.Run("falls back to archived artifacts", func(t *testing.T) {
		selected := selectOutputArtifacts(rawArtifacts, archivedArtifacts, nil)
		if !stdlibAssertEqual(archivedArtifacts, selected) {
			t.Fatalf("want %v, got %v", archivedArtifacts, selected)
		}

	})

	t.Run("falls back to raw artifacts", func(t *testing.T) {
		selected := selectOutputArtifacts(rawArtifacts, nil, nil)
		if !stdlibAssertEqual(rawArtifacts, selected) {
			t.Fatalf("want %v, got %v", rawArtifacts, selected)
		}

	})
}

func TestBuildCmd_runProjectBuild_PwaOverride_Good(t *testing.T) {
	expectedWD := requireBuildCmdString(t, ax.Getwd())

	original := runLocalPwaBuild
	t.Cleanup(func() {
		runLocalPwaBuild = original
	})

	called := false
	runLocalPwaBuild = func(ctx context.Context, projectDir string) core.Result {
		called = true
		if !stdlibAssertEqual(expectedWD, projectDir) {
			t.Fatalf("want %v, got %v", expectedWD, projectDir)
		}

		return core.Ok(nil)
	}

	requireBuildCmdOK(t, runProjectBuild(ProjectBuildRequest{
		Context:   context.Background(),
		BuildType: "pwa",
	}))
	if !(called) {
		t.Fatal("expected true")
	}

}

func TestBuildCmd_runProjectBuild_NoConfigGoPassthrough_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() core.Result {
		return core.Ok(projectDir)
	}
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	requireBuildCmdOK(t, runProjectBuild(ProjectBuildRequest{
		Context:       context.Background(),
		ArchiveOutput: true,
	}))
	requireBuildCmdOK(t, ax.Stat(ax.Join(projectDir, "passthrough")))
	if ax.Exists(ax.Join(projectDir, "dist")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(projectDir, "dist"))
	}

}

func TestBuildCmd_runProjectBuild_ConfiguredBuildDefaultsToRawArtifacts_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() core.Result {
		return core.Ok(projectDir)
	}
	requireBuildCmdOK(t, ax.MkdirAll(ax.Join(projectDir, ".core"), 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/configured\n\ngo 1.24\n"), 0o644))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte("version: 1\n"+"project:\n"+"  name: configured\n"+"  binary: configured\n"+"targets:\n"+"  - os: "+runtime.GOOS+"\n"+"    arch: "+runtime.GOARCH+"\n"+"sign:\n"+"  enabled: false\n"), 0o644))

	requireBuildCmdOK(t, runProjectBuild(ProjectBuildRequest{
		Context: context.Background(),
	}))

	expectedBinary := ax.Join(projectDir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "configured")
	if runtime.GOOS == "windows" {
		expectedBinary += ".exe"
	}
	requireBuildCmdOK(t, ax.Stat(expectedBinary))
	if ax.Exists(ax.Join(projectDir, "dist", "CHECKSUMS.txt")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(projectDir, "dist", "CHECKSUMS.txt"))
	}
	if ax.Exists(ax.Join(projectDir, "dist", "configured_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.gz")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(projectDir, "dist", "configured_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.gz"))
	}
	if ax.Exists(ax.Join(projectDir, "dist", "configured_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.xz")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(projectDir, "dist", "configured_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.xz"))
	}
	if ax.Exists(ax.Join(projectDir, "dist", "configured_"+runtime.GOOS+"_"+runtime.GOARCH+".zip")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(projectDir, "dist", "configured_"+runtime.GOOS+"_"+runtime.GOARCH+".zip"))
	}

}

func TestBuildCmd_shouldUseGoBuildPassthrough_Good(t *testing.T) {
	projectDir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	t.Run("keeps simple no-config go builds on passthrough", func(t *testing.T) {
		if !(shouldUseGoBuildPassthrough(storage.Local, projectDir, ProjectBuildRequest{})) {
			t.Fatal("expected true")
		}

	})

	t.Run("uses the pipeline for ci mode", func(t *testing.T) {
		if (shouldUseGoBuildPassthrough(storage.Local, projectDir, ProjectBuildRequest{CIMode: true})) {
			t.Fatal("expected false")
		}

	})

	t.Run("uses the pipeline for explicit archive requests", func(t *testing.T) {
		if (shouldUseGoBuildPassthrough(storage.Local, projectDir, ProjectBuildRequest{ArchiveOutput: true, ArchiveOutputSet: true})) {
			t.Fatal("expected false")
		}

	})

	t.Run("uses the pipeline for explicit package requests", func(t *testing.T) {
		if (shouldUseGoBuildPassthrough(storage.Local, projectDir, ProjectBuildRequest{ArchiveOutput: true, ChecksumOutput: true, PackageSet: true})) {
			t.Fatal("expected false")
		}

	})

	t.Run("uses the pipeline for explicit versioning", func(t *testing.T) {
		if (shouldUseGoBuildPassthrough(storage.Local, projectDir, ProjectBuildRequest{Version: "v1.2.3"})) {
			t.Fatal("expected false")
		}

	})

	t.Run("uses the pipeline for Wails projects even without config", func(t *testing.T) {
		wailsDir := t.TempDir()
		requireBuildCmdOK(t, ax.WriteFile(ax.Join(wailsDir, "go.mod"), []byte("module example.com/wails\n\ngo 1.24\n"), 0o644))
		requireBuildCmdOK(t, ax.WriteFile(ax.Join(wailsDir, "wails.json"), []byte(`{"name":"demo"}`), 0o644))
		if (shouldUseGoBuildPassthrough(storage.Local, wailsDir, ProjectBuildRequest{})) {
			t.Fatal("expected false")
		}

	})

	t.Run("uses the pipeline for multi-type Go and Node projects", func(t *testing.T) {
		stackDir := t.TempDir()
		requireBuildCmdOK(t, ax.WriteFile(ax.Join(stackDir, "go.mod"), []byte("module example.com/fullstack\n\ngo 1.24\n"), 0o644))
		requireBuildCmdOK(t, ax.WriteFile(ax.Join(stackDir, "package.json"), []byte(`{"name":"fullstack"}`), 0o644))
		if (shouldUseGoBuildPassthrough(storage.Local, stackDir, ProjectBuildRequest{})) {
			t.Fatal("expected false")
		}

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
	getProjectBuildWorkingDir = func() core.Result {
		return core.Ok(projectDir)
	}
	requireBuildCmdOK(t, ax.MkdirAll(outputDir, 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	requireBuildCmdOK(t, runProjectBuild(ProjectBuildRequest{
		Context:     context.Background(),
		TargetsFlag: "linux/amd64",
		OutputDir:   outputDir,
		BuildName:   "custom-binary",
	}))
	requireBuildCmdOK(t, ax.Stat(outputPath))

}

func TestBuildCmd_runProjectBuild_NoConfigGoCIModeUsesPipeline_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() core.Result {
		return core.Ok(projectDir)
	}
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	buildName := ax.Base(projectDir)

	requireBuildCmdOK(t, runProjectBuild(ProjectBuildRequest{
		Context:        context.Background(),
		CIMode:         true,
		TargetsFlag:    "linux/amd64",
		ArchiveOutput:  false,
		ChecksumOutput: false,
	}))
	if ax.Exists(ax.Join(projectDir, "passthrough")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(projectDir, "passthrough"))
	}
	requireBuildCmdOK(t, ax.Stat(ax.Join(projectDir, "dist", "linux_amd64", buildName)))

}

func TestBuildCmd_runProjectBuild_CIModeCopiesCIStampedArtifacts_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() core.Result {
		return core.Ok(projectDir)
	}

	t.Setenv("GITHUB_SHA", "abc1234def5678901234567890123456789012345")
	t.Setenv("GITHUB_REF", "refs/tags/v1.2.3")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	requireBuildCmdOK(t, runProjectBuild(ProjectBuildRequest{
		Context:     context.Background(),
		CIMode:      true,
		TargetsFlag: "linux/amd64",
	}))

	ciArtifactPath := ax.Join(projectDir, "dist", "linux_amd64", ax.Base(projectDir)+"_linux_amd64_v1.2.3")
	requireBuildCmdOK(t, ax.Stat(ciArtifactPath))
	requireBuildCmdOK(t, ax.Stat(ax.Join(projectDir, "dist", "linux_amd64", ax.Base(projectDir))))
	requireBuildCmdOK(t, ax.Stat(ax.Join(projectDir, "dist", "linux_amd64", "artifact_meta.json")))

}

func TestBuildCmd_runProjectBuild_NoConfigGoArchiveRequestUsesPipeline_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getProjectBuildWorkingDir
	t.Cleanup(func() {
		getProjectBuildWorkingDir = originalGetwd
	})
	getProjectBuildWorkingDir = func() core.Result {
		return core.Ok(projectDir)
	}
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/passthrough\n\ngo 1.24\n"), 0o644))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	buildName := ax.Base(projectDir)

	requireBuildCmdOK(t, runProjectBuild(ProjectBuildRequest{
		Context:          context.Background(),
		TargetsFlag:      "linux/amd64",
		ArchiveOutput:    true,
		ArchiveOutputSet: true,
		ChecksumOutput:   false,
	}))
	if ax.Exists(ax.Join(projectDir, "passthrough")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(projectDir, "passthrough"))
	}
	requireBuildCmdOK(t, ax.Stat(ax.Join(projectDir, "dist", "linux_amd64", buildName)))
	requireBuildCmdOK(t, ax.Stat(ax.Join(projectDir, "dist", buildName+"_linux_amd64.tar.gz")))

}

package build

import (
	"testing"

	"dappco.re/go/core/io"
)

func TestBuild_RuntimeConfigFromBuildConfig_Good(t *testing.T) {
	source := &BuildConfig{
		Project: Project{
			Name:   "Core",
			Main:   "./cmd/core",
			Binary: "core",
		},
		Build: Build{
			CGO:            true,
			Obfuscate:      true,
			DenoBuild:      "deno task bundle",
			NSIS:           true,
			WebView2:       "embed",
			Flags:          []string{"-mod=readonly"},
			LDFlags:        []string{"-s", "-w"},
			BuildTags:      []string{"integration"},
			Env:            []string{"FOO=bar"},
			Cache:          CacheConfig{Enabled: true, Paths: []string{"/tmp/go-build"}},
			Dockerfile:     "build/Dockerfile",
			Registry:       "ghcr.io",
			Image:          "host-uk/core",
			Tags:           []string{"latest"},
			BuildArgs:      map[string]string{"VERSION": "1.2.3"},
			Push:           false,
			Load:           true,
			LinuxKitConfig: ".core/linuxkit/core.yaml",
			Formats:        []string{"iso", "qcow2"},
		},
		LinuxKit: LinuxKitConfig{
			Base:     "core-dev",
			Packages: []string{"git"},
			Mounts:   []string{"/workspace"},
			GPU:      true,
			Formats:  []string{"oci", "apple"},
			Registry: "ghcr.io/dappcore",
		},
	}

	cfg := RuntimeConfigFromBuildConfig(io.Local, "/workspace/core", "/workspace/core/dist", "core-bin", source, true, "override/image", "v1.2.3")
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual(io.Local, cfg.FS) {
		t.Fatalf("want %v, got %v", io.Local, cfg.FS)
	}
	if !stdlibAssertEqual(source.Project, cfg.Project) {
		t.Fatalf("want %v, got %v", source.Project, cfg.Project)
	}
	if !stdlibAssertEqual("/workspace/core", cfg.ProjectDir) {
		t.Fatalf("want %v, got %v", "/workspace/core", cfg.ProjectDir)
	}
	if !stdlibAssertEqual("/workspace/core/dist", cfg.OutputDir) {
		t.Fatalf("want %v, got %v", "/workspace/core/dist", cfg.OutputDir)
	}
	if !stdlibAssertEqual("core-bin", cfg.Name) {
		t.Fatalf("want %v, got %v", "core-bin", cfg.Name)
	}
	if !stdlibAssertEqual("v1.2.3", cfg.Version) {
		t.Fatalf("want %v, got %v", "v1.2.3", cfg.Version)
	}
	if !stdlibAssertEqual([]string{"-mod=readonly"}, cfg.Flags) {
		t.Fatalf("want %v, got %v", []string{"-mod=readonly"}, cfg.Flags)
	}
	if !stdlibAssertEqual([]string{"-s", "-w"}, cfg.LDFlags) {
		t.Fatalf("want %v, got %v", []string{"-s", "-w"}, cfg.LDFlags)
	}
	if !stdlibAssertEqual([]string{"integration"}, cfg.BuildTags) {
		t.Fatalf("want %v, got %v", []string{"integration"}, cfg.BuildTags)
	}
	if !stdlibAssertEqual([]string{"FOO=bar"}, cfg.Env) {
		t.Fatalf("want %v, got %v", []string{"FOO=bar"}, cfg.Env)
	}
	if !stdlibAssertEqual(CacheConfig{Enabled: true, Paths: []string{"/tmp/go-build"}}, cfg.Cache) {
		t.Fatalf("want %v, got %v", CacheConfig{Enabled: true, Paths: []string{"/tmp/go-build"}}, cfg.Cache)
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
	if !stdlibAssertEqual("build/Dockerfile", cfg.Dockerfile) {
		t.Fatalf("want %v, got %v", "build/Dockerfile", cfg.Dockerfile)
	}
	if !stdlibAssertEqual("ghcr.io", cfg.Registry) {
		t.Fatalf("want %v, got %v", "ghcr.io", cfg.Registry)
	}
	if !stdlibAssertEqual("override/image", cfg.Image) {
		t.Fatalf("want %v, got %v", "override/image", cfg.Image)
	}
	if !stdlibAssertEqual([]string{"latest"}, cfg.Tags) {
		t.Fatalf("want %v, got %v", []string{"latest"}, cfg.Tags)
	}
	if !stdlibAssertEqual(map[string]string{"VERSION": "1.2.3"}, cfg.BuildArgs) {
		t.Fatalf("want %v, got %v", map[string]string{"VERSION": "1.2.3"}, cfg.BuildArgs)
	}
	if !(cfg.Push) {
		t.Fatal("expected true")
	}
	if !(cfg.Load) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(".core/linuxkit/core.yaml", cfg.LinuxKitConfig) {
		t.Fatalf("want %v, got %v", ".core/linuxkit/core.yaml", cfg.LinuxKitConfig)
	}
	if !stdlibAssertEqual([]string{"iso", "qcow2"}, cfg.Formats) {
		t.Fatalf("want %v, got %v", []string{"iso", "qcow2"}, cfg.Formats)
	}
	if !stdlibAssertEqual(LinuxKitConfig{Base: "core-dev", Packages: []string{"git"}, Mounts: []string{"/workspace"}, GPU: true, Formats: []string{"oci", "apple"}, Registry: "ghcr.io/dappcore"}, cfg.LinuxKit) {
		t.Fatalf("want %v, got %v", LinuxKitConfig{Base: "core-dev", Packages: []string{"git"}, Mounts: []string{"/workspace"}, GPU: true, Formats: []string{"oci", "apple"}, Registry: "ghcr.io/dappcore"}, cfg.LinuxKit)
	}

	cfg.Flags[0] = "-trimpath"
	cfg.LDFlags[0] = "-X"
	cfg.BuildTags[0] = "ui"
	cfg.Env[0] = "BAR=baz"
	cfg.Tags[0] = "stable"
	cfg.BuildArgs["VERSION"] = "2.0.0"
	cfg.LinuxKit.Packages[0] = "task"
	if !stdlibAssertEqual([]string{"-mod=readonly"}, source.Build.Flags) {
		t.Fatalf("want %v, got %v", []string{"-mod=readonly"}, source.Build.Flags)
	}
	if !stdlibAssertEqual([]string{"-s", "-w"}, source.Build.LDFlags) {
		t.Fatalf("want %v, got %v", []string{"-s", "-w"}, source.Build.LDFlags)
	}
	if !stdlibAssertEqual([]string{"integration"}, source.Build.BuildTags) {
		t.Fatalf("want %v, got %v", []string{"integration"}, source.Build.BuildTags)
	}
	if !stdlibAssertEqual([]string{"FOO=bar"}, source.Build.Env) {
		t.Fatalf("want %v, got %v", []string{"FOO=bar"}, source.Build.Env)
	}
	if !stdlibAssertEqual([]string{"latest"}, source.Build.Tags) {
		t.Fatalf("want %v, got %v", []string{"latest"}, source.Build.Tags)
	}
	if !stdlibAssertEqual(map[string]string{"VERSION": "1.2.3"}, source.Build.BuildArgs) {
		t.Fatalf("want %v, got %v", map[string]string{"VERSION": "1.2.3"}, source.Build.BuildArgs)
	}
	if !stdlibAssertEqual([]string{"git"}, source.LinuxKit.Packages) {
		t.Fatalf("want %v, got %v", []string{"git"}, source.LinuxKit.Packages)
	}

}

func TestBuild_RuntimeConfigFromBuildConfig_ExpandsVersionTemplates_Good(t *testing.T) {
	source := &BuildConfig{
		Build: Build{
			Flags:   []string{"-X-build=v{{.Version}}"},
			LDFlags: []string{"-X main.Version={{.Tag}}"},
			Env:     []string{"RELEASE_TAG={{.Tag}}", "IMAGE_TAG=v{{.Version}}"},
		},
	}

	cfg := RuntimeConfigFromBuildConfig(io.Local, "/workspace/core", "/workspace/core/dist", "core-bin", source, false, "", "v1.2.3")
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual([]string{"-X-build=v1.2.3"}, cfg.Flags) {
		t.Fatalf("want %v, got %v", []string{"-X-build=v1.2.3"}, cfg.Flags)
	}
	if !stdlibAssertEqual([]string{"-X main.Version=v1.2.3"}, cfg.LDFlags) {
		t.Fatalf("want %v, got %v", []string{"-X main.Version=v1.2.3"}, cfg.LDFlags)
	}
	if !stdlibAssertEqual([]string{"RELEASE_TAG=v1.2.3", "IMAGE_TAG=v1.2.3"}, cfg.Env) {
		t.Fatalf("want %v, got %v", []string{"RELEASE_TAG=v1.2.3", "IMAGE_TAG=v1.2.3"}, cfg.Env)
	}
	if !stdlibAssertEqual([]string{"-X-build=v{{.Version}}"}, source.Build.Flags) {
		t.Fatalf("want %v, got %v", []string{"-X-build=v{{.Version}}"}, source.Build.Flags)
	}
	if !stdlibAssertEqual([]string{"-X main.Version={{.Tag}}"}, source.Build.LDFlags) {
		t.Fatalf("want %v, got %v", []string{"-X main.Version={{.Tag}}"}, source.Build.LDFlags)
	}
	if !stdlibAssertEqual([]string{"RELEASE_TAG={{.Tag}}", "IMAGE_TAG=v{{.Version}}"}, source.Build.Env) {
		t.Fatalf("want %v, got %v", []string{"RELEASE_TAG={{.Tag}}", "IMAGE_TAG=v{{.Version}}"}, source.Build.Env)
	}

}

func TestBuild_RuntimeConfigFromBuildConfig_StripsUnsafeVersionTemplateFlags(t *testing.T) {
	source := &BuildConfig{
		Build: Build{
			LDFlags: []string{
				"-s",
				"-w",
				"-X main.Version={{.Tag}}",
				"-X build.commit=abc123",
			},
		},
	}

	cfg := RuntimeConfigFromBuildConfig(io.Local, "/workspace/core", "/workspace/core/dist", "core-bin", source, false, "", "v1.2.3 -bad")
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual([]string{"-s", "-w", "-X build.commit=abc123"}, cfg.LDFlags) {
		t.Fatalf("want %v, got %v", []string{"-s", "-w", "-X build.commit=abc123"}, cfg.LDFlags)
	}
	if !stdlibAssertEmpty(cfg.Flags) {
		t.Fatalf("expected empty, got %v", cfg.Flags)
	}
	if !stdlibAssertEmpty(cfg.Env) {
		t.Fatalf("expected empty, got %v", cfg.Env)
	}
	if !stdlibAssertEqual([]string{"-s", "-w", "-X main.Version={{.Tag}}", "-X build.commit=abc123"}, source.Build.LDFlags) {
		t.Fatalf("want %v, got %v", []string{"-s", "-w", "-X main.Version={{.Tag}}", "-X build.commit=abc123"}, source.Build.LDFlags)
	}

}

func TestBuild_RuntimeConfigFromBuildConfig_UsesRFCPreBuildAliases_Good(t *testing.T) {
	source := &BuildConfig{
		PreBuild: PreBuild{
			Deno: "deno task build",
			Npm:  "npm run build",
		},
	}

	cfg := RuntimeConfigFromBuildConfig(io.Local, "/workspace/core", "/workspace/core/dist", "core-bin", source, false, "", "v1.2.3")
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("deno task build", cfg.DenoBuild) {
		t.Fatalf("want %v, got %v", "deno task build", cfg.DenoBuild)
	}
	if !stdlibAssertEqual("npm run build", cfg.NpmBuild) {
		t.Fatalf("want %v, got %v", "npm run build", cfg.NpmBuild)
	}

}

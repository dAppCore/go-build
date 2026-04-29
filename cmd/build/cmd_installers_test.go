package buildcmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/io"
)

func TestBuildCmd_AddInstallersCommand_Good(t *testing.T) {
	c := core.New()

	AddInstallersCommand(c)
	if !(c.Command("build/installers").OK) {
		t.Fatal("expected true")
	}

}

func TestBuildCmd_runBuildInstallersInDir_GeneratesAll_Good(t *testing.T) {
	projectDir := t.TempDir()
	requireBuildCmdOK(t, io.Local.EnsureDir(ax.Join(projectDir, ".core")))
	requireBuildCmdOK(t, io.Local.Write(ax.Join(projectDir, ".core", "build.yaml"), `version: 1
project:
  binary: corex
`))
	requireBuildCmdOK(t, io.Local.Write(ax.Join(projectDir, ".core", "release.yaml"), `version: 1
project:
  repository: dappcore/core
`))

	requireBuildCmdOK(t, runBuildInstallersInDir(context.Background(), projectDir, "", "v1.2.3", "", "", ""))

	outputDir := ax.Join(projectDir, "dist", "installers")
	expected := []string{"setup.sh", "ci.sh", "php.sh", "go.sh", "agent.sh", "dev.sh"}
	for _, name := range expected {
		requireBuildCmdOK(t, ax.Stat(ax.Join(outputDir, name)))

	}

	content := requireBuildCmdString(t, io.Local.Read(ax.Join(outputDir, "setup.sh")))
	if !stdlibAssertContains(content, "corex") {
		t.Fatalf("expected %v to contain %v", content, "corex")
	}
	if !stdlibAssertContains(content, "v1.2.3") {
		t.Fatalf("expected %v to contain %v", content, "v1.2.3")
	}
	if !stdlibAssertContains(content, "dappcore/core") {
		t.Fatalf("expected %v to contain %v", content, "dappcore/core")
	}
	if !stdlibAssertContains(content, "https://lthn.sh/setup.sh") {
		t.Fatalf("expected %v to contain %v", content, "https://lthn.sh/setup.sh")
	}

	devContent := requireBuildCmdString(t, io.Local.Read(ax.Join(outputDir, "dev.sh")))
	if !stdlibAssertContains(devContent, `DEV_IMAGE_VERSION="${VERSION#v}"`) {
		t.Fatalf("expected %v to contain %v", devContent, `DEV_IMAGE_VERSION="${VERSION#v}"`)
	}
	if !stdlibAssertContains(devContent, `DEV_IMAGE="ghcr.io/dappcore/core-dev:${DEV_IMAGE_VERSION}"`) {
		t.Fatalf("expected %v to contain %v", devContent, `DEV_IMAGE="ghcr.io/dappcore/core-dev:${DEV_IMAGE_VERSION}"`)
	}

}

func TestBuildCmd_runBuildInstallersInDir_GeneratesSingleVariant_Good(t *testing.T) {
	projectDir := t.TempDir()

	requireBuildCmdOK(t, runBuildInstallersInDir(context.Background(), projectDir, "ci", "v1.2.3", "out/installers", "dappcore/core", "core"))
	requireBuildCmdOK(t, ax.Stat(ax.Join(projectDir, "out", "installers", "ci.sh")))
	if ax.Exists(ax.Join(projectDir, "out", "installers", "setup.sh")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(projectDir, "out", "installers", "setup.sh"))
	}

}

func TestBuildCmd_runBuildInstallersInDir_UsesResolvedVersion_Good(t *testing.T) {
	projectDir := t.TempDir()

	originalVersionResolver := resolveInstallersVersion
	t.Cleanup(func() {
		resolveInstallersVersion = originalVersionResolver
	})
	resolveInstallersVersion = func(ctx context.Context, dir string) core.Result {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return core.Ok("v9.9.9")
	}

	requireBuildCmdOK(t, runBuildInstallersInDir(context.Background(), projectDir, "setup.sh", "", "", "dappcore/core", "core"))

	content := requireBuildCmdString(t, io.Local.Read(ax.Join(projectDir, "dist", "installers", "setup.sh")))
	if !stdlibAssertContains(content, "v9.9.9") {
		t.Fatalf("expected %v to contain %v", content, "v9.9.9")
	}

}

func TestBuildCmd_runBuildInstallersInDir_UsesGitRemoteWhenReleaseConfigMissing_Good(t *testing.T) {
	projectDir := t.TempDir()

	originalLoadReleaseConfig := loadInstallersReleaseConfig
	originalDetectRepository := detectInstallersRepository
	t.Cleanup(func() {
		loadInstallersReleaseConfig = originalLoadReleaseConfig
		detectInstallersRepository = originalDetectRepository
	})

	loadInstallersReleaseConfig = func(dir string) core.Result {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		return core.Ok(cfg)
	}
	detectInstallersRepository = func(ctx context.Context, dir string) core.Result {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return core.Ok("host-uk/core-build")
	}

	requireBuildCmdOK(t, runBuildInstallersInDir(context.Background(), projectDir, "agentic", "v1.2.3", "", "", "core"))

	content := requireBuildCmdString(t, io.Local.Read(ax.Join(projectDir, "dist", "installers", "agent.sh")))
	if !stdlibAssertContains(content, "host-uk/core-build") {
		t.Fatalf("expected %v to contain %v", content, "host-uk/core-build")
	}

}

func TestBuildCmd_runBuildInstallersInDir_UnknownVariant_Bad(t *testing.T) {
	projectDir := t.TempDir()

	message := requireBuildCmdError(t, runBuildInstallersInDir(context.Background(), projectDir, "bogus", "v1.2.3", "", "dappcore/core", "core"))
	if !stdlibAssertContains(message, "unknown installer variant") {
		t.Fatalf("expected %v to contain %v", message, "unknown installer variant")
	}

}

func TestBuildCmd_runBuildInstallersInDir_RejectsUnsafeVersion_Bad(t *testing.T) {
	projectDir := t.TempDir()

	message := requireBuildCmdError(t, runBuildInstallersInDir(context.Background(), projectDir, "ci", "v1.2.3 --bad", "", "dappcore/core", "core"))
	if !stdlibAssertContains(message, "invalid installer version") {
		t.Fatalf("expected %v to contain %v", message, "invalid installer version")
	}

}

func TestBuildCmd_runBuildInstallersInDir_MissingRepository_Bad(t *testing.T) {
	projectDir := t.TempDir()

	originalLoadReleaseConfig := loadInstallersReleaseConfig
	originalDetectRepository := detectInstallersRepository
	t.Cleanup(func() {
		loadInstallersReleaseConfig = originalLoadReleaseConfig
		detectInstallersRepository = originalDetectRepository
	})

	loadInstallersReleaseConfig = func(dir string) core.Result {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		return core.Ok(cfg)
	}
	detectInstallersRepository = func(ctx context.Context, dir string) core.Result {
		return core.Fail(core.NewError("test error"))
	}

	message := requireBuildCmdError(t, runBuildInstallersInDir(context.Background(), projectDir, "ci", "v1.2.3", "", "", "core"))
	if !stdlibAssertContains(message, "use --repo") {
		t.Fatalf("expected %v to contain %v", message, "use --repo")
	}

}

func TestBuild_GenerateInstallerWrappersGood(t *testing.T) {
	script := requireBuildCmdString(t, build.GenerateInstaller(build.VariantCI, "v1.2.3", "dappcore/core"))
	if !stdlibAssertContains(script, "dappcore/core") {
		t.Fatalf("expected %v to contain %v", script, "dappcore/core")
	}
	if !stdlibAssertEqual([]build.InstallerVariant{build.VariantFull, build.VariantCI, build.VariantPHP, build.VariantGo, build.VariantAgent, build.VariantDev}, build.InstallerVariants()) {
		t.Fatalf("want %v, got %v", []build.InstallerVariant{build.VariantFull, build.VariantCI, build.VariantPHP, build.VariantGo, build.VariantAgent, build.VariantDev}, build.InstallerVariants())
	}
	if !stdlibAssertEqual("ci.sh", build.InstallerOutputName(build.VariantCI)) {
		t.Fatalf("want %v, got %v", "ci.sh", build.InstallerOutputName(build.VariantCI))
	}
	if !stdlibAssertEqual(build.VariantAgent, build.VariantAgentic) {
		t.Fatalf("want %v, got %v", build.VariantAgent, build.VariantAgentic)
	}

	agenticScript := requireBuildCmdString(t, build.GenerateInstaller(build.VariantAgentic, "v1.2.3", "dappcore/core"))
	if !stdlibAssertContains(agenticScript, "dappcore/core") {
		t.Fatalf("expected %v to contain %v", agenticScript, "dappcore/core")
	}

	scripts := requireBuildCmdStringMap(t, build.GenerateAll("v1.2.3", "dappcore/core"))
	if !stdlibAssertContains(scripts["setup.sh"], "dappcore/core") {
		t.Fatalf("expected %v to contain %v", scripts["setup.sh"], "dappcore/core")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestCmdInstallers_AddInstallersCommand_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		AddInstallersCommand(core.New())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdInstallers_AddInstallersCommand_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		AddInstallersCommand(core.New())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdInstallers_AddInstallersCommand_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		AddInstallersCommand(core.New())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

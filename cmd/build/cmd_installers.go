package buildcmd

import (
	"context"
	"strings"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/internal/cmdutil"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/release"
	"dappco.re/go/core/build/pkg/release/publishers"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

var (
	getInstallersWorkingDir     = ax.Getwd
	loadInstallersBuildConfig   = build.LoadConfig
	loadInstallersReleaseConfig = release.LoadConfig
	resolveInstallersVersion    = resolveBuildVersion
	detectInstallersRepository  = publishers.DetectGitHubRepository
)

// BuildInstallersRequest groups the inputs for `core build installers`.
type BuildInstallersRequest struct {
	Context    context.Context
	Variant    string
	Version    string
	OutputDir  string
	Repo       string
	BinaryName string
}

// AddInstallersCommand registers the installer generation command.
func AddInstallersCommand(c *core.Core) {
	c.Command("build/installers", core.Command{
		Description: "Generate installer scripts",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runBuildInstallers(BuildInstallersRequest{
				Context:    cmdutil.ContextOrBackground(),
				Variant:    cmdutil.OptionString(opts, "variant"),
				Version:    cmdutil.OptionString(opts, "version"),
				OutputDir:  cmdutil.OptionString(opts, "output"),
				Repo:       cmdutil.OptionString(opts, "repo"),
				BinaryName: cmdutil.OptionString(opts, "name", "binary"),
			}))
		},
	})
}

func runBuildInstallers(req BuildInstallersRequest) error {
	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}

	projectDir, err := getInstallersWorkingDir()
	if err != nil {
		return coreerr.E("build.runBuildInstallers", "failed to get working directory", err)
	}

	return runBuildInstallersInDir(ctx, projectDir, req.Variant, req.Version, req.OutputDir, req.Repo, req.BinaryName)
}

func runBuildInstallersInDir(ctx context.Context, projectDir, variant, version, outputDir, repo, binaryName string) error {
	filesystem := io.Local

	buildConfig, err := loadInstallersBuildConfig(filesystem, projectDir)
	if err != nil {
		return coreerr.E("build.runBuildInstallers", "failed to load build config", err)
	}

	installerVersion := strings.TrimSpace(version)
	if installerVersion == "" {
		installerVersion, err = resolveInstallersVersion(ctx, projectDir)
		if err != nil {
			return coreerr.E("build.runBuildInstallers", "failed to determine installer version; use --version to override", err)
		}
	}

	installerRepo := strings.TrimSpace(repo)
	if installerRepo == "" {
		installerRepo, err = resolveInstallersRepository(ctx, projectDir)
		if err != nil {
			return err
		}
	}

	if outputDir == "" {
		outputDir = ax.Join(projectDir, "dist", "installers")
	} else if !ax.IsAbs(outputDir) {
		outputDir = ax.Join(projectDir, outputDir)
	}

	if err := filesystem.EnsureDir(outputDir); err != nil {
		return coreerr.E("build.runBuildInstallers", "failed to create output directory", err)
	}

	cfg := build.InstallerConfig{
		Version:    installerVersion,
		Repo:       installerRepo,
		BinaryName: build.ResolveBuildName(projectDir, buildConfig, binaryName),
	}

	normalizedVariant, ok := normalizeInstallersVariant(variant)
	if !ok {
		return coreerr.E("build.runBuildInstallers", "unknown installer variant: "+strings.TrimSpace(variant), nil)
	}

	cli.Print("%s %s\n", buildHeaderStyle.Render("Installers"), "generating installer scripts")

	if normalizedVariant != "" {
		return writeInstallerVariant(filesystem, projectDir, outputDir, normalizedVariant, cfg)
	}

	for _, candidate := range build.InstallerVariants() {
		if err := writeInstallerVariant(filesystem, projectDir, outputDir, candidate, cfg); err != nil {
			return err
		}
	}

	return nil
}

func writeInstallerVariant(filesystem io.Medium, projectDir, outputDir string, variant build.InstallerVariant, cfg build.InstallerConfig) error {
	scriptName := build.InstallerOutputName(variant)
	if scriptName == "" {
		return coreerr.E("build.writeInstallerVariant", "unknown installer variant: "+string(variant), nil)
	}

	script, err := build.GenerateInstaller(variant, cfg)
	if err != nil {
		return coreerr.E("build.writeInstallerVariant", "failed to generate "+scriptName, err)
	}

	targetPath := ax.Join(outputDir, scriptName)
	if err := filesystem.WriteMode(targetPath, script, 0o755); err != nil {
		return coreerr.E("build.writeInstallerVariant", "failed to write "+scriptName, err)
	}

	relPath, relErr := ax.Rel(projectDir, targetPath)
	if relErr != nil {
		relPath = targetPath
	}
	cli.Print("  %s\n", relPath)

	return nil
}

func resolveInstallersRepository(ctx context.Context, projectDir string) (string, error) {
	releaseConfig, err := loadInstallersReleaseConfig(projectDir)
	if err != nil {
		return "", coreerr.E("build.resolveInstallersRepository", "failed to load release config", err)
	}

	if releaseConfig != nil {
		repo := strings.TrimSpace(releaseConfig.GetRepository())
		if repo != "" {
			return repo, nil
		}
	}

	repo, err := detectInstallersRepository(ctx, projectDir)
	if err != nil {
		return "", coreerr.E("build.resolveInstallersRepository", "failed to determine repository; use --repo or configure .core/release.yaml project.repository", err)
	}

	return repo, nil
}

func normalizeInstallersVariant(value string) (build.InstallerVariant, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return "", true
	case "full", "setup", "setup.sh":
		return build.VariantFull, true
	case "ci", "ci.sh":
		return build.VariantCI, true
	case "php", "php.sh":
		return build.VariantPHP, true
	case "go", "go.sh":
		return build.VariantGo, true
	case "agent", "agent.sh":
		return build.VariantAgent, true
	case "dev", "dev.sh":
		return build.VariantDev, true
	default:
		return "", false
	}
}

package buildcmd

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/pkg/build"
	buildinstallers "dappco.re/go/build/pkg/build/installers"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/build/pkg/release/publishers"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
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
			return runBuildInstallers(BuildInstallersRequest{
				Context:    cmdutil.ContextOrBackground(),
				Variant:    cmdutil.OptionString(opts, "variant"),
				Version:    cmdutil.OptionString(opts, "version"),
				OutputDir:  cmdutil.OptionString(opts, "output"),
				Repo:       cmdutil.OptionString(opts, "repo"),
				BinaryName: cmdutil.OptionString(opts, "name", "binary"),
			})
		},
	})
}

func runBuildInstallers(req BuildInstallersRequest) core.Result {
	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}

	projectDirResult := getInstallersWorkingDir()
	if !projectDirResult.OK {
		return core.Fail(coreerr.E("build.runBuildInstallers", "failed to get working directory", core.NewError(projectDirResult.Error())))
	}

	return runBuildInstallersInDir(ctx, projectDirResult.Value.(string), req.Variant, req.Version, req.OutputDir, req.Repo, req.BinaryName)
}

func runBuildInstallersInDir(ctx context.Context, projectDir, variant, version, outputDir, repo, binaryName string) core.Result {
	filesystem := io.Local

	buildConfigResult := loadInstallersBuildConfig(filesystem, projectDir)
	if !buildConfigResult.OK {
		return core.Fail(coreerr.E("build.runBuildInstallers", "failed to load build config", core.NewError(buildConfigResult.Error())))
	}
	buildConfig := buildConfigResult.Value.(*build.BuildConfig)

	installerVersion := core.Trim(version)
	if installerVersion == "" {
		versionResult := resolveInstallersVersion(ctx, projectDir)
		if !versionResult.OK {
			return core.Fail(coreerr.E("build.runBuildInstallers", "failed to determine installer version; use --version to override", core.NewError(versionResult.Error())))
		}
		installerVersion = versionResult.Value.(string)
	}
	validVersion := build.ValidateVersionIdentifier(installerVersion)
	if !validVersion.OK {
		return core.Fail(coreerr.E("build.runBuildInstallers", "invalid installer version; use a safe release identifier", core.NewError(validVersion.Error())))
	}

	installerRepo := core.Trim(repo)
	if installerRepo == "" {
		repoResult := resolveInstallersRepository(ctx, projectDir)
		if !repoResult.OK {
			return repoResult
		}
		installerRepo = repoResult.Value.(string)
	}

	if outputDir == "" {
		outputDir = ax.Join(projectDir, "dist", "installers")
	} else if !ax.IsAbs(outputDir) {
		outputDir = ax.Join(projectDir, outputDir)
	}

	created := filesystem.EnsureDir(outputDir)
	if !created.OK {
		return core.Fail(coreerr.E("build.runBuildInstallers", "failed to create output directory", core.NewError(created.Error())))
	}

	cfg := buildinstallers.InstallerConfig{
		Version:    installerVersion,
		Repo:       installerRepo,
		BinaryName: build.ResolveBuildName(projectDir, buildConfig, binaryName),
	}

	normalizedVariant, ok := normalizeInstallersVariant(variant)
	if !ok {
		return core.Fail(coreerr.E("build.runBuildInstallers", "unknown installer variant: "+core.Trim(variant), nil))
	}

	cli.Print("%s %s\n", buildHeaderStyle.Render("Installers"), "generating installer scripts")

	if normalizedVariant != "" {
		return writeInstallerVariant(filesystem, projectDir, outputDir, normalizedVariant, cfg)
	}

	for _, candidate := range build.InstallerVariants() {
		written := writeInstallerVariant(filesystem, projectDir, outputDir, candidate, cfg)
		if !written.OK {
			return written
		}
	}

	return core.Ok(nil)
}

func writeInstallerVariant(filesystem io.Medium, projectDir, outputDir string, variant build.InstallerVariant, cfg buildinstallers.InstallerConfig) core.Result {
	scriptName := build.InstallerOutputName(variant)
	if scriptName == "" {
		return core.Fail(coreerr.E("build.writeInstallerVariant", "unknown installer variant: "+string(variant), nil))
	}

	scriptResult := buildinstallers.GenerateInstaller(variant, cfg)
	if !scriptResult.OK {
		return core.Fail(coreerr.E("build.writeInstallerVariant", "failed to generate "+scriptName, core.NewError(scriptResult.Error())))
	}
	script := scriptResult.Value.(string)

	targetPath := ax.Join(outputDir, scriptName)
	written := filesystem.WriteMode(targetPath, script, 0o755)
	if !written.OK {
		return core.Fail(coreerr.E("build.writeInstallerVariant", "failed to write "+scriptName, core.NewError(written.Error())))
	}

	relPath := targetPath
	relPathResult := ax.Rel(projectDir, targetPath)
	if relPathResult.OK {
		relPath = relPathResult.Value.(string)
	}
	cli.Print("  %s\n", relPath)

	return core.Ok(nil)
}

func resolveInstallersRepository(ctx context.Context, projectDir string) core.Result {
	releaseConfigResult := loadInstallersReleaseConfig(projectDir)
	if !releaseConfigResult.OK {
		return core.Fail(coreerr.E("build.resolveInstallersRepository", "failed to load release config", core.NewError(releaseConfigResult.Error())))
	}
	releaseConfig := releaseConfigResult.Value.(*release.Config)

	if releaseConfig != nil {
		repo := core.Trim(releaseConfig.GetRepository())
		if repo != "" {
			return core.Ok(repo)
		}
	}

	repoResult := detectInstallersRepository(ctx, projectDir)
	if !repoResult.OK {
		return core.Fail(coreerr.E("build.resolveInstallersRepository", "failed to determine repository; use --repo or configure .core/release.yaml project.repository", core.NewError(repoResult.Error())))
	}

	return repoResult
}

func normalizeInstallersVariant(value string) (build.InstallerVariant, bool) {
	switch core.Lower(core.Trim(value)) {
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
	case "agent", "agentic", "agent.sh":
		return build.VariantAgent, true
	case "dev", "dev.sh":
		return build.VariantDev, true
	default:
		return "", false
	}
}

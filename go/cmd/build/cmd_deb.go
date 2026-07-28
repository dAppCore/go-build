package buildcmd

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/packages"
	storage "dappco.re/go/build/pkg/storage"
)

var (
	getDebWorkingDir   = ax.Getwd
	loadDebBuildConfig = build.LoadConfig
	resolveDebVersion  = resolveBuildVersion
	buildDebPackage    = packages.BuildDeb
)

// BuildDebRequest groups the inputs for `core build deb`.
type BuildDebRequest struct {
	Context      context.Context
	Binary       string
	Name         string
	Version      string
	Architecture string
	Maintainer   string
	Homepage     string
	Description  string
	Depends      string
	InstallPath  string
	OutputDir    string
}

// AddDebCommand registers the Debian package command.
func AddDebCommand(c *core.Core) core.Result {
	return c.Command("build/deb", core.Command{
		Description: "Build a Debian package (.deb) from a built binary",
		Action: func(opts core.Options) core.Result {
			return runBuildDeb(BuildDebRequest{
				Context:      cmdutil.ContextOrBackground(),
				Binary:       cmdutil.OptionString(opts, "binary", "_arg"),
				Name:         cmdutil.OptionString(opts, "name"),
				Version:      cmdutil.OptionString(opts, "version"),
				Architecture: cmdutil.OptionString(opts, "arch", "architecture"),
				Maintainer:   cmdutil.OptionString(opts, "maintainer"),
				Homepage:     cmdutil.OptionString(opts, "homepage"),
				Description:  cmdutil.OptionString(opts, "description"),
				Depends:      cmdutil.OptionString(opts, "depends"),
				InstallPath:  cmdutil.OptionString(opts, "install-path", "install_path"),
				OutputDir:    cmdutil.OptionString(opts, "output"),
			})
		},
	})
}

func runBuildDeb(req BuildDebRequest) core.Result {
	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}

	projectDirResult := getDebWorkingDir()
	if !projectDirResult.OK {
		return core.Fail(core.E("build.runBuildDeb", "failed to get working directory", core.NewError(projectDirResult.Error())))
	}

	return runBuildDebInDir(ctx, projectDirResult.Value.(string), req)
}

func runBuildDebInDir(ctx context.Context, projectDir string, req BuildDebRequest) core.Result {
	filesystem := storage.Local

	binary := core.Trim(req.Binary)
	if binary == "" {
		return core.Fail(core.E("build.runBuildDeb", "no binary given; pass the built binary as an argument or with --binary", nil))
	}
	if !ax.IsAbs(binary) {
		binary = ax.Join(projectDir, binary)
	}
	if !filesystem.IsFile(binary) {
		return core.Fail(core.E("build.runBuildDeb", "binary not found: "+binary, nil))
	}

	buildConfigResult := loadDebBuildConfig(filesystem, projectDir)
	if !buildConfigResult.OK {
		return core.Fail(core.E("build.runBuildDeb", "failed to load build config", core.NewError(buildConfigResult.Error())))
	}
	buildConfig := buildConfigResult.Value.(*build.BuildConfig)

	name := build.ResolveBuildName(projectDir, buildConfig, req.Name)

	version := core.Trim(req.Version)
	if version == "" {
		versionResult := resolveDebVersion(ctx, projectDir)
		if !versionResult.OK {
			return core.Fail(core.E("build.runBuildDeb", "failed to determine version; use --version to override", core.NewError(versionResult.Error())))
		}
		version = versionResult.Value.(string)
	}

	// Without an architecture the package would claim the host's, which is
	// wrong for every cross-compiled binary — the common case in a release.
	architecture := core.Trim(req.Architecture)
	if architecture == "" {
		architecture = archFromBinaryName(binary)
	}
	if architecture == "" {
		return core.Fail(core.E("build.runBuildDeb", "cannot infer architecture from "+core.PathBase(binary)+"; pass --arch", nil))
	}

	outputDir := core.Trim(req.OutputDir)
	if outputDir == "" {
		outputDir = ax.Join(projectDir, "dist")
	} else if !ax.IsAbs(outputDir) {
		outputDir = ax.Join(projectDir, outputDir)
	}

	outputPath := ax.Join(outputDir, packages.DebFileName(name, version, architecture))

	cli.Print("%s %s\n", buildHeaderStyle.Render("Deb"), "packaging "+name+" "+version+" ("+packages.DebArchitecture(architecture)+")")

	result := buildDebPackage(filesystem, packages.DebSpec{
		Name:         name,
		Version:      version,
		Architecture: architecture,
		Maintainer:   core.Trim(req.Maintainer),
		Homepage:     core.Trim(req.Homepage),
		Description:  core.Trim(req.Description),
		Depends:      core.Trim(req.Depends),
		BinaryPath:   binary,
		InstallPath:  core.Trim(req.InstallPath),
	}, outputPath)
	if !result.OK {
		return result
	}

	relPath := outputPath
	if rel := ax.Rel(projectDir, outputPath); rel.OK {
		relPath = rel.Value.(string)
	}
	cli.Print("  %s\n", relPath)

	return result
}

// archFromBinaryName reads the architecture out of a release asset name such as
// core-linux-amd64 or core_linux_arm64, which is how the build action names the
// binaries it produces.
func archFromBinaryName(path string) string {
	base := core.PathBase(path)
	base = core.TrimSuffix(base, ".exe")

	for _, candidate := range []string{"amd64", "arm64", "386", "arm", "ppc64le", "s390x", "riscv64", "mips64le"} {
		if core.HasSuffix(base, "-"+candidate) || core.HasSuffix(base, "_"+candidate) {
			return candidate
		}
	}
	return ""
}

package buildcmd

import (
	"context"
	"regexp"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

var buildAppleFn = build.BuildApple

type appleCLIOptions struct {
	Arch              string
	ArchChanged       bool
	Sign              bool
	SignChanged       bool
	Notarise          bool
	NotariseChanged   bool
	DMG               bool
	DMGChanged        bool
	TestFlight        bool
	TestFlightChanged bool
	AppStore          bool
	AppStoreChanged   bool
	TeamID            string
	TeamIDChanged     bool
	BundleID          string
	BundleIDChanged   bool
	Version           string
	BuildNumber       string
	ConfigPath        string
	OutputDir         string
}

// AddAppleCommand adds the Apple build subcommand to the build command.
func AddAppleCommand(c *core.Core) {
	c.Command("build/apple", core.Command{
		Description: "cmd.build.apple.long",
		Action: func(opts core.Options) core.Result {
			return runAppleBuild(cmdutil.ContextOrBackground(), appleCLIOptions{
				Arch:              cmdutil.OptionString(opts, "arch"),
				ArchChanged:       opts.Has("arch"),
				Sign:              cmdutil.OptionBoolDefault(opts, true, "sign"),
				SignChanged:       opts.Has("sign"),
				Notarise:          cmdutil.OptionBoolDefault(opts, true, "notarise"),
				NotariseChanged:   opts.Has("notarise"),
				DMG:               cmdutil.OptionBool(opts, "dmg"),
				DMGChanged:        opts.Has("dmg"),
				TestFlight:        cmdutil.OptionBool(opts, "testflight"),
				TestFlightChanged: opts.Has("testflight"),
				AppStore:          cmdutil.OptionBool(opts, "appstore"),
				AppStoreChanged:   opts.Has("appstore"),
				TeamID:            cmdutil.OptionString(opts, "team-id"),
				TeamIDChanged:     opts.Has("team-id"),
				BundleID:          cmdutil.OptionString(opts, "bundle-id"),
				BundleIDChanged:   opts.Has("bundle-id"),
				Version:           cmdutil.OptionString(opts, "version"),
				BuildNumber:       cmdutil.OptionString(opts, "build-number"),
				ConfigPath:        cmdutil.OptionString(opts, "config"),
				OutputDir:         cmdutil.OptionString(opts, "output"),
			})
		},
	})
}

func runAppleBuild(ctx context.Context, opts appleCLIOptions) core.Result {
	projectDirResult := ax.Getwd()
	if !projectDirResult.OK {
		return core.Fail(core.E("build.apple", "failed to get working directory", core.NewError(projectDirResult.Error())))
	}
	return runAppleBuildInDir(ctx, projectDirResult.Value.(string), opts)
}

func runAppleBuildInDir(ctx context.Context, projectDir string, opts appleCLIOptions) core.Result {
	if ctx == nil {
		ctx = context.Background()
	}

	filesystem := storage.Local

	buildConfigResult := loadAppleBuildConfig(filesystem, projectDir, opts.ConfigPath)
	if !buildConfigResult.OK {
		return buildConfigResult
	}
	buildConfig := buildConfigResult.Value.(*build.BuildConfig)
	cacheSetup := build.SetupBuildCache(filesystem, projectDir, buildConfig)
	if !cacheSetup.OK {
		return core.Fail(core.E("build.apple", "failed to set up build cache", core.NewError(cacheSetup.Error())))
	}
	if build.HasXcodeCloudConfig(buildConfig) {
		written := build.WriteXcodeCloudScripts(filesystem, projectDir, buildConfig)
		if !written.OK {
			return core.Fail(core.E("build.apple", "failed to write Xcode Cloud scripts", core.NewError(written.Error())))
		}
	}

	version := opts.Version
	if version == "" {
		versionResult := resolveBuildVersion(ctx, projectDir)
		if !versionResult.OK {
			return core.Fail(core.E("build.apple", "failed to determine version", core.NewError(versionResult.Error())))
		}
		version = versionResult.Value.(string)
	}
	validVersion := build.ValidateVersionIdentifier(version)
	if !validVersion.OK {
		return core.Fail(core.E("build.apple", "invalid build version; use a safe release identifier", core.NewError(validVersion.Error())))
	}

	buildNumber := opts.BuildNumber
	if buildNumber != "" {
		validBuildNumber := validateAppleBuildNumber(buildNumber)
		if !validBuildNumber.OK {
			return validBuildNumber
		}
	} else {
		buildNumberResult := resolveAppleBuildNumber(ctx, projectDir)
		if !buildNumberResult.OK {
			return buildNumberResult
		}
		buildNumber = buildNumberResult.Value.(string)
	}

	appleOptions := resolveAppleCommandOptions(buildConfig, opts)

	name := buildConfig.Project.Binary
	if name == "" {
		name = buildConfig.Project.Name
	}
	if name == "" {
		name = ax.Base(projectDir)
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(projectDir, "dist", "apple")
	} else if !ax.IsAbs(outputDir) {
		outputDir = ax.Join(projectDir, outputDir)
	}

	runtimeCfg := buildRuntimeConfig(filesystem, projectDir, outputDir, name, buildConfig, false, "", version)
	resultValue := buildAppleFn(ctx, runtimeCfg, appleOptions, buildNumber)
	if !resultValue.OK {
		return resultValue
	}
	result := resultValue.Value.(*build.AppleBuildResult)

	cli.Print("%s %s\n", buildSuccessStyle.Render("Success"), "Apple build completed")
	cli.Print("  %s %s\n", "bundle", buildTargetStyle.Render(result.BundlePath))
	cli.Print("  %s %s\n", "version", buildTargetStyle.Render(result.Version))
	cli.Print("  %s %s\n", "build number", buildTargetStyle.Render(result.BuildNumber))
	if result.DMGPath != "" {
		cli.Print("  %s %s\n", "dmg", buildTargetStyle.Render(result.DMGPath))
	}

	return core.Ok(nil)
}

func loadAppleBuildConfig(filesystem storage.Medium, projectDir, configPath string) core.Result {
	if configPath == "" {
		cfg := build.LoadConfig(filesystem, projectDir)
		if !cfg.OK {
			return core.Fail(core.E("build.apple", "failed to load config", core.NewError(cfg.Error())))
		}
		return cfg
	}

	if !ax.IsAbs(configPath) {
		configPath = ax.Join(projectDir, configPath)
	}
	if !filesystem.Exists(configPath) {
		return core.Fail(core.E("build.apple", "build config not found: "+configPath, nil))
	}

	cfg := build.LoadConfigAtPath(filesystem, configPath)
	if !cfg.OK {
		return core.Fail(core.E("build.apple", "failed to load config", core.NewError(cfg.Error())))
	}
	return cfg
}

func resolveAppleCommandOptions(cfg *build.BuildConfig, overrides appleCLIOptions) build.AppleOptions {
	var options build.AppleOptions
	if cfg != nil {
		options = cfg.Apple.Resolve()
		options.CertIdentity = firstNonEmptyString(options.CertIdentity, cfg.Sign.MacOS.Identity)
		options.TeamID = firstNonEmptyString(options.TeamID, cfg.Sign.MacOS.TeamID)
		options.AppleID = firstNonEmptyString(options.AppleID, cfg.Sign.MacOS.AppleID)
		options.Password = firstNonEmptyString(options.Password, cfg.Sign.MacOS.AppPassword)
	} else {
		options = build.DefaultAppleOptions()
	}

	if overrides.ArchChanged {
		options.Arch = overrides.Arch
	}
	if overrides.SignChanged {
		options.Sign = overrides.Sign
	}
	if overrides.NotariseChanged {
		options.Notarise = overrides.Notarise
	}
	if overrides.DMGChanged {
		options.DMG = overrides.DMG
	}
	if overrides.TestFlightChanged {
		options.TestFlight = overrides.TestFlight
	}
	if overrides.AppStoreChanged {
		options.AppStore = overrides.AppStore
	}
	if overrides.TeamIDChanged {
		options.TeamID = overrides.TeamID
	}
	if overrides.BundleIDChanged {
		options.BundleID = overrides.BundleID
	}

	return options
}

func resolveAppleBuildNumber(ctx context.Context, projectDir string) core.Result {
	if value := core.Trim(core.Env("GITHUB_RUN_NUMBER")); value != "" {
		if validated := validateAppleBuildNumber(value); validated.OK {
			return core.Ok(value)
		}
	}

	outputResult := ax.RunDir(ctx, projectDir, "git", "rev-list", "--count", "HEAD")
	if !outputResult.OK {
		return core.Ok("1")
	}

	buildNumber := core.Trim(outputResult.Value.(string))
	if buildNumber == "" {
		return core.Ok("1")
	}
	validated := validateAppleBuildNumber(buildNumber)
	if !validated.OK {
		return validated
	}
	return core.Ok(buildNumber)
}

var appleBuildNumberPattern = regexp.MustCompile(`^[0-9]+$`)

func validateAppleBuildNumber(value string) core.Result {
	if !appleBuildNumberPattern.MatchString(value) {
		return core.Fail(core.E("build.apple", "build-number must be a positive integer", nil))
	}
	return core.Ok(nil)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if core.Trim(value) != "" {
			return value
		}
	}
	return ""
}

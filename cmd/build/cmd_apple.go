package buildcmd

import (
	"context"
	"regexp"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/internal/cmdutil"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
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
			return cmdutil.ResultFromError(runAppleBuild(cmdutil.ContextOrBackground(), appleCLIOptions{
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
			}))
		},
	})
}

func runAppleBuild(ctx context.Context, opts appleCLIOptions) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.apple", "failed to get working directory", err)
	}
	return runAppleBuildInDir(ctx, projectDir, opts)
}

func runAppleBuildInDir(ctx context.Context, projectDir string, opts appleCLIOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}

	filesystem := io.Local

	buildConfig, err := loadAppleBuildConfig(filesystem, projectDir, opts.ConfigPath)
	if err != nil {
		return err
	}
	if err := build.SetupBuildCache(filesystem, projectDir, buildConfig); err != nil {
		return coreerr.E("build.apple", "failed to set up build cache", err)
	}

	version := opts.Version
	if version == "" {
		version, err = resolveBuildVersion(ctx, projectDir)
		if err != nil {
			return coreerr.E("build.apple", "failed to determine version", err)
		}
	}

	buildNumber := opts.BuildNumber
	if buildNumber != "" {
		if err := validateAppleBuildNumber(buildNumber); err != nil {
			return err
		}
	} else {
		buildNumber, err = resolveAppleBuildNumber(ctx, projectDir)
		if err != nil {
			return err
		}
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
	result, err := buildAppleFn(ctx, runtimeCfg, appleOptions, buildNumber)
	if err != nil {
		return err
	}

	cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("common.label.success")), i18n.T("cmd.build.apple.completed"))
	cli.Print("  %s %s\n", i18n.T("cmd.build.apple.label.bundle"), buildTargetStyle.Render(result.BundlePath))
	cli.Print("  %s %s\n", i18n.T("cmd.build.apple.label.version"), buildTargetStyle.Render(result.Version))
	cli.Print("  %s %s\n", i18n.T("cmd.build.apple.label.build_number"), buildTargetStyle.Render(result.BuildNumber))
	if result.DMGPath != "" {
		cli.Print("  %s %s\n", i18n.T("cmd.build.apple.label.dmg"), buildTargetStyle.Render(result.DMGPath))
	}

	return nil
}

func loadAppleBuildConfig(filesystem io.Medium, projectDir, configPath string) (*build.BuildConfig, error) {
	if configPath == "" {
		cfg, err := build.LoadConfig(filesystem, projectDir)
		if err != nil {
			return nil, coreerr.E("build.apple", "failed to load config", err)
		}
		return cfg, nil
	}

	if !ax.IsAbs(configPath) {
		configPath = ax.Join(projectDir, configPath)
	}
	if !filesystem.Exists(configPath) {
		return nil, coreerr.E("build.apple", "build config not found: "+configPath, nil)
	}

	cfg, err := build.LoadConfigAtPath(filesystem, configPath)
	if err != nil {
		return nil, coreerr.E("build.apple", "failed to load config", err)
	}
	return cfg, nil
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

func resolveAppleBuildNumber(ctx context.Context, projectDir string) (string, error) {
	if value := core.Trim(core.Env("GITHUB_RUN_NUMBER")); value != "" {
		if err := validateAppleBuildNumber(value); err == nil {
			return value, nil
		}
	}

	output, err := ax.RunDir(ctx, projectDir, "git", "rev-list", "--count", "HEAD")
	if err != nil {
		return "1", nil
	}

	buildNumber := core.Trim(output)
	if buildNumber == "" {
		return "1", nil
	}
	if err := validateAppleBuildNumber(buildNumber); err != nil {
		return "", err
	}
	return buildNumber, nil
}

var appleBuildNumberPattern = regexp.MustCompile(`^[0-9]+$`)

func validateAppleBuildNumber(value string) error {
	if !appleBuildNumberPattern.MatchString(value) {
		return coreerr.E("build.apple", "build-number must be a positive integer", nil)
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if core.Trim(value) != "" {
			return value
		}
	}
	return ""
}

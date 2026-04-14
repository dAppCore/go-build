package buildcmd

import (
	"context"
	"regexp"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

var (
	appleArch        string
	appleSign        bool
	appleNotarise    bool
	appleDMG         bool
	appleTestFlight  bool
	appleAppStore    bool
	appleTeamID      string
	appleBundleID    string
	appleVersion     string
	appleBuildNumber string
	appleConfigPath  string
	appleOutputDir   string
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

var appleCmd = &cli.Command{
	Use: "apple",
	RunE: func(cmd *cli.Command, args []string) error {
		return runAppleBuild(cmd.Context(), appleCLIOptions{
			Arch:              appleArch,
			ArchChanged:       cmd.Flags().Changed("arch"),
			Sign:              appleSign,
			SignChanged:       cmd.Flags().Changed("sign"),
			Notarise:          appleNotarise,
			NotariseChanged:   cmd.Flags().Changed("notarise"),
			DMG:               appleDMG,
			DMGChanged:        cmd.Flags().Changed("dmg"),
			TestFlight:        appleTestFlight,
			TestFlightChanged: cmd.Flags().Changed("testflight"),
			AppStore:          appleAppStore,
			AppStoreChanged:   cmd.Flags().Changed("appstore"),
			TeamID:            appleTeamID,
			TeamIDChanged:     cmd.Flags().Changed("team-id"),
			BundleID:          appleBundleID,
			BundleIDChanged:   cmd.Flags().Changed("bundle-id"),
			Version:           appleVersion,
			BuildNumber:       appleBuildNumber,
			ConfigPath:        appleConfigPath,
			OutputDir:         appleOutputDir,
		})
	},
}

func setAppleI18n() {
	appleCmd.Short = i18n.T("cmd.build.apple.short")
	appleCmd.Long = i18n.T("cmd.build.apple.long")
}

func initAppleFlags() {
	appleCmd.Flags().StringVar(&appleArch, "arch", "universal", i18n.T("cmd.build.apple.flag.arch"))
	appleCmd.Flags().BoolVar(&appleSign, "sign", true, i18n.T("cmd.build.apple.flag.sign"))
	appleCmd.Flags().BoolVar(&appleNotarise, "notarise", true, i18n.T("cmd.build.apple.flag.notarise"))
	appleCmd.Flags().BoolVar(&appleDMG, "dmg", false, i18n.T("cmd.build.apple.flag.dmg"))
	appleCmd.Flags().BoolVar(&appleTestFlight, "testflight", false, i18n.T("cmd.build.apple.flag.testflight"))
	appleCmd.Flags().BoolVar(&appleAppStore, "appstore", false, i18n.T("cmd.build.apple.flag.appstore"))
	appleCmd.Flags().StringVar(&appleTeamID, "team-id", "", i18n.T("cmd.build.apple.flag.team_id"))
	appleCmd.Flags().StringVar(&appleBundleID, "bundle-id", "", i18n.T("cmd.build.apple.flag.bundle_id"))
	appleCmd.Flags().StringVar(&appleVersion, "version", "", i18n.T("cmd.build.apple.flag.version"))
	appleCmd.Flags().StringVar(&appleBuildNumber, "build-number", "", i18n.T("cmd.build.apple.flag.build_number"))
	appleCmd.Flags().StringVar(&appleConfigPath, "config", "", i18n.T("cmd.build.flag.config"))
	appleCmd.Flags().StringVar(&appleOutputDir, "output", "", i18n.T("cmd.build.flag.output"))
}

// AddAppleCommand adds the Apple build subcommand to the build command.
func AddAppleCommand(buildCmd *cli.Command) {
	setAppleI18n()
	initAppleFlags()
	buildCmd.AddCommand(appleCmd)
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

// Package buildcmd provides project build commands with auto-detection.
package buildcmd

import (
	"embed"

	"forge.lthn.ai/core/cli/pkg/cli"
	_ "forge.lthn.ai/core/go-build/locales" // registers locale translations
	"dappco.re/go/core/i18n"
)

func init() {
	cli.RegisterCommands(AddBuildCommands)
}

// Style aliases from shared package
var (
	buildHeaderStyle  = cli.TitleStyle
	buildTargetStyle  = cli.ValueStyle
	buildSuccessStyle = cli.SuccessStyle
	buildErrorStyle   = cli.ErrorStyle
	buildDimStyle     = cli.DimStyle
)

//go:embed all:tmpl/gui
var guiTemplate embed.FS

// Flags for the main build command
var (
	buildType  string
	ciMode     bool
	targets    string
	outputDir  string
	doArchive  bool
	doChecksum bool
	verbose    bool

	// Docker/LinuxKit specific flags
	configPath string
	format     string
	push       bool
	imageName  string

	// Signing flags
	noSign   bool
	notarize bool

	// from-path subcommand
	fromPath string

	// pwa subcommand
	pwaURL string

	// sdk subcommand
	sdkSpec    string
	sdkLang    string
	sdkVersion string
	sdkDryRun  bool
)

var buildCmd = &cli.Command{
	Use: "build",
	RunE: func(cmd *cli.Command, args []string) error {
		return runProjectBuild(cmd.Context(), buildType, ciMode, targets, outputDir, doArchive, doChecksum, configPath, format, push, imageName, noSign, notarize, verbose)
	},
}

var fromPathCmd = &cli.Command{
	Use: "from-path",
	RunE: func(cmd *cli.Command, args []string) error {
		if fromPath == "" {
			return errPathRequired
		}
		return runBuild(fromPath)
	},
}

var pwaCmd = &cli.Command{
	Use: "pwa",
	RunE: func(cmd *cli.Command, args []string) error {
		if pwaURL == "" {
			return errURLRequired
		}
		return runPwaBuild(pwaURL)
	},
}

var sdkBuildCmd = &cli.Command{
	Use: "sdk",
	RunE: func(cmd *cli.Command, args []string) error {
		return runBuildSDK(sdkSpec, sdkLang, sdkVersion, sdkDryRun)
	},
}

func setBuildI18n() {
	buildCmd.Short = i18n.T("cmd.build.short")
	buildCmd.Long = i18n.T("cmd.build.long")
	fromPathCmd.Short = i18n.T("cmd.build.from_path.short")
	pwaCmd.Short = i18n.T("cmd.build.pwa.short")
	sdkBuildCmd.Short = i18n.T("cmd.build.sdk.short")
	sdkBuildCmd.Long = i18n.T("cmd.build.sdk.long")
}

func initBuildFlags() {
	// Main build command flags
	buildCmd.Flags().StringVar(&buildType, "type", "", i18n.T("cmd.build.flag.type"))
	buildCmd.Flags().BoolVar(&ciMode, "ci", false, i18n.T("cmd.build.flag.ci"))
	buildCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, i18n.T("common.flag.verbose"))
	buildCmd.Flags().StringVar(&targets, "targets", "", i18n.T("cmd.build.flag.targets"))
	buildCmd.Flags().StringVar(&outputDir, "output", "", i18n.T("cmd.build.flag.output"))
	buildCmd.Flags().BoolVar(&doArchive, "archive", true, i18n.T("cmd.build.flag.archive"))
	buildCmd.Flags().BoolVar(&doChecksum, "checksum", true, i18n.T("cmd.build.flag.checksum"))

	// Docker/LinuxKit specific
	buildCmd.Flags().StringVar(&configPath, "config", "", i18n.T("cmd.build.flag.config"))
	buildCmd.Flags().StringVar(&format, "format", "", i18n.T("cmd.build.flag.format"))
	buildCmd.Flags().BoolVar(&push, "push", false, i18n.T("cmd.build.flag.push"))
	buildCmd.Flags().StringVar(&imageName, "image", "", i18n.T("cmd.build.flag.image"))

	// Signing flags
	buildCmd.Flags().BoolVar(&noSign, "no-sign", false, i18n.T("cmd.build.flag.no_sign"))
	buildCmd.Flags().BoolVar(&notarize, "notarize", false, i18n.T("cmd.build.flag.notarize"))

	// from-path subcommand flags
	fromPathCmd.Flags().StringVar(&fromPath, "path", "", i18n.T("cmd.build.from_path.flag.path"))

	// pwa subcommand flags
	pwaCmd.Flags().StringVar(&pwaURL, "url", "", i18n.T("cmd.build.pwa.flag.url"))

	// sdk subcommand flags
	sdkBuildCmd.Flags().StringVar(&sdkSpec, "spec", "", i18n.T("common.flag.spec"))
	sdkBuildCmd.Flags().StringVar(&sdkLang, "lang", "", i18n.T("cmd.build.sdk.flag.lang"))
	sdkBuildCmd.Flags().StringVar(&sdkVersion, "version", "", i18n.T("cmd.build.sdk.flag.version"))
	sdkBuildCmd.Flags().BoolVar(&sdkDryRun, "dry-run", false, i18n.T("cmd.build.sdk.flag.dry_run"))

	// Add subcommands
	buildCmd.AddCommand(fromPathCmd)
	buildCmd.AddCommand(pwaCmd)
	buildCmd.AddCommand(sdkBuildCmd)
}

// AddBuildCommands registers the 'build' command and all subcommands.
func AddBuildCommands(root *cli.Command) {
	setBuildI18n()
	initBuildFlags()
	AddReleaseCommand(buildCmd)
	root.AddCommand(buildCmd)
}

// cmd_release.go implements the release command: build + archive + publish in one step.

package buildcmd

import (
	"context"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/release"
	"dappco.re/go/core/i18n"
	coreerr "dappco.re/go/core/log"
	"dappco.re/go/core/cli/pkg/cli"
)

// Flag variables for release command.
var (
	releaseVersion       string
	releaseDraft         bool
	releasePrerelease    bool
	releaseLaunchMode    bool
	releaseArchiveFormat string
)

var releaseCmd = &cli.Command{
	Use: "release",
	RunE: func(cmd *cli.Command, args []string) error {
		return runRelease(cmd.Context(), !releaseLaunchMode, releaseVersion, releaseDraft, releasePrerelease, releaseArchiveFormat)
	},
}

func setReleaseI18n() {
	releaseCmd.Short = i18n.T("cmd.build.release.short")
	releaseCmd.Long = i18n.T("cmd.build.release.long")
}

func initReleaseFlags() {
	releaseCmd.Flags().BoolVar(&releaseLaunchMode, "we-are-go-for-launch", false, i18n.T("cmd.build.release.flag.go_for_launch"))
	releaseCmd.Flags().StringVar(&releaseVersion, "version", "", i18n.T("cmd.build.release.flag.version"))
	releaseCmd.Flags().BoolVar(&releaseDraft, "draft", false, i18n.T("cmd.build.release.flag.draft"))
	releaseCmd.Flags().BoolVar(&releasePrerelease, "prerelease", false, i18n.T("cmd.build.release.flag.prerelease"))
	releaseCmd.Flags().StringVar(&releaseArchiveFormat, "archive-format", "", i18n.T("cmd.build.flag.archive_format"))
}

// AddReleaseCommand adds the release subcommand to the build command.
//
// buildcmd.AddReleaseCommand(buildCmd)
func AddReleaseCommand(buildCmd *cli.Command) {
	setReleaseI18n()
	initReleaseFlags()
	buildCmd.AddCommand(releaseCmd)
}

// runRelease executes the full release workflow: build + archive + checksum + publish.
//
// runRelease(ctx, true, "v1.2.3", true, false, "xz") // dry run with a forced release version, draft output
func runRelease(ctx context.Context, dryRun bool, version string, draft, prerelease bool, archiveFormat string) error {
	// Get current directory
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("release", "get working directory", err)
	}

	// Check for release config
	if !release.ConfigExists(projectDir) {
		cli.Print("%s %s\n",
			buildErrorStyle.Render(i18n.Label("error")),
			i18n.T("cmd.build.release.error.no_config"),
		)
		cli.Print("  %s\n", buildDimStyle.Render(i18n.T("cmd.build.release.hint.create_config")))
		return coreerr.E("release", "config not found", nil)
	}

	// Load configuration
	cfg, err := release.LoadConfig(projectDir)
	if err != nil {
		return coreerr.E("release", "load config", err)
	}

	// Apply CLI overrides
	if version != "" {
		cfg.SetVersion(version)
	}
	if err := applyReleaseArchiveFormatOverride(cfg, archiveFormat); err != nil {
		return err
	}

	// Apply draft/prerelease overrides to all publishers
	if draft || prerelease {
		for i := range cfg.Publishers {
			if draft {
				cfg.Publishers[i].Draft = true
			}
			if prerelease {
				cfg.Publishers[i].Prerelease = true
			}
		}
	}

	// Print header
	cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.release.label.release")), i18n.T("cmd.build.release.building_and_publishing"))
	if dryRun {
		cli.Print("  %s\n", buildDimStyle.Render(i18n.T("cmd.build.release.dry_run_hint")))
	}
	cli.Blank()

	// Run full release (build + archive + checksum + publish)
	rel, err := release.Run(ctx, cfg, dryRun)
	if err != nil {
		return err
	}

	// Print summary
	cli.Blank()
	cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("i18n.done.pass")), i18n.T("cmd.build.release.completed"))
	cli.Print("  %s   %s\n", i18n.Label("version"), buildTargetStyle.Render(rel.Version))
	cli.Print("  %s %d\n", i18n.T("cmd.build.release.label.artifacts"), len(rel.Artifacts))

	if !dryRun {
		for _, pub := range cfg.Publishers {
			cli.Print("  %s %s\n", i18n.T("cmd.build.release.label.published"), buildTargetStyle.Render(pub.Type))
		}
	}

	return nil
}

// applyReleaseArchiveFormatOverride applies the archive-format CLI override to the release config.
//
// applyReleaseArchiveFormatOverride(cfg, "xz") // cfg.Build.ArchiveFormat = "xz"
func applyReleaseArchiveFormatOverride(cfg *release.Config, archiveFormat string) error {
	if cfg == nil || archiveFormat == "" {
		return nil
	}

	formatValue, err := resolveArchiveFormat("", archiveFormat)
	if err != nil {
		return err
	}

	cfg.Build.ArchiveFormat = string(formatValue)
	return nil
}

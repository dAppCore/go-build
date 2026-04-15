// cmd_release.go implements the release command: build + archive + publish in one step.

package buildcmd

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/i18n"
	coreerr "dappco.re/go/core/log"
)

var (
	getReleaseWorkingDir  = ax.Getwd
	releaseConfigExistsFn = release.ConfigExists
	loadReleaseConfigFn   = release.LoadConfig
	runFullReleaseFn      = release.Run
	runSDKReleaseFn       = release.RunSDK
)

// AddReleaseCommand adds the release subcommand to the build command.
//
// buildcmd.AddReleaseCommand(buildCmd)
func AddReleaseCommand(c *core.Core) {
	registerReleaseCommand(c, "build/release")
	registerReleaseCommand(c, "release")
}

func registerReleaseCommand(c *core.Core, path string) {
	c.Command(path, core.Command{
		Description: "cmd.build.release.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runRelease(
				cmdutil.ContextOrBackground(),
				resolveReleaseDryRun(
					cmdutil.OptionBool(opts, "dry-run"),
					cmdutil.OptionBool(opts, "publish"),
					cmdutil.OptionBool(opts, "we-are-go-for-launch"),
				),
				cmdutil.OptionString(opts, "target"),
				cmdutil.OptionString(opts, "version", "tag"),
				cmdutil.OptionBool(opts, "draft"),
				cmdutil.OptionBool(opts, "prerelease"),
				cmdutil.OptionString(opts, "archive-format"),
			))
		},
	})
}

// runRelease executes the full release workflow: build + archive + checksum + publish.
//
// runRelease(ctx, true, "sdk", "v1.2.3", true, false, "xz") // dry run with an SDK-only target
func runRelease(ctx context.Context, dryRun bool, target, version string, draft, prerelease bool, archiveFormat string) error {
	// Get current directory
	projectDir, err := getReleaseWorkingDir()
	if err != nil {
		return coreerr.E("release", "get working directory", err)
	}

	// Check for release config
	if !releaseConfigExistsFn(projectDir) {
		cli.Print("%s %s\n",
			buildErrorStyle.Render(i18n.Label("error")),
			i18n.T("cmd.build.release.error.no_config"),
		)
		cli.Print("  %s\n", buildDimStyle.Render(i18n.T("cmd.build.release.hint.create_config")))
		return coreerr.E("release", "config not found", nil)
	}

	// Load configuration
	cfg, err := loadReleaseConfigFn(projectDir)
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

	target = core.Lower(core.Trim(target))
	if target == "" {
		target = "release"
	}

	// Apply draft/prerelease overrides to all publishers
	if target == "release" && (draft || prerelease) {
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
	cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.release.label.release")), releaseTargetLabel(target))
	if dryRun {
		cli.Print("  %s\n", buildDimStyle.Render(i18n.T("cmd.build.release.dry_run_hint")))
	}
	cli.Blank()

	switch target {
	case "release":
		rel, err := runFullReleaseFn(ctx, cfg, dryRun)
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
	case "sdk":
		result, err := runSDKReleaseFn(ctx, cfg, dryRun)
		if err != nil {
			return err
		}

		cli.Blank()
		cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("i18n.done.pass")), "SDK release completed")
		cli.Print("  %s   %s\n", i18n.Label("version"), buildTargetStyle.Render(result.Version))
		cli.Print("  %s   %s\n", "output", buildTargetStyle.Render(result.Output))
		cli.Print("  %s %s\n", "languages", buildTargetStyle.Render(core.Join(", ", result.Languages...)))
		return nil
	default:
		return coreerr.E("release", "unsupported release target: "+target, nil)
	}
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

func resolveReleaseDryRun(dryRun, publish, weAreGoForLaunch bool) bool {
	if publish || weAreGoForLaunch {
		return false
	}
	return dryRun
}

func releaseTargetLabel(target string) string {
	if target == "sdk" {
		return "Generating SDK release"
	}
	return i18n.T("cmd.build.release.building_and_publishing")
}

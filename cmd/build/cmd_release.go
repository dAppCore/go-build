// cmd_release.go implements the release command: build + archive + publish in one step.

package buildcmd

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/i18n"
	coreerr "dappco.re/go/log"
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
			return runRelease(
				cmdutil.ContextOrBackground(),
				resolveReleaseDryRun(
					cmdutil.OptionBool(opts, "dry-run"),
					cmdutil.OptionBool(opts, "publish"),
					cmdutil.OptionBool(opts, "we-are-go-for-launch"),
				),
				cmdutil.OptionBool(opts, "ci"),
				cmdutil.OptionString(opts, "target"),
				cmdutil.OptionString(opts, "version", "tag"),
				cmdutil.OptionBool(opts, "draft"),
				cmdutil.OptionBool(opts, "prerelease"),
				cmdutil.OptionString(opts, "archive-format"),
				cmdutil.OptionBool(opts, "apple-testflight", "apple_testflight", "testflight"),
			)
		},
	})
}

// runRelease executes the full release workflow: build + archive + checksum + publish.
//
// runRelease(ctx, true, false, "sdk", "v1.2.3", true, false, "xz") // dry run with an SDK-only target
func runRelease(ctx context.Context, dryRun bool, ciMode bool, target, version string, draft, prerelease bool, archiveFormat string, appleTestFlightFlag ...bool) (result core.Result) {
	if ciMode {
		defer func() {
			emitCIErrorAnnotation(result)
		}()
	}

	// Get current directory
	projectDirResult := getReleaseWorkingDir()
	if !projectDirResult.OK {
		return core.Fail(coreerr.E("release", "get working directory", core.NewError(projectDirResult.Error())))
	}
	projectDir := projectDirResult.Value.(string)

	target = core.Lower(core.Trim(target))
	if releaseAppleTestFlightRequested(target, appleTestFlightFlag...) {
		return runAppleBuildInDir(ctx, projectDir, appleCLIOptions{
			Version:           version,
			TestFlight:        true,
			TestFlightChanged: true,
		})
	}
	if target == "" {
		target = "release"
	}

	// Check for release config
	if !releaseConfigExistsFn(projectDir) {
		cli.Print("%s %s\n",
			buildErrorStyle.Render(i18n.Label("error")),
			i18n.T("cmd.build.release.error.no_config"),
		)
		cli.Print("  %s\n", buildDimStyle.Render(i18n.T("cmd.build.release.hint.create_config")))
		return core.Fail(coreerr.E("release", "config not found", nil))
	}

	// Load configuration
	cfgResult := loadReleaseConfigFn(projectDir)
	if !cfgResult.OK {
		return core.Fail(coreerr.E("release", "load config", core.NewError(cfgResult.Error())))
	}
	cfg := cfgResult.Value.(*release.Config)

	// Apply CLI overrides
	if version != "" {
		if !release.ValidateVersion(version) {
			return core.Fail(coreerr.E("release", "invalid release version override", nil))
		}
		cfg.SetVersion(version)
	}
	archiveFormatOverride := applyReleaseArchiveFormatOverride(cfg, archiveFormat)
	if !archiveFormatOverride.OK {
		return archiveFormatOverride
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
		relResult := runFullReleaseFn(ctx, cfg, dryRun)
		if !relResult.OK {
			return relResult
		}
		rel := relResult.Value.(*release.Release)

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

		return core.Ok(nil)
	case "sdk":
		sdkResult := runSDKReleaseFn(ctx, cfg, dryRun)
		if !sdkResult.OK {
			return sdkResult
		}
		sdkRelease := sdkResult.Value.(*release.SDKRelease)

		cli.Blank()
		cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("i18n.done.pass")), "SDK release completed")
		cli.Print("  %s   %s\n", i18n.Label("version"), buildTargetStyle.Render(sdkRelease.Version))
		cli.Print("  %s   %s\n", "output", buildTargetStyle.Render(sdkRelease.Output))
		cli.Print("  %s %s\n", "languages", buildTargetStyle.Render(core.Join(", ", sdkRelease.Languages...)))
		return core.Ok(nil)
	default:
		return core.Fail(coreerr.E("release", "unsupported release target: "+target, nil))
	}
}

// applyReleaseArchiveFormatOverride applies the archive-format CLI override to the release config.
//
// applyReleaseArchiveFormatOverride(cfg, "xz") // cfg.Build.ArchiveFormat = "xz"
func applyReleaseArchiveFormatOverride(cfg *release.Config, archiveFormat string) core.Result {
	if cfg == nil || archiveFormat == "" {
		return core.Ok(nil)
	}

	formatValue := resolveArchiveFormat("", archiveFormat)
	if !formatValue.OK {
		return formatValue
	}

	cfg.Build.ArchiveFormat = string(formatValue.Value.(build.ArchiveFormat))
	return core.Ok(nil)
}

func releaseAppleTestFlightRequested(target string, appleTestFlightFlag ...bool) bool {
	if len(appleTestFlightFlag) > 0 && appleTestFlightFlag[0] {
		return true
	}

	return target == "apple-testflight" || target == "testflight"
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

package ci

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/i18n"
	coreerr "dappco.re/go/log"
)

// Style aliases used by CI command output.
var (
	headerStyle  = cli.RepoStyle
	successStyle = cli.SuccessStyle
	errorStyle   = cli.ErrorStyle
	dimStyle     = cli.DimStyle
	valueStyle   = cli.ValueStyle
)

func registerCICommands(c *core.Core) {
	c.Command("ci", core.Command{
		Description: "cmd.ci.long",
		Action: func(opts core.Options) core.Result {
			dryRun := !cmdutil.OptionBool(opts, "we-are-go-for-launch")
			return runCIPublish(
				cmdutil.ContextOrBackground(),
				dryRun,
				cmdutil.OptionString(opts, "version"),
				cmdutil.OptionBool(opts, "draft"),
				cmdutil.OptionBool(opts, "prerelease"),
			)
		},
	})

	c.Command("ci/init", core.Command{
		Description: "cmd.ci.init.long",
		Action: func(opts core.Options) core.Result {
			return runCIReleaseInit()
		},
	})

	c.Command("ci/changelog", core.Command{
		Description: "cmd.ci.changelog.long",
		Action: func(opts core.Options) core.Result {
			return runChangelog(
				cmdutil.ContextOrBackground(),
				cmdutil.OptionString(opts, "from"),
				cmdutil.OptionString(opts, "to"),
			)
		},
	})

	c.Command("ci/version", core.Command{
		Description: "cmd.ci.version.long",
		Action: func(opts core.Options) core.Result {
			return runCIReleaseVersion(cmdutil.ContextOrBackground())
		},
	})
}

// runCIPublish publishes pre-built artifacts from dist/.
func runCIPublish(ctx context.Context, dryRun bool, version string, draft, prerelease bool) core.Result {
	projectDirResult := ax.Getwd()
	if !projectDirResult.OK {
		return cli.WrapVerb(core.NewError(projectDirResult.Error()), "get", "working directory")
	}
	projectDir := projectDirResult.Value.(string)

	cfgResult := release.LoadConfig(projectDir)
	if !cfgResult.OK {
		return cli.WrapVerb(core.NewError(cfgResult.Error()), "load", "config")
	}
	cfg := cfgResult.Value.(*release.Config)

	if version != "" {
		cfg.SetVersion(version)
	}

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

	cli.Print("%s %s\n", headerStyle.Render(i18n.T("cmd.ci.label.ci")), i18n.T("cmd.ci.publishing"))
	if dryRun {
		cli.Print("  %s\n", dimStyle.Render(i18n.T("cmd.ci.dry_run_hint")))
	} else {
		cli.Print("  %s\n", successStyle.Render(i18n.T("cmd.ci.go_for_launch")))
	}
	cli.Blank()

	if len(cfg.Publishers) == 0 {
		return core.Fail(coreerr.E("ci.Publish", i18n.T("cmd.ci.error.no_publishers"), nil))
	}

	relResult := release.Publish(ctx, cfg, dryRun)
	if !relResult.OK {
		cli.Print("%s %v\n", errorStyle.Render(i18n.Label("error")), relResult.Error())
		return relResult
	}
	rel := relResult.Value.(*release.Release)

	cli.Blank()
	cli.Print("%s %s\n", successStyle.Render(i18n.T("i18n.done.pass")), i18n.T("cmd.ci.publish_completed"))
	cli.Print("  %s   %s\n", i18n.Label("version"), valueStyle.Render(rel.Version))
	cli.Print("  %s %d\n", i18n.T("cmd.ci.label.artifacts"), len(rel.Artifacts))

	if !dryRun {
		for _, pub := range cfg.Publishers {
			cli.Print("  %s %s\n", i18n.T("cmd.ci.label.published"), valueStyle.Render(pub.Type))
		}
	}

	return core.Ok(nil)
}

// runCIReleaseInit scaffolds a release config.
func runCIReleaseInit() core.Result {
	cwdResult := ax.Getwd()
	if !cwdResult.OK {
		return cli.Wrap(core.NewError(cwdResult.Error()), i18n.T("i18n.fail.get", "working directory"))
	}
	cwd := cwdResult.Value.(string)

	return runCIReleaseInitInDir(cwd)
}

func runCIReleaseInitInDir(cwd string) core.Result {
	cli.Print("%s %s\n\n", dimStyle.Render(i18n.Label("init")), i18n.T("cmd.ci.init.initializing"))

	if release.ConfigExists(cwd) {
		cli.Text(i18n.T("cmd.ci.init.already_initialised"))
		return core.Ok(nil)
	}

	cfg := release.ScaffoldConfig()
	written := release.WriteConfig(cfg, cwd)
	if !written.OK {
		return cli.Wrap(core.NewError(written.Error()), i18n.T("i18n.fail.create", "config"))
	}

	cli.Blank()
	cli.Print("%s %s\n", successStyle.Render("v"), i18n.T("cmd.ci.init.created_config"))
	cli.Blank()
	cli.Text(i18n.T("cmd.ci.init.next_steps"))
	cli.Print("  %s\n", i18n.T("cmd.ci.init.edit_config"))
	cli.Print("  %s\n", i18n.T("cmd.ci.init.run_ci"))

	return core.Ok(nil)
}

// runChangelog generates a changelog between two git refs.
func runChangelog(ctx context.Context, fromRef, toRef string) core.Result {
	cwdResult := ax.Getwd()
	if !cwdResult.OK {
		return cli.Wrap(core.NewError(cwdResult.Error()), i18n.T("i18n.fail.get", "working directory"))
	}
	cwd := cwdResult.Value.(string)

	if fromRef == "" || toRef == "" {
		tagResult := latestTagWithContext(ctx, cwd)
		if tagResult.OK {
			tag := tagResult.Value.(string)
			if fromRef == "" {
				fromRef = tag
			}
			if toRef == "" {
				toRef = "HEAD"
			}
		} else {
			if ctx.Err() != nil {
				return core.Fail(ctx.Err())
			}
			cli.Text(i18n.T("cmd.ci.changelog.no_tags"))
			return core.Ok(nil)
		}
	}

	cli.Print("%s %s..%s\n\n", dimStyle.Render(i18n.T("cmd.ci.changelog.generating")), fromRef, toRef)

	changelogResult := release.GenerateWithContext(ctx, cwd, fromRef, toRef)
	if !changelogResult.OK {
		return cli.Wrap(core.NewError(changelogResult.Error()), i18n.T("i18n.fail.generate", "changelog"))
	}
	changelog := changelogResult.Value.(string)

	cli.Text(changelog)
	return core.Ok(nil)
}

// runCIReleaseVersion shows the determined version.
func runCIReleaseVersion(ctx context.Context) core.Result {
	projectDirResult := ax.Getwd()
	if !projectDirResult.OK {
		return cli.WrapVerb(core.NewError(projectDirResult.Error()), "get", "working directory")
	}
	projectDir := projectDirResult.Value.(string)

	versionResult := release.DetermineVersionWithContext(ctx, projectDir)
	if !versionResult.OK {
		return cli.WrapVerb(core.NewError(versionResult.Error()), "determine", "version")
	}
	version := versionResult.Value.(string)

	cli.Print("%s %s\n", i18n.Label("version"), valueStyle.Render(version))
	return core.Ok(nil)
}

func latestTagWithContext(ctx context.Context, dir string) core.Result {
	out := ax.RunDir(ctx, dir, "git", "describe", "--tags", "--abbrev=0")
	if !out.OK {
		return out
	}
	return core.Ok(core.Trim(out.Value.(string)))
}

package ci

import (
	"context"

	"dappco.re/go/core"
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
			return cmdutil.ResultFromError(runCIPublish(
				cmdutil.ContextOrBackground(),
				dryRun,
				cmdutil.OptionString(opts, "version"),
				cmdutil.OptionBool(opts, "draft"),
				cmdutil.OptionBool(opts, "prerelease"),
			))
		},
	})

	c.Command("ci/init", core.Command{
		Description: "cmd.ci.init.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runCIReleaseInit())
		},
	})

	c.Command("ci/changelog", core.Command{
		Description: "cmd.ci.changelog.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runChangelog(
				cmdutil.ContextOrBackground(),
				cmdutil.OptionString(opts, "from"),
				cmdutil.OptionString(opts, "to"),
			))
		},
	})

	c.Command("ci/version", core.Command{
		Description: "cmd.ci.version.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runCIReleaseVersion(cmdutil.ContextOrBackground()))
		},
	})
}

// runCIPublish publishes pre-built artifacts from dist/.
func runCIPublish(ctx context.Context, dryRun bool, version string, draft, prerelease bool) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return cli.WrapVerb(err, "get", "working directory")
	}

	cfg, err := release.LoadConfig(projectDir)
	if err != nil {
		return cli.WrapVerb(err, "load", "config")
	}

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
		return coreerr.E("ci.Publish", i18n.T("cmd.ci.error.no_publishers"), nil)
	}

	rel, err := release.Publish(ctx, cfg, dryRun)
	if err != nil {
		cli.Print("%s %v\n", errorStyle.Render(i18n.Label("error")), err)
		return err
	}

	cli.Blank()
	cli.Print("%s %s\n", successStyle.Render(i18n.T("i18n.done.pass")), i18n.T("cmd.ci.publish_completed"))
	cli.Print("  %s   %s\n", i18n.Label("version"), valueStyle.Render(rel.Version))
	cli.Print("  %s %d\n", i18n.T("cmd.ci.label.artifacts"), len(rel.Artifacts))

	if !dryRun {
		for _, pub := range cfg.Publishers {
			cli.Print("  %s %s\n", i18n.T("cmd.ci.label.published"), valueStyle.Render(pub.Type))
		}
	}

	return nil
}

// runCIReleaseInit scaffolds a release config.
func runCIReleaseInit() error {
	cwd, err := ax.Getwd()
	if err != nil {
		return cli.Wrap(err, i18n.T("i18n.fail.get", "working directory"))
	}

	return runCIReleaseInitInDir(cwd)
}

func runCIReleaseInitInDir(cwd string) error {
	cli.Print("%s %s\n\n", dimStyle.Render(i18n.Label("init")), i18n.T("cmd.ci.init.initializing"))

	if release.ConfigExists(cwd) {
		cli.Text(i18n.T("cmd.ci.init.already_initialised"))
		return nil
	}

	cfg := release.ScaffoldConfig()
	if err := release.WriteConfig(cfg, cwd); err != nil {
		return cli.Wrap(err, i18n.T("i18n.fail.create", "config"))
	}

	cli.Blank()
	cli.Print("%s %s\n", successStyle.Render("v"), i18n.T("cmd.ci.init.created_config"))
	cli.Blank()
	cli.Text(i18n.T("cmd.ci.init.next_steps"))
	cli.Print("  %s\n", i18n.T("cmd.ci.init.edit_config"))
	cli.Print("  %s\n", i18n.T("cmd.ci.init.run_ci"))

	return nil
}

// runChangelog generates a changelog between two git refs.
func runChangelog(ctx context.Context, fromRef, toRef string) error {
	cwd, err := ax.Getwd()
	if err != nil {
		return cli.Wrap(err, i18n.T("i18n.fail.get", "working directory"))
	}

	if fromRef == "" || toRef == "" {
		tag, err := latestTagWithContext(ctx, cwd)
		if err == nil {
			if fromRef == "" {
				fromRef = tag
			}
			if toRef == "" {
				toRef = "HEAD"
			}
		} else {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			cli.Text(i18n.T("cmd.ci.changelog.no_tags"))
			return nil
		}
	}

	cli.Print("%s %s..%s\n\n", dimStyle.Render(i18n.T("cmd.ci.changelog.generating")), fromRef, toRef)

	changelog, err := release.GenerateWithContext(ctx, cwd, fromRef, toRef)
	if err != nil {
		return cli.Wrap(err, i18n.T("i18n.fail.generate", "changelog"))
	}

	cli.Text(changelog)
	return nil
}

// runCIReleaseVersion shows the determined version.
func runCIReleaseVersion(ctx context.Context) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return cli.WrapVerb(err, "get", "working directory")
	}

	version, err := release.DetermineVersionWithContext(ctx, projectDir)
	if err != nil {
		return cli.WrapVerb(err, "determine", "version")
	}

	cli.Print("%s %s\n", i18n.Label("version"), valueStyle.Render(version))
	return nil
}

func latestTagWithContext(ctx context.Context, dir string) (string, error) {
	out, err := ax.RunDir(ctx, dir, "git", "describe", "--tags", "--abbrev=0")
	if err != nil {
		return "", err
	}
	return core.Trim(out), nil
}

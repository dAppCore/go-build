package ci

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/pkg/release"
)

// Style aliases used by CI command output.
var (
	headerStyle  = cli.RepoStyle
	successStyle = cli.SuccessStyle
	errorStyle   = cli.ErrorStyle
	dimStyle     = cli.DimStyle
	valueStyle   = cli.ValueStyle
)

func registerCICommands(c *core.Core) core.Result {
	if r := c.Command("ci", core.Command{
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
	}); !r.OK {
		return r
	}

	if r := c.Command("ci/init", core.Command{
		Description: "cmd.ci.init.long",
		Action: func(opts core.Options) core.Result {
			return runCIReleaseInit()
		},
	}); !r.OK {
		return r
	}

	if r := c.Command("ci/changelog", core.Command{
		Description: "cmd.ci.changelog.long",
		Action: func(opts core.Options) core.Result {
			return runChangelog(
				cmdutil.ContextOrBackground(),
				cmdutil.OptionString(opts, "from"),
				cmdutil.OptionString(opts, "to"),
			)
		},
	}); !r.OK {
		return r
	}

	if r := c.Command("ci/version", core.Command{
		Description: "cmd.ci.version.long",
		Action: func(opts core.Options) core.Result {
			return runCIReleaseVersion(cmdutil.ContextOrBackground())
		},
	}); !r.OK {
		return r
	}
	return core.Ok(nil)
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

	cli.Print("%s %s\n", headerStyle.Render("CI"), "Publishing release")
	if dryRun {
		cli.Print("  %s\n", dimStyle.Render("Dry run: no publishers will be changed"))
	} else {
		cli.Print("  %s\n", successStyle.Render("Publishing enabled"))
	}
	cli.Blank()

	if len(cfg.Publishers) == 0 {
		return core.Fail(core.E("ci.Publish", "no publishers configured", nil))
	}

	relResult := release.Publish(ctx, cfg, dryRun)
	if !relResult.OK {
		cli.Print("%s %v\n", errorStyle.Render("error:"), relResult.Error())
		return relResult
	}
	rel := relResult.Value.(*release.Release)

	cli.Blank()
	cli.Print("%s %s\n", successStyle.Render("Done"), "Publish completed")
	cli.Print("  %s   %s\n", "version:", valueStyle.Render(rel.Version))
	cli.Print("  %s %d\n", "artifacts", len(rel.Artifacts))

	if !dryRun {
		for _, pub := range cfg.Publishers {
			cli.Print("  %s %s\n", "published", valueStyle.Render(pub.Type))
		}
	}

	return core.Ok(nil)
}

// runCIReleaseInit scaffolds a release config.
func runCIReleaseInit() core.Result {
	cwdResult := ax.Getwd()
	if !cwdResult.OK {
		return cli.Wrap(core.NewError(cwdResult.Error()), "failed to get working directory")
	}
	cwd := cwdResult.Value.(string)

	return runCIReleaseInitInDir(cwd)
}

func runCIReleaseInitInDir(cwd string) core.Result {
	cli.Print("%s %s\n\n", dimStyle.Render("init:"), "Initializing release config")

	if release.ConfigExists(cwd) {
		cli.Text("Release config already initialised")
		return core.Ok(nil)
	}

	cfg := release.ScaffoldConfig()
	written := release.WriteConfig(cfg, cwd)
	if !written.OK {
		return cli.Wrap(core.NewError(written.Error()), "failed to create config")
	}

	cli.Blank()
	cli.Print("%s %s\n", successStyle.Render("v"), "Created .core/release.yaml")
	cli.Blank()
	cli.Text("Next steps")
	cli.Print("  %s\n", "Edit .core/release.yaml")
	cli.Print("  %s\n", "Run core ci")

	return core.Ok(nil)
}

// runChangelog generates a changelog between two git refs.
func runChangelog(ctx context.Context, fromRef, toRef string) core.Result {
	cwdResult := ax.Getwd()
	if !cwdResult.OK {
		return cli.Wrap(core.NewError(cwdResult.Error()), "failed to get working directory")
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
			cli.Text("No tags found")
			return core.Ok(nil)
		}
	}

	cli.Print("%s %s..%s\n\n", dimStyle.Render("Generating changelog"), fromRef, toRef)

	changelogResult := release.GenerateWithContext(ctx, cwd, fromRef, toRef)
	if !changelogResult.OK {
		return cli.Wrap(core.NewError(changelogResult.Error()), "failed to generate changelog")
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

	cli.Print("%s %s\n", "version:", valueStyle.Render(version))
	return core.Ok(nil)
}

func latestTagWithContext(ctx context.Context, dir string) core.Result {
	out := ax.RunDir(ctx, dir, "git", "describe", "--tags", "--abbrev=0")
	if !out.OK {
		return out
	}
	return core.Ok(core.Trim(out.Value.(string)))
}

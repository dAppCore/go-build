package ci

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/release"
	"dappco.re/go/core/i18n"
	coreerr "dappco.re/go/core/log"
	"dappco.re/go/core/cli/pkg/cli"
)

// Style aliases used by CI command output.
var (
	headerStyle  = cli.RepoStyle
	successStyle = cli.SuccessStyle
	errorStyle   = cli.ErrorStyle
	dimStyle     = cli.DimStyle
	valueStyle   = cli.ValueStyle
)

// Flag variables for ci command.
var (
	ciLaunchMode bool
	ciVersion    string
	ciDraft      bool
	ciPrerelease bool
)

// Flag variables for changelog subcommand.
var (
	changelogFromRef string
	changelogToRef   string
)

var ciCmd = &cli.Command{
	Use: "ci",
	RunE: func(cmd *cli.Command, args []string) error {
		dryRun := !ciLaunchMode
		return runCIPublish(cmd.Context(), dryRun, ciVersion, ciDraft, ciPrerelease)
	},
}

var ciInitCmd = &cli.Command{
	Use: "init",
	RunE: func(cmd *cli.Command, args []string) error {
		return runCIReleaseInit()
	},
}

var ciChangelogCmd = &cli.Command{
	Use: "changelog",
	RunE: func(cmd *cli.Command, args []string) error {
		return runChangelog(cmd.Context(), changelogFromRef, changelogToRef)
	},
}

var ciVersionCmd = &cli.Command{
	Use: "version",
	RunE: func(cmd *cli.Command, args []string) error {
		return runCIReleaseVersion(cmd.Context())
	},
}

func setCII18n() {
	ciCmd.Short = i18n.T("cmd.ci.short")
	ciCmd.Long = i18n.T("cmd.ci.long")
	ciInitCmd.Short = i18n.T("cmd.ci.init.short")
	ciInitCmd.Long = i18n.T("cmd.ci.init.long")
	ciChangelogCmd.Short = i18n.T("cmd.ci.changelog.short")
	ciChangelogCmd.Long = i18n.T("cmd.ci.changelog.long")
	ciVersionCmd.Short = i18n.T("cmd.ci.version.short")
	ciVersionCmd.Long = i18n.T("cmd.ci.version.long")
}

func initCIFlags() {
	// Main ci command flags
	ciCmd.Flags().BoolVar(&ciLaunchMode, "we-are-go-for-launch", false, i18n.T("cmd.ci.flag.go_for_launch"))
	ciCmd.Flags().StringVar(&ciVersion, "version", "", i18n.T("cmd.ci.flag.version"))
	ciCmd.Flags().BoolVar(&ciDraft, "draft", false, i18n.T("cmd.ci.flag.draft"))
	ciCmd.Flags().BoolVar(&ciPrerelease, "prerelease", false, i18n.T("cmd.ci.flag.prerelease"))

	// Changelog subcommand flags
	ciChangelogCmd.Flags().StringVar(&changelogFromRef, "from", "", i18n.T("cmd.ci.changelog.flag.from"))
	ciChangelogCmd.Flags().StringVar(&changelogToRef, "to", "", i18n.T("cmd.ci.changelog.flag.to"))

	// Add subcommands
	ciCmd.AddCommand(ciInitCmd)
	ciCmd.AddCommand(ciChangelogCmd)
	ciCmd.AddCommand(ciVersionCmd)
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
		return cli.Err("%s: %w", i18n.T("i18n.fail.get", "working directory"), err)
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
		return cli.Err("%s: %w", i18n.T("i18n.fail.create", "config"), err)
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
		return cli.Err("%s: %w", i18n.T("i18n.fail.get", "working directory"), err)
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
		return cli.Err("%s: %w", i18n.T("i18n.fail.generate", "changelog"), err)
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

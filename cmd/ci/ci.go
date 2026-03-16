package ci

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"forge.lthn.ai/core/cli/pkg/cli"
	"forge.lthn.ai/core/go-i18n"
	"forge.lthn.ai/core/go-build/pkg/release"
	coreerr "forge.lthn.ai/core/go-log"
)

// Style aliases from shared
var (
	headerStyle  = cli.RepoStyle
	successStyle = cli.SuccessStyle
	errorStyle   = cli.ErrorStyle
	dimStyle     = cli.DimStyle
	valueStyle   = cli.ValueStyle
)

// Flag variables for ci command
var (
	ciGoForLaunch bool
	ciVersion     string
	ciDraft       bool
	ciPrerelease  bool
)

// Flag variables for changelog subcommand
var (
	changelogFromRef string
	changelogToRef   string
)

var ciCmd = &cli.Command{
	Use:   "ci",
	Short: i18n.T("cmd.ci.short"),
	Long:  i18n.T("cmd.ci.long"),
	RunE: func(cmd *cli.Command, args []string) error {
		dryRun := !ciGoForLaunch
		return runCIPublish(dryRun, ciVersion, ciDraft, ciPrerelease)
	},
}

var ciInitCmd = &cli.Command{
	Use:   "init",
	Short: i18n.T("cmd.ci.init.short"),
	Long:  i18n.T("cmd.ci.init.long"),
	RunE: func(cmd *cli.Command, args []string) error {
		return runCIReleaseInit()
	},
}

var ciChangelogCmd = &cli.Command{
	Use:   "changelog",
	Short: i18n.T("cmd.ci.changelog.short"),
	Long:  i18n.T("cmd.ci.changelog.long"),
	RunE: func(cmd *cli.Command, args []string) error {
		return runChangelog(changelogFromRef, changelogToRef)
	},
}

var ciVersionCmd = &cli.Command{
	Use:   "version",
	Short: i18n.T("cmd.ci.version.short"),
	Long:  i18n.T("cmd.ci.version.long"),
	RunE: func(cmd *cli.Command, args []string) error {
		return runCIReleaseVersion()
	},
}

func init() {
	// Main ci command flags
	ciCmd.Flags().BoolVar(&ciGoForLaunch, "we-are-go-for-launch", false, i18n.T("cmd.ci.flag.go_for_launch"))
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
func runCIPublish(dryRun bool, version string, draft, prerelease bool) error {
	ctx := context.Background()

	projectDir, err := os.Getwd()
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
	cwd, err := os.Getwd()
	if err != nil {
		return cli.Err("%s: %w", i18n.T("i18n.fail.get", "working directory"), err)
	}

	cli.Print("%s %s\n\n", dimStyle.Render(i18n.Label("init")), i18n.T("cmd.ci.init.initializing"))

	if release.ConfigExists(cwd) {
		cli.Text(i18n.T("cmd.ci.init.already_initialised"))
		return nil
	}

	cfg := release.DefaultConfig()
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
func runChangelog(fromRef, toRef string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return cli.Err("%s: %w", i18n.T("i18n.fail.get", "working directory"), err)
	}

	if fromRef == "" || toRef == "" {
		tag, err := latestTag(cwd)
		if err == nil {
			if fromRef == "" {
				fromRef = tag
			}
			if toRef == "" {
				toRef = "HEAD"
			}
		} else {
			cli.Text(i18n.T("cmd.ci.changelog.no_tags"))
			return nil
		}
	}

	cli.Print("%s %s..%s\n\n", dimStyle.Render(i18n.T("cmd.ci.changelog.generating")), fromRef, toRef)

	changelog, err := release.Generate(cwd, fromRef, toRef)
	if err != nil {
		return cli.Err("%s: %w", i18n.T("i18n.fail.generate", "changelog"), err)
	}

	cli.Text(changelog)
	return nil
}

// runCIReleaseVersion shows the determined version.
func runCIReleaseVersion() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return cli.WrapVerb(err, "get", "working directory")
	}

	version, err := release.DetermineVersion(projectDir)
	if err != nil {
		return cli.WrapVerb(err, "determine", "version")
	}

	cli.Print("%s %s\n", i18n.Label("version"), valueStyle.Render(version))
	return nil
}

func latestTag(dir string) (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

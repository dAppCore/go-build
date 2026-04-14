// Package buildcmd registers auto-detected project build commands.
package buildcmd

import (
	"embed"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/cmdutil"
	_ "dappco.re/go/core/build/locales" // registers locale translations
	"dappco.re/go/core/cli/pkg/cli"
)

func init() {
	cli.RegisterCommands(AddBuildCommands)
}

// Style aliases used by build command output.
var (
	buildHeaderStyle  = cli.TitleStyle
	buildTargetStyle  = cli.ValueStyle
	buildSuccessStyle = cli.SuccessStyle
	buildErrorStyle   = cli.ErrorStyle
	buildDimStyle     = cli.DimStyle
)

//go:embed all:tmpl/gui
var guiTemplate embed.FS

// AddBuildCommands registers the 'build' command and all subcommands.
//
// buildcmd.AddBuildCommands(root)
func AddBuildCommands(c *core.Core) {
	c.Command("build", core.Command{
		Description: "cmd.build.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runProjectBuild(ProjectBuildRequest{
				Context:        cmdutil.ContextOrBackground(),
				BuildType:      cmdutil.OptionString(opts, "type"),
				CIMode:         cmdutil.OptionBool(opts, "ci"),
				TargetsFlag:    cmdutil.OptionString(opts, "targets"),
				OutputDir:      cmdutil.OptionString(opts, "output"),
				ArchiveOutput:  cmdutil.OptionBoolDefault(opts, true, "archive"),
				ChecksumOutput: cmdutil.OptionBoolDefault(opts, true, "checksum"),
				ArchiveFormat:  cmdutil.OptionString(opts, "archive-format"),
				ConfigPath:     cmdutil.OptionString(opts, "config"),
				Format:         cmdutil.OptionString(opts, "format"),
				Push:           cmdutil.OptionBool(opts, "push"),
				ImageName:      cmdutil.OptionString(opts, "image"),
				NoSign:         cmdutil.OptionBool(opts, "no-sign"),
				Notarize:       cmdutil.OptionBool(opts, "notarize"),
				Verbose:        cmdutil.OptionBool(opts, "verbose", "v"),
			}))
		},
	})

	c.Command("build/from-path", core.Command{
		Description: "cmd.build.from_path.short",
		Action: func(opts core.Options) core.Result {
			fromPath := cmdutil.OptionString(opts, "path")
			if fromPath == "" {
				return cmdutil.ResultFromError(errPathRequired)
			}
			return cmdutil.ResultFromError(runBuild(cmdutil.ContextOrBackground(), fromPath))
		},
	})

	c.Command("build/pwa", core.Command{
		Description: "cmd.build.pwa.short",
		Action: func(opts core.Options) core.Result {
			pwaURL := cmdutil.OptionString(opts, "url")
			if pwaURL == "" {
				return cmdutil.ResultFromError(errURLRequired)
			}
			return cmdutil.ResultFromError(runPwaBuild(cmdutil.ContextOrBackground(), pwaURL))
		},
	})

	c.Command("build/sdk", core.Command{
		Description: "cmd.build.sdk.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runBuildSDK(
				cmdutil.ContextOrBackground(),
				cmdutil.OptionString(opts, "spec"),
				cmdutil.OptionString(opts, "lang"),
				cmdutil.OptionString(opts, "version"),
				cmdutil.OptionBool(opts, "dry-run"),
			))
		},
	})

	AddAppleCommand(c)
	AddReleaseCommand(c)
	AddWorkflowCommand(c)
}

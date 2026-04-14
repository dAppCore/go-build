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
				TargetsFlag:    cmdutil.OptionString(opts, "targets", "build-platform", "build_platform"),
				OutputDir:      cmdutil.OptionString(opts, "output"),
				BuildName:      cmdutil.OptionString(opts, "name", "build-name", "build_name"),
				BuildTagsFlag:  cmdutil.OptionString(opts, "build-tags", "build_tags"),
				Obfuscate:      cmdutil.OptionBool(opts, "build-obfuscate", "build_obfuscate", "obfuscate"),
				ObfuscateSet:   cmdutil.OptionHas(opts, "build-obfuscate", "build_obfuscate", "obfuscate"),
				NSIS:           cmdutil.OptionBool(opts, "nsis"),
				NSISSet:        cmdutil.OptionHas(opts, "nsis"),
				WebView2:       cmdutil.OptionString(opts, "wails-build-webview2", "wails_build_webview2", "webview2"),
				WebView2Set:    cmdutil.OptionHas(opts, "wails-build-webview2", "wails_build_webview2", "webview2"),
				DenoBuild:      cmdutil.OptionString(opts, "deno-build", "deno_build"),
				DenoBuildSet:   cmdutil.OptionHas(opts, "deno-build", "deno_build"),
				BuildCache:     cmdutil.OptionBool(opts, "build-cache", "build_cache"),
				BuildCacheSet:  cmdutil.OptionHas(opts, "build-cache", "build_cache"),
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

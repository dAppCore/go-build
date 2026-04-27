// Package buildcmd registers auto-detected project build commands.
package buildcmd

import (
	"embed"

	"dappco.re/go/build/internal/cmdutil"
	_ "dappco.re/go/build/locales" // registers locale translations
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/core"
)

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
			archiveOutput := cmdutil.OptionBoolDefault(opts, false, "archive")
			archiveOutputSet := cmdutil.OptionHas(opts, "archive")
			checksumOutput := cmdutil.OptionBoolDefault(opts, false, "checksum")
			checksumOutputSet := cmdutil.OptionHas(opts, "checksum")
			packageEnabled := cmdutil.OptionBoolDefault(opts, false, "package")
			packageSet := cmdutil.OptionHas(opts, "package")
			archiveOutput, checksumOutput = resolvePackageOutputs(
				packageEnabled,
				packageSet,
				archiveOutput,
				archiveOutputSet,
				checksumOutput,
				checksumOutputSet,
			)

			return cmdutil.ResultFromError(runProjectBuild(ProjectBuildRequest{
				Context:           cmdutil.ContextOrBackground(),
				BuildType:         cmdutil.OptionString(opts, "type"),
				Version:           cmdutil.OptionString(opts, "version"),
				CIMode:            cmdutil.OptionBool(opts, "ci"),
				TargetsFlag:       cmdutil.OptionString(opts, "targets", "build-platform", "build_platform"),
				OutputDir:         cmdutil.OptionString(opts, "output"),
				BuildName:         cmdutil.OptionString(opts, "name", "build-name", "build_name"),
				BuildTagsFlag:     cmdutil.OptionString(opts, "build-tags", "build_tags"),
				Obfuscate:         cmdutil.OptionBool(opts, "build-obfuscate", "build_obfuscate", "obfuscate"),
				ObfuscateSet:      cmdutil.OptionHas(opts, "build-obfuscate", "build_obfuscate", "obfuscate"),
				NSIS:              cmdutil.OptionBool(opts, "nsis"),
				NSISSet:           cmdutil.OptionHas(opts, "nsis"),
				WebView2:          cmdutil.OptionString(opts, "wails-build-webview2", "wails_build_webview2", "webview2"),
				WebView2Set:       cmdutil.OptionHas(opts, "wails-build-webview2", "wails_build_webview2", "webview2"),
				DenoBuild:         cmdutil.OptionString(opts, "deno-build", "deno_build"),
				DenoBuildSet:      cmdutil.OptionHas(opts, "deno-build", "deno_build"),
				NpmBuild:          cmdutil.OptionString(opts, "npm-build", "npm_build"),
				NpmBuildSet:       cmdutil.OptionHas(opts, "npm-build", "npm_build"),
				BuildCache:        cmdutil.OptionBool(opts, "build-cache", "build_cache"),
				BuildCacheSet:     cmdutil.OptionHas(opts, "build-cache", "build_cache"),
				ArchiveOutput:     archiveOutput,
				ArchiveOutputSet:  archiveOutputSet,
				ChecksumOutput:    checksumOutput,
				ChecksumOutputSet: checksumOutputSet,
				PackageSet:        packageSet,
				ArchiveFormat:     cmdutil.OptionString(opts, "archive-format"),
				ConfigPath:        cmdutil.OptionString(opts, "config"),
				Format:            cmdutil.OptionString(opts, "format"),
				Push:              cmdutil.OptionBool(opts, "push"),
				ImageName:         cmdutil.OptionString(opts, "image"),
				Sign:              cmdutil.OptionBoolDefault(opts, true, "sign"),
				SignSet:           cmdutil.OptionHas(opts, "sign"),
				NoSign: resolveNoSign(
					cmdutil.OptionBool(opts, "no-sign"),
					cmdutil.OptionBoolDefault(opts, true, "sign"),
					cmdutil.OptionHas(opts, "sign"),
				),
				Notarize: cmdutil.OptionBool(opts, "notarize"),
				Verbose:  cmdutil.OptionBool(opts, "verbose", "v"),
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
			pwaPath := cmdutil.OptionString(opts, "path")
			pwaURL := cmdutil.OptionString(opts, "url")
			switch {
			case pwaPath != "":
				return cmdutil.ResultFromError(runLocalPwaBuild(cmdutil.ContextOrBackground(), pwaPath))
			case pwaURL != "":
				return cmdutil.ResultFromError(runPwaBuild(cmdutil.ContextOrBackground(), pwaURL))
			default:
				return cmdutil.ResultFromError(errPWAInputRequired)
			}
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
				cmdutil.OptionBool(opts, "skip-unavailable", "skip_unavailable"),
			))
		},
	})

	AddAppleCommand(c)
	AddImageCommand(c)
	AddInstallersCommand(c)
	AddReleaseCommand(c)
	AddServiceCommands(c)
	AddWorkflowCommand(c)
}

func resolveNoSign(noSign bool, signEnabled bool, signSet bool) bool {
	if noSign {
		return true
	}
	if signSet && !signEnabled {
		return true
	}
	return false
}

func resolvePackageOutputs(packageEnabled bool, packageSet bool, archiveOutput bool, archiveOutputSet bool, checksumOutput bool, checksumOutputSet bool) (bool, bool) {
	if !packageSet {
		return archiveOutput, checksumOutput
	}

	if !archiveOutputSet {
		archiveOutput = packageEnabled
	}
	if !checksumOutputSet {
		checksumOutput = packageEnabled
	}

	return archiveOutput, checksumOutput
}

// Package sdkcmd provides SDK generation, validation, and API compatibility commands.
//
// Commands:
//   - sdk: generate SDKs from the detected or configured OpenAPI spec
//   - sdk diff: check for breaking API changes between spec versions
//   - sdk validate: validate OpenAPI spec syntax
//
// The legacy `core build sdk` alias remains available through cmd/build.
package sdkcmd

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/internal/sdkcfg"
	"dappco.re/go/build/pkg/sdk"
	storage "dappco.re/go/build/pkg/storage"
	"github.com/oasdiff/oasdiff/checker"
)

// SDK styles (aliases to shared)
var (
	sdkHeaderStyle  = cli.TitleStyle
	sdkTargetStyle  = cli.ValueStyle
	sdkSuccessStyle = cli.SuccessStyle
	sdkErrorStyle   = cli.ErrorStyle
	sdkDimStyle     = cli.DimStyle
)

// AddSDKCommands registers the 'sdk' command and all subcommands.
//
// sdkcmd.AddSDKCommands(root)
func AddSDKCommands(c *core.Core) {
	registerSDKGenerateCommand(c, "sdk")
	registerSDKGenerateCommand(c, "sdk/generate")

	c.Command("sdk/diff", core.Command{
		Description: "cmd.sdk.diff.long",
		Action: func(opts core.Options) core.Result {
			return runSDKDiff(
				cmdutil.OptionString(opts, "base"),
				cmdutil.OptionString(opts, "spec"),
				cmdutil.OptionBool(opts, "fail-on-warn", "fail_on_warn"),
			)
		},
	})

	c.Command("sdk/validate", core.Command{
		Description: "cmd.sdk.validate.long",
		Action: func(opts core.Options) core.Result {
			return runSDKValidate(
				cmdutil.OptionString(opts, "spec"),
			)
		},
	})
}

func registerSDKGenerateCommand(c *core.Core, path string) {
	c.Command(path, core.Command{
		Description: "cmd.sdk.long",
		Action: func(opts core.Options) core.Result {
			return runSDKGenerate(
				cmdutil.ContextOrBackground(),
				cmdutil.OptionString(opts, "spec"),
				cmdutil.OptionString(opts, "lang"),
				cmdutil.OptionString(opts, "version"),
				cmdutil.OptionBool(opts, "dry-run"),
				cmdutil.OptionBool(opts, "skip-unavailable", "skip_unavailable"),
			)
		},
	})
}

func runSDKGenerate(ctx context.Context, specPath, lang, version string, dryRun bool, skipUnavailable bool) core.Result {
	projectDirResult := ax.Getwd()
	if !projectDirResult.OK {
		return core.Fail(core.E("sdk.Generate", "failed to get working directory", core.NewError(projectDirResult.Error())))
	}
	projectDir := projectDirResult.Value.(string)

	return runSDKGenerateInDir(ctx, projectDir, specPath, lang, version, dryRun, skipUnavailable)
}

func runSDKGenerateInDir(ctx context.Context, projectDir, specPath, lang, version string, dryRun bool, skipUnavailable bool) core.Result {
	configResult := sdkcfg.LoadProjectConfig(storage.Local, projectDir)
	if !configResult.OK {
		return core.Fail(core.E("sdk.Generate", "failed to load sdk config", core.NewError(configResult.Error())))
	}
	config := configResult.Value.(*sdk.Config)
	if specPath != "" {
		config.Spec = specPath
	}
	if skipUnavailable {
		config.SkipUnavailable = true
	}

	s := sdk.New(projectDir, config)
	if version != "" {
		s.SetVersion(version)
	}
	resolvedConfig := s.Config()

	cli.Print("%s %s\n", sdkHeaderStyle.Render("SDK"), "Generating SDKs")
	if dryRun {
		cli.Print("  %s\n", sdkDimStyle.Render("Dry run mode"))
	}
	cli.Blank()

	detectedSpecResult := s.ValidateSpec(ctx)
	if !detectedSpecResult.OK {
		cli.Print("%s %v\n", sdkErrorStyle.Render("error"), detectedSpecResult.Error())
		return detectedSpecResult
	}
	detectedSpec := detectedSpecResult.Value.(string)
	cli.Print("  %s %s\n", "spec", sdkTargetStyle.Render(detectedSpec))

	if dryRun {
		if lang != "" {
			cli.Print("  %s %s\n", "language", sdkTargetStyle.Render(lang))
		} else {
			cli.Print("  %s %s\n", "languages", sdkTargetStyle.Render(core.Join(", ", resolvedConfig.Languages...)))
		}
		cli.Blank()
		cli.Print("%s %s\n", sdkSuccessStyle.Render("OK"), "Would generate SDKs")
		return core.Ok(nil)
	}

	if lang != "" {
		result := s.GenerateLanguageWithStatus(ctx, lang)
		if !result.OK {
			cli.Print("%s %v\n", sdkErrorStyle.Render("error"), result.Error())
			return result
		}
		status := result.Value.(sdk.LanguageResult)
		if status.Skipped {
			cli.Print("  %s %s\n", "Skipped:", sdkTargetStyle.Render(status.Language))
		} else {
			cli.Print("  %s %s\n", "generated", sdkTargetStyle.Render(status.Language))
		}
	} else {
		resultsResult := s.GenerateWithStatus(ctx)
		if !resultsResult.OK {
			cli.Print("%s %v\n", sdkErrorStyle.Render("error"), resultsResult.Error())
			return resultsResult
		}
		results := resultsResult.Value.([]sdk.LanguageResult)
		generated := make([]string, 0, len(results))
		skipped := make([]string, 0)
		for _, result := range results {
			if result.Generated {
				generated = append(generated, result.Language)
			}
			if result.Skipped {
				skipped = append(skipped, result.Language)
			}
		}
		if len(generated) > 0 {
			cli.Print("  %s %s\n", "generated", sdkTargetStyle.Render(core.Join(", ", generated...)))
		}
		if len(skipped) > 0 {
			cli.Print("  %s %s\n", "Skipped:", sdkTargetStyle.Render(core.Join(", ", skipped...)))
		}
	}

	cli.Blank()
	cli.Print("%s %s\n", sdkSuccessStyle.Render("Success"), "SDK generation complete")
	return core.Ok(nil)
}

func runSDKDiff(basePath, specPath string, failOnWarn bool) core.Result {
	projectDirResult := ax.Getwd()
	if !projectDirResult.OK {
		return core.Fail(core.E("sdk.Diff", "failed to get working directory", core.NewError(projectDirResult.Error())))
	}
	projectDir := projectDirResult.Value.(string)

	return runSDKDiffInDir(projectDir, basePath, specPath, failOnWarn)
}

func runSDKDiffInDir(projectDir, basePath, specPath string, failOnWarn bool) core.Result {
	if specPath == "" {
		configResult := sdkcfg.LoadProjectConfig(storage.Local, projectDir)
		if !configResult.OK {
			return core.Fail(core.E("sdk.Diff", "failed to load sdk config", core.NewError(configResult.Error())))
		}
		config := configResult.Value.(*sdk.Config)

		s := sdk.New(projectDir, config)
		specPathResult := s.DetectSpec()
		if !specPathResult.OK {
			return specPathResult
		}
		specPath = specPathResult.Value.(string)
	}

	if basePath == "" {
		return core.Fail(core.E("sdk.Diff", "base spec is required", nil))
	}

	cli.Print("%s %s\n", sdkHeaderStyle.Render("SDK diff"), "Checking breaking changes...")
	cli.Print("  %s %s\n", "base:", sdkDimStyle.Render(basePath))
	cli.Print("  %s %s\n", "current:", sdkDimStyle.Render(specPath))
	cli.Blank()

	diffOptions := sdk.DiffOptions{}
	if failOnWarn {
		diffOptions.MinimumLevel = checker.WARN
	}

	diffResult := sdk.DiffWithOptions(basePath, specPath, diffOptions)
	if !diffResult.OK {
		return cli.Exit(2, cli.Wrap(core.NewError(diffResult.Error()), "error:"))
	}
	result := diffResult.Value.(*sdk.DiffResult)

	if result.Breaking || (failOnWarn && result.HasWarnings) {
		cli.Print("%s %s\n", sdkErrorStyle.Render("Breaking changes"), result.Summary)
		for _, change := range result.Changes {
			cli.Print("  - %s\n", change)
		}
		for _, warning := range result.Warnings {
			cli.Print("  - warning: %s\n", warning)
		}
		return cli.Exit(1, cli.Err("%s", result.Summary))
	}

	for _, warning := range result.Warnings {
		cli.Print("  - warning: %s\n", warning)
	}
	cli.Print("%s %s\n", sdkSuccessStyle.Render("OK"), result.Summary)
	return core.Ok(nil)
}

func runSDKValidate(specPath string) core.Result {
	projectDirResult := ax.Getwd()
	if !projectDirResult.OK {
		return core.Fail(core.E("sdk.Validate", "failed to get working directory", core.NewError(projectDirResult.Error())))
	}
	projectDir := projectDirResult.Value.(string)

	return runSDKValidateInDir(context.Background(), projectDir, specPath)
}

func runSDKValidateInDir(ctx context.Context, projectDir, specPath string) core.Result {
	configResult := sdkcfg.LoadProjectConfig(storage.Local, projectDir)
	if !configResult.OK {
		return core.Fail(core.E("sdk.Validate", "failed to load sdk config", core.NewError(configResult.Error())))
	}
	config := configResult.Value.(*sdk.Config)
	if specPath != "" {
		config.Spec = specPath
	}

	s := sdk.New(projectDir, config)

	cli.Print("%s %s\n", sdkHeaderStyle.Render("SDK"), "Validating OpenAPI spec")

	detectedPathResult := s.ValidateSpec(ctx)
	if !detectedPathResult.OK {
		cli.Print("%s %v\n", sdkErrorStyle.Render("error:"), detectedPathResult.Error())
		return detectedPathResult
	}
	detectedPath := detectedPathResult.Value.(string)

	cli.Print("  %s %s\n", "spec:", sdkDimStyle.Render(detectedPath))
	cli.Print("%s %s\n", sdkSuccessStyle.Render("OK"), "OpenAPI spec is valid")
	return core.Ok(nil)
}

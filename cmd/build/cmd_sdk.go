// cmd_sdk.go implements SDK generation from OpenAPI specifications.
//
// Generates typed API clients for TypeScript, Python, Go, and PHP
// from OpenAPI/Swagger specifications.

package buildcmd

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/internal/sdkcfg"
	"dappco.re/go/build/pkg/sdk"
	storage "dappco.re/go/build/pkg/storage"
)

// runBuildSDK handles the `core build sdk` command.
func runBuildSDK(ctx context.Context, specPath, lang, version string, dryRun bool, skipUnavailable bool) core.Result {
	projectDirResult := ax.Getwd()
	if !projectDirResult.OK {
		return core.Fail(core.E("build.SDK", "failed to get working directory", core.NewError(projectDirResult.Error())))
	}

	return runBuildSDKInDir(ctx, projectDirResult.Value.(string), specPath, lang, version, dryRun, skipUnavailable)
}

func runBuildSDKInDir(ctx context.Context, projectDir, specPath, lang, version string, dryRun bool, skipUnavailable bool) core.Result {
	configResult := sdkcfg.LoadProjectConfig(storage.Local, projectDir)
	if !configResult.OK {
		return core.Fail(core.E("build.SDK", "failed to load sdk config", core.NewError(configResult.Error())))
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

	cli.Print("%s %s\n", buildHeaderStyle.Render("SDK"), "Generating SDK")
	if dryRun {
		cli.Print("  %s\n", buildDimStyle.Render("dry run mode"))
	}
	cli.Blank()

	// Validate the spec before generating anything.
	detectedSpecResult := s.ValidateSpec(ctx)
	if !detectedSpecResult.OK {
		cli.Print("%s %v\n", buildErrorStyle.Render("error"), detectedSpecResult.Error())
		return detectedSpecResult
	}
	detectedSpec := detectedSpecResult.Value.(string)
	cli.Print("  %s %s\n", "spec", buildTargetStyle.Render(detectedSpec))

	if dryRun {
		if lang != "" {
			cli.Print("  %s %s\n", "language", buildTargetStyle.Render(lang))
		} else {
			cli.Print("  %s %s\n", "languages", buildTargetStyle.Render(core.Join(", ", resolvedConfig.Languages...)))
		}
		cli.Blank()
		cli.Print("%s %s\n", buildSuccessStyle.Render("OK"), "Would generate SDK")
		return core.Ok(nil)
	}

	if lang != "" {
		// Generate single language
		resultResult := s.GenerateLanguageWithStatus(ctx, lang)
		if !resultResult.OK {
			cli.Print("%s %v\n", buildErrorStyle.Render("error"), resultResult.Error())
			return resultResult
		}
		result := resultResult.Value.(sdk.LanguageResult)
		if result.Skipped {
			cli.Print("  %s %s\n", "Skipped:", buildTargetStyle.Render(result.Language))
		} else {
			cli.Print("  %s %s\n", "generated", buildTargetStyle.Render(result.Language))
		}
	} else {
		// Generate all
		resultsResult := s.GenerateWithStatus(ctx)
		if !resultsResult.OK {
			cli.Print("%s %v\n", buildErrorStyle.Render("error"), resultsResult.Error())
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
			cli.Print("  %s %s\n", "generated", buildTargetStyle.Render(core.Join(", ", generated...)))
		}
		if len(skipped) > 0 {
			cli.Print("  %s %s\n", "Skipped:", buildTargetStyle.Render(core.Join(", ", skipped...)))
		}
	}

	cli.Blank()
	cli.Print("%s %s\n", buildSuccessStyle.Render("Success"), "SDK generation complete")
	return core.Ok(nil)
}

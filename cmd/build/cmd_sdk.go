// cmd_sdk.go implements SDK generation from OpenAPI specifications.
//
// Generates typed API clients for TypeScript, Python, Go, and PHP
// from OpenAPI/Swagger specifications.

package buildcmd

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/sdkcfg"
	"dappco.re/go/build/pkg/sdk"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/i18n"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// runBuildSDK handles the `core build sdk` command.
func runBuildSDK(ctx context.Context, specPath, lang, version string, dryRun bool, skipUnavailable bool) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.SDK", "failed to get working directory", err)
	}

	return runBuildSDKInDir(ctx, projectDir, specPath, lang, version, dryRun, skipUnavailable)
}

func runBuildSDKInDir(ctx context.Context, projectDir, specPath, lang, version string, dryRun bool, skipUnavailable bool) error {
	config, err := sdkcfg.LoadProjectConfig(io.Local, projectDir)
	if err != nil {
		return coreerr.E("build.SDK", "failed to load sdk config", err)
	}
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

	cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.sdk.label")), i18n.T("cmd.build.sdk.generating"))
	if dryRun {
		cli.Print("  %s\n", buildDimStyle.Render(i18n.T("cmd.build.sdk.dry_run_mode")))
	}
	cli.Blank()

	// Validate the spec before generating anything.
	detectedSpec, err := s.ValidateSpec(ctx)
	if err != nil {
		cli.Print("%s %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), err)
		return err
	}
	cli.Print("  %s %s\n", i18n.T("common.label.spec"), buildTargetStyle.Render(detectedSpec))

	if dryRun {
		if lang != "" {
			cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.language_label"), buildTargetStyle.Render(lang))
		} else {
			cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.languages_label"), buildTargetStyle.Render(core.Join(", ", resolvedConfig.Languages...)))
		}
		cli.Blank()
		cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("cmd.build.label.ok")), i18n.T("cmd.build.sdk.would_generate"))
		return nil
	}

	if lang != "" {
		// Generate single language
		result, err := s.GenerateLanguageWithStatus(ctx, lang)
		if err != nil {
			cli.Print("%s %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), err)
			return err
		}
		if result.Skipped {
			cli.Print("  %s %s\n", "Skipped:", buildTargetStyle.Render(result.Language))
		} else {
			cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.generated_label"), buildTargetStyle.Render(result.Language))
		}
	} else {
		// Generate all
		results, err := s.GenerateWithStatus(ctx)
		if err != nil {
			cli.Print("%s %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), err)
			return err
		}
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
			cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.generated_label"), buildTargetStyle.Render(core.Join(", ", generated...)))
		}
		if len(skipped) > 0 {
			cli.Print("  %s %s\n", "Skipped:", buildTargetStyle.Render(core.Join(", ", skipped...)))
		}
	}

	cli.Blank()
	cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("common.label.success")), i18n.T("cmd.build.sdk.complete"))
	return nil
}

// cmd_sdk.go implements SDK generation from OpenAPI specifications.
//
// Generates typed API clients for TypeScript, Python, Go, and PHP
// from OpenAPI/Swagger specifications.

package buildcmd

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/sdk"
	"dappco.re/go/core/i18n"
	coreerr "dappco.re/go/core/log"
	"forge.lthn.ai/core/cli/pkg/cli"
)

// runBuildSDK handles the `core build sdk` command.
func runBuildSDK(ctx context.Context, specPath, lang, version string, dryRun bool) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.SDK", "failed to get working directory", err)
	}

	return runBuildSDKInDir(ctx, projectDir, specPath, lang, version, dryRun)
}

func runBuildSDKInDir(ctx context.Context, projectDir, specPath, lang, version string, dryRun bool) error {
	// Load config
	config := sdk.DefaultConfig()
	if specPath != "" {
		config.Spec = specPath
	}

	s := sdk.New(projectDir, config)
	if version != "" {
		s.SetVersion(version)
	}

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
			cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.languages_label"), buildTargetStyle.Render(core.Join(", ", config.Languages...)))
		}
		cli.Blank()
		cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("cmd.build.label.ok")), i18n.T("cmd.build.sdk.would_generate"))
		return nil
	}

	if lang != "" {
		// Generate single language
		if err := s.GenerateLanguage(ctx, lang); err != nil {
			cli.Print("%s %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), err)
			return err
		}
		cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.generated_label"), buildTargetStyle.Render(lang))
	} else {
		// Generate all
		if err := s.Generate(ctx); err != nil {
			cli.Print("%s %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), err)
			return err
		}
		cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.generated_label"), buildTargetStyle.Render(core.Join(", ", config.Languages...)))
	}

	cli.Blank()
	cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("common.label.success")), i18n.T("cmd.build.sdk.complete"))
	return nil
}

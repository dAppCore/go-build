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

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/internal/cmdutil"
	"dappco.re/go/core/build/pkg/sdk"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/i18n"
	coreerr "dappco.re/go/core/log"
)

func init() {
	cli.RegisterCommands(AddSDKCommands)
}

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
	c.Command("sdk", core.Command{
		Description: "cmd.sdk.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runSDKGenerate(
				cmdutil.ContextOrBackground(),
				cmdutil.OptionString(opts, "spec"),
				cmdutil.OptionString(opts, "lang"),
				cmdutil.OptionString(opts, "version"),
				cmdutil.OptionBool(opts, "dry-run"),
			))
		},
	})

	c.Command("sdk/diff", core.Command{
		Description: "cmd.sdk.diff.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runSDKDiff(
				cmdutil.OptionString(opts, "base"),
				cmdutil.OptionString(opts, "spec"),
			))
		},
	})

	c.Command("sdk/validate", core.Command{
		Description: "cmd.sdk.validate.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runSDKValidate(
				cmdutil.OptionString(opts, "spec"),
			))
		},
	})
}

func runSDKGenerate(ctx context.Context, specPath, lang, version string, dryRun bool) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("sdk.Generate", "failed to get working directory", err)
	}

	return runSDKGenerateInDir(ctx, projectDir, specPath, lang, version, dryRun)
}

func runSDKGenerateInDir(ctx context.Context, projectDir, specPath, lang, version string, dryRun bool) error {
	config := sdk.DefaultConfig()
	if specPath != "" {
		config.Spec = specPath
	}

	s := sdk.New(projectDir, config)
	if version != "" {
		s.SetVersion(version)
	}

	cli.Print("%s %s\n", sdkHeaderStyle.Render(i18n.T("cmd.build.sdk.label")), i18n.T("cmd.build.sdk.generating"))
	if dryRun {
		cli.Print("  %s\n", sdkDimStyle.Render(i18n.T("cmd.build.sdk.dry_run_mode")))
	}
	cli.Blank()

	detectedSpec, err := s.ValidateSpec(ctx)
	if err != nil {
		cli.Print("%s %v\n", sdkErrorStyle.Render(i18n.T("common.label.error")), err)
		return err
	}
	cli.Print("  %s %s\n", i18n.T("common.label.spec"), sdkTargetStyle.Render(detectedSpec))

	if dryRun {
		if lang != "" {
			cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.language_label"), sdkTargetStyle.Render(lang))
		} else {
			cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.languages_label"), sdkTargetStyle.Render(core.Join(", ", config.Languages...)))
		}
		cli.Blank()
		cli.Print("%s %s\n", sdkSuccessStyle.Render(i18n.T("cmd.build.label.ok")), i18n.T("cmd.build.sdk.would_generate"))
		return nil
	}

	if lang != "" {
		if err := s.GenerateLanguage(ctx, lang); err != nil {
			cli.Print("%s %v\n", sdkErrorStyle.Render(i18n.T("common.label.error")), err)
			return err
		}
		cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.generated_label"), sdkTargetStyle.Render(lang))
	} else {
		if err := s.Generate(ctx); err != nil {
			cli.Print("%s %v\n", sdkErrorStyle.Render(i18n.T("common.label.error")), err)
			return err
		}
		cli.Print("  %s %s\n", i18n.T("cmd.build.sdk.generated_label"), sdkTargetStyle.Render(core.Join(", ", config.Languages...)))
	}

	cli.Blank()
	cli.Print("%s %s\n", sdkSuccessStyle.Render(i18n.T("common.label.success")), i18n.T("cmd.build.sdk.complete"))
	return nil
}

func runSDKDiff(basePath, specPath string) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("sdk.Diff", "failed to get working directory", err)
	}

	if specPath == "" {
		s := sdk.New(projectDir, nil)
		specPath, err = s.DetectSpec()
		if err != nil {
			return err
		}
	}

	if basePath == "" {
		return coreerr.E("sdk.Diff", i18n.T("cmd.sdk.diff.error.base_required"), nil)
	}

	cli.Print("%s %s\n", sdkHeaderStyle.Render(i18n.T("cmd.sdk.diff.label")), i18n.ProgressSubject("check", "breaking changes"))
	cli.Print("  %s %s\n", i18n.T("cmd.sdk.diff.base_label"), sdkDimStyle.Render(basePath))
	cli.Print("  %s %s\n", i18n.Label("current"), sdkDimStyle.Render(specPath))
	cli.Blank()

	result, err := sdk.Diff(basePath, specPath)
	if err != nil {
		return cli.Exit(2, cli.Wrap(err, i18n.Label("error")))
	}

	if result.Breaking {
		cli.Print("%s %s\n", sdkErrorStyle.Render(i18n.T("cmd.sdk.diff.breaking")), result.Summary)
		for _, change := range result.Changes {
			cli.Print("  - %s\n", change)
		}
		return cli.Exit(1, cli.Err("%s", result.Summary))
	}

	cli.Print("%s %s\n", sdkSuccessStyle.Render(i18n.T("cmd.sdk.label.ok")), result.Summary)
	return nil
}

func runSDKValidate(specPath string) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("sdk.Validate", "failed to get working directory", err)
	}

	return runSDKValidateInDir(context.Background(), projectDir, specPath)
}

func runSDKValidateInDir(ctx context.Context, projectDir, specPath string) error {
	s := sdk.New(projectDir, &sdk.Config{Spec: specPath})

	cli.Print("%s %s\n", sdkHeaderStyle.Render(i18n.T("cmd.sdk.label.sdk")), i18n.T("cmd.sdk.validate.validating"))

	detectedPath, err := s.ValidateSpec(ctx)
	if err != nil {
		cli.Print("%s %v\n", sdkErrorStyle.Render(i18n.Label("error")), err)
		return err
	}

	cli.Print("  %s %s\n", i18n.Label("spec"), sdkDimStyle.Render(detectedPath))
	cli.Print("%s %s\n", sdkSuccessStyle.Render(i18n.T("cmd.sdk.label.ok")), i18n.T("cmd.sdk.validate.valid"))
	return nil
}

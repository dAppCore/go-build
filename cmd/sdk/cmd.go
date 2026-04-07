// Package sdkcmd provides SDK validation and API compatibility commands.
//
// Commands:
//   - sdk diff: check for breaking API changes between spec versions
//   - sdk validate: validate OpenAPI spec syntax
//
// For SDK generation, use: core build sdk
package sdkcmd

import (
	"context"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/sdk"
	"dappco.re/go/core/i18n"
	coreerr "dappco.re/go/core/log"
	"dappco.re/go/core/cli/pkg/cli"
)

func init() {
	cli.RegisterCommands(AddSDKCommands)
}

// SDK styles (aliases to shared)
var (
	sdkHeaderStyle  = cli.TitleStyle
	sdkSuccessStyle = cli.SuccessStyle
	sdkErrorStyle   = cli.ErrorStyle
	sdkDimStyle     = cli.DimStyle
)

var sdkCmd = &cli.Command{
	Use: "sdk",
}

var diffBasePath string
var diffSpecPath string

var sdkDiffCmd = &cli.Command{
	Use: "diff",
	RunE: func(cmd *cli.Command, args []string) error {
		return runSDKDiff(diffBasePath, diffSpecPath)
	},
}

var validateSpecPath string

var sdkValidateCmd = &cli.Command{
	Use: "validate",
	RunE: func(cmd *cli.Command, args []string) error {
		return runSDKValidate(validateSpecPath)
	},
}

func setSDKI18n() {
	sdkCmd.Short = i18n.T("cmd.sdk.short")
	sdkCmd.Long = i18n.T("cmd.sdk.long")
	sdkDiffCmd.Short = i18n.T("cmd.sdk.diff.short")
	sdkDiffCmd.Long = i18n.T("cmd.sdk.diff.long")
	sdkValidateCmd.Short = i18n.T("cmd.sdk.validate.short")
	sdkValidateCmd.Long = i18n.T("cmd.sdk.validate.long")
}

// AddSDKCommands registers the 'sdk' command and all subcommands.
//
// sdkcmd.AddSDKCommands(root)
func AddSDKCommands(root *cli.Command) {
	setSDKI18n()

	// sdk diff flags
	sdkDiffCmd.Flags().StringVar(&diffBasePath, "base", "", i18n.T("cmd.sdk.diff.flag.base"))
	sdkDiffCmd.Flags().StringVar(&diffSpecPath, "spec", "", i18n.T("cmd.sdk.diff.flag.spec"))

	// sdk validate flags
	sdkValidateCmd.Flags().StringVar(&validateSpecPath, "spec", "", i18n.T("common.flag.spec"))

	// Add subcommands
	sdkCmd.AddCommand(sdkDiffCmd)
	sdkCmd.AddCommand(sdkValidateCmd)

	root.AddCommand(sdkCmd)
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

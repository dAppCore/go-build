// Package sdkcmd provides SDK validation and API compatibility commands.
//
// Commands:
//   - sdk diff: check for breaking API changes between spec versions
//   - sdk validate: validate OpenAPI spec syntax
//
// For SDK generation, use: core build sdk
package sdkcmd

import (
	"fmt"
	"os"

	"forge.lthn.ai/core/go-build/pkg/sdk"
	"forge.lthn.ai/core/cli/pkg/cli"
	"forge.lthn.ai/core/go-i18n"
	coreerr "forge.lthn.ai/core/go-log"
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
	Use:   "sdk",
	Short: i18n.T("cmd.sdk.short"),
	Long:  i18n.T("cmd.sdk.long"),
}

var diffBasePath string
var diffSpecPath string

var sdkDiffCmd = &cli.Command{
	Use:   "diff",
	Short: i18n.T("cmd.sdk.diff.short"),
	Long:  i18n.T("cmd.sdk.diff.long"),
	RunE: func(cmd *cli.Command, args []string) error {
		return runSDKDiff(diffBasePath, diffSpecPath)
	},
}

var validateSpecPath string

var sdkValidateCmd = &cli.Command{
	Use:   "validate",
	Short: i18n.T("cmd.sdk.validate.short"),
	Long:  i18n.T("cmd.sdk.validate.long"),
	RunE: func(cmd *cli.Command, args []string) error {
		return runSDKValidate(validateSpecPath)
	},
}

// AddSDKCommands registers the 'sdk' command and all subcommands.
func AddSDKCommands(root *cli.Command) {
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
	projectDir, err := os.Getwd()
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

	fmt.Printf("%s %s\n", sdkHeaderStyle.Render(i18n.T("cmd.sdk.diff.label")), i18n.ProgressSubject("check", "breaking changes"))
	fmt.Printf("  %s %s\n", i18n.T("cmd.sdk.diff.base_label"), sdkDimStyle.Render(basePath))
	fmt.Printf("  %s %s\n", i18n.Label("current"), sdkDimStyle.Render(specPath))
	fmt.Println()

	result, err := sdk.Diff(basePath, specPath)
	if err != nil {
		return cli.Exit(2, cli.Wrap(err, i18n.Label("error")))
	}

	if result.Breaking {
		fmt.Printf("%s %s\n", sdkErrorStyle.Render(i18n.T("cmd.sdk.diff.breaking")), result.Summary)
		for _, change := range result.Changes {
			fmt.Printf("  - %s\n", change)
		}
		return cli.Exit(1, cli.Err("%s", result.Summary))
	}

	fmt.Printf("%s %s\n", sdkSuccessStyle.Render(i18n.T("cmd.sdk.label.ok")), result.Summary)
	return nil
}

func runSDKValidate(specPath string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return coreerr.E("sdk.Validate", "failed to get working directory", err)
	}

	s := sdk.New(projectDir, &sdk.Config{Spec: specPath})

	fmt.Printf("%s %s\n", sdkHeaderStyle.Render(i18n.T("cmd.sdk.label.sdk")), i18n.T("cmd.sdk.validate.validating"))

	detectedPath, err := s.DetectSpec()
	if err != nil {
		fmt.Printf("%s %v\n", sdkErrorStyle.Render(i18n.Label("error")), err)
		return err
	}

	fmt.Printf("  %s %s\n", i18n.Label("spec"), sdkDimStyle.Render(detectedPath))
	fmt.Printf("%s %s\n", sdkSuccessStyle.Render(i18n.T("cmd.sdk.label.ok")), i18n.T("cmd.sdk.validate.valid"))
	return nil
}

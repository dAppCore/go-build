// Package ci provides release lifecycle commands for CI/CD pipelines.
//
// Commands:
//   - ci init: scaffold release config
//   - ci changelog: generate changelog from git history
//   - ci version: show determined version
//   - ci publish: publish pre-built artifacts (dry-run by default)
//
// Configuration via .core/release.yaml.
package ci

import (
	"forge.lthn.ai/core/cli/pkg/cli"
)

func init() {
	cli.RegisterCommands(AddCICommands)
}

// AddCICommands registers the 'ci' command and all subcommands.
func AddCICommands(root *cli.Command) {
	root.AddCommand(ciCmd)
}

// Package ci registers release lifecycle commands.
//
// ci.AddCICommands(root)
package ci

import (
	"forge.lthn.ai/core/cli/pkg/cli"
)

func init() {
	cli.RegisterCommands(AddCICommands)
}

// AddCICommands registers the 'ci' command and all subcommands.
//
// ci.AddCICommands(root)
func AddCICommands(root *cli.Command) {
	setCII18n()
	initCIFlags()
	root.AddCommand(ciCmd)
}

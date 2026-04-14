// Package ci registers release lifecycle commands.
//
// ci.AddCICommands(root)
package ci

import (
	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
)

func init() {
	cli.RegisterCommands(AddCICommands)
}

// AddCICommands registers the 'ci' command and all subcommands.
//
// ci.AddCICommands(root)
func AddCICommands(c *core.Core) {
	registerCICommands(c)
}

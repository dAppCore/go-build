// Package ci registers release lifecycle commands.
//
// ci.AddCICommands(root)
package ci

import (
	"dappco.re/go/core"
)

// AddCICommands registers the 'ci' command and all subcommands.
//
// ci.AddCICommands(root)
func AddCICommands(c *core.Core) {
	registerCICommands(c)
}

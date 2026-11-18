// Package ci registers release lifecycle commands.
//
// ci.AddCICommands(root)
package ci

import (
	"github.com/dappcore/go"
)

// AddCICommands registers the 'ci' command and all subcommands.
//
// ci.AddCICommands(root)
func AddCICommands(c *core.Core) core.Result {
	return registerCICommands(c)
}

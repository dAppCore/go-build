package ci

import (
	core "dappco.re/go"
)

// noopCIAction is a placeholder executable action used to pre-occupy command
// paths so AddCICommands' partial-failure branches can be observed.
func noopCIAction(core.Options) core.Result { return core.Ok(nil) }

func TestCmd_AddCICommands_Good(t *core.T) {
	c := core.New()

	result := AddCICommands(c)
	core.AssertTrue(t, result.OK)
	for _, path := range []string{"ci", "ci/init", "ci/changelog", "ci/version"} {
		core.AssertTrue(t, c.Command(path).OK, "expected command "+path+" registered")
	}
	cmd := c.Command("ci").Value.(*core.Command)
	core.AssertNotNil(t, cmd.Action)
}

func TestCmd_AddCICommands_Bad(t *core.T) {
	// Failure at the first step: `ci` is already an executable command, so
	// registration aborts and the subcommands are never reached.
	c := core.New()
	core.AssertTrue(t, c.Command("ci", core.Command{Action: noopCIAction}).OK)

	result := AddCICommands(c)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "ci")
	core.AssertContains(t, result.Error(), "already registered")
	core.AssertFalse(t, c.Command("ci/init").OK)
}

func TestCmd_AddCICommands_Ugly(t *core.T) {
	// Edge case: every registration step can fail independently. A clash on any
	// single command path aborts the whole registration.
	for _, path := range []string{"ci", "ci/init", "ci/changelog", "ci/version"} {
		c := core.New()
		core.AssertTrue(t, c.Command(path, core.Command{Action: noopCIAction}).OK)
		result := AddCICommands(c)
		core.AssertFalse(t, result.OK, "clash on "+path+" should abort registration")
		core.AssertContains(t, result.Error(), path)
	}
}

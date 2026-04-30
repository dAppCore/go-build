package buildcmd

import core "dappco.re/go"

func TestCmdBuild_AddBuildCommands_Good(t *core.T) {
	c := core.New()
	AddBuildCommands(c)
	core.AssertNotNil(t, c)
}

func TestCmdBuild_AddBuildCommands_Bad(t *core.T) {
	c := core.New()
	core.AssertNotPanics(t, func() {
		AddBuildCommands(c)
	})
	core.AssertNotNil(t, c)
}

func TestCmdBuild_AddBuildCommands_Ugly(t *core.T) {
	c := core.New()
	AddBuildCommands(c)
	AddBuildCommands(core.New())
	core.AssertNotNil(t, c)
}

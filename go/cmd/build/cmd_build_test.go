package buildcmd

import (
	core "dappco.re/go"
)

func TestCmdBuild_AddBuildCommands_Good(t *core.T) {
	c := core.New()
	result := AddBuildCommands(c)
	core.AssertTrue(t, result.OK)
	// All top-level build commands and the registered subcommands are present.
	for _, path := range []string{
		"build", "build/from-path", "build/pwa", "build/sdk",
		"build/apple", "build/image", "build/installers", "build/release",
		"service", "build/workflow",
	} {
		core.AssertTrue(t, c.Command(path).OK, "expected command "+path+" registered")
	}
}

func TestCmdBuild_AddBuildCommands_Bad(t *core.T) {
	// Registration aborts if the root `build` command is already taken by an
	// executable command.
	c := core.New()
	core.AssertTrue(t, c.Command("build", core.Command{
		Action: func(core.Options) core.Result { return core.Ok(nil) },
	}).OK)
	result := AddBuildCommands(c)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "already registered")
}

func TestCmdBuild_AddBuildCommands_Ugly(t *core.T) {
	// Edge case: drive the registered Action closures that validate their inputs.
	// `build/from-path` with no path and `build/pwa` with neither path nor url
	// both fail fast with their required-input errors.
	c := core.New()
	core.AssertTrue(t, AddBuildCommands(c).OK)
	captureBuildStdout(t)

	fromPath := c.Command("build/from-path").Value.(*core.Command).Run(core.NewOptions())
	core.AssertFalse(t, fromPath.OK)
	core.AssertContains(t, fromPath.Error(), "--path flag is required")

	pwa := c.Command("build/pwa").Value.(*core.Command).Run(core.NewOptions())
	core.AssertFalse(t, pwa.OK)
	core.AssertContains(t, pwa.Error(), "either --path or --url is required")
}

// TestCmdBuild_resolveNoSign covers the no-sign precedence logic.
func TestCmdBuild_resolveNoSign_Good(t *core.T) {
	// Explicit --no-sign always wins.
	core.AssertTrue(t, resolveNoSign(true, true, true))
}

func TestCmdBuild_resolveNoSign_Bad(t *core.T) {
	// --sign=false (explicitly set) implies no-sign.
	core.AssertTrue(t, resolveNoSign(false, false, true))
}

func TestCmdBuild_resolveNoSign_Ugly(t *core.T) {
	// Edge case: signing left at its default (set but enabled) keeps signing on.
	core.AssertFalse(t, resolveNoSign(false, true, true))
	// And entirely unset also keeps signing on.
	core.AssertFalse(t, resolveNoSign(false, true, false))
}

// TestCmdBuild_resolvePackageOutputs covers the --package convenience flag
// fanning out to archive + checksum outputs.
func TestCmdBuild_resolvePackageOutputs_Good(t *core.T) {
	// --package not set: archive/checksum pass through unchanged.
	archive, checksum := resolvePackageOutputs(false, false, true, true, false, true)
	core.AssertTrue(t, archive)
	core.AssertFalse(t, checksum)
}

func TestCmdBuild_resolvePackageOutputs_Bad(t *core.T) {
	// --package=true with neither archive nor checksum explicitly set enables
	// both.
	archive, checksum := resolvePackageOutputs(true, true, false, false, false, false)
	core.AssertTrue(t, archive)
	core.AssertTrue(t, checksum)
}

func TestCmdBuild_resolvePackageOutputs_Ugly(t *core.T) {
	// Edge case: an explicit archive/checksum value overrides --package.
	archive, checksum := resolvePackageOutputs(true, true, false, true, true, true)
	core.AssertFalse(t, archive)
	core.AssertTrue(t, checksum)
}

// TestCmdBuild_runBuildFromPathAction drives the build/from-path success-guard:
// a non-directory path is rejected with a directory error.
func TestCmdBuild_runBuildFromPathAction(t *core.T) {
	c := core.New()
	core.AssertTrue(t, AddBuildCommands(c).OK)
	captureBuildStdout(t)

	result := c.Command("build/from-path").Value.(*core.Command).Run(core.NewOptions(
		core.Option{Key: buildPathOptionKey, Value: "/definitely/not/a/real/dir"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "must be a directory")
}

// TestCmdBuild_buildSDKActionContext confirms the build/sdk action wires through
// to SDK generation (which fails fast with no spec in the working directory).
func TestCmdBuild_buildSDKActionContext(t *core.T) {
	c := core.New()
	core.AssertTrue(t, AddBuildCommands(c).OK)
	captureBuildStdout(t)

	result := c.Command("build/sdk").Value.(*core.Command).Run(core.NewOptions(
		core.Option{Key: "dry-run", Value: true},
	))
	// No OpenAPI spec exists in the test working directory, so the SDK action
	// reports a detection failure rather than generating anything.
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "no OpenAPI spec found")
}

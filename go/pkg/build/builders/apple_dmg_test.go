package builders

import (
	"context"

	core "dappco.re/go"
	coreio "dappco.re/go/build/pkg/storage"
)

func TestAppleDmg_AppleBuilder_CreateDMG_Good(t *core.T) {
	fs := coreio.NewMemoryMedium()
	runner := &recordingAppleRunner{}
	builder := NewAppleBuilder(WithAppleCommandRunner(runner))

	result := builder.CreateDMG(context.Background(), fs, "dist/Core.app", AppleDMGConfig{OutputPath: "dist/Core.dmg", VolumeName: "Core"})
	core.RequireTrue(t, result.OK)
	core.AssertLen(t, runner.calls, 4)
	core.AssertTrue(t, fs.IsFile("dist/Core.dmg"))
}

func TestAppleDmg_AppleBuilder_CreateDMG_Bad(t *core.T) {
	builder := NewAppleBuilder(WithAppleCommandRunner(&recordingAppleRunner{}))
	result := builder.CreateDMG(context.Background(), coreio.NewMemoryMedium(), "", AppleDMGConfig{OutputPath: "dist/Core.dmg"})
	core.AssertFalse(t, result.OK)
}

func TestAppleDmg_AppleBuilder_CreateDMG_Ugly(t *core.T) {
	fs := coreio.NewMemoryMedium()
	builder := NewAppleBuilder(WithAppleCommandRunner(&recordingAppleRunner{}))

	result := builder.CreateDMG(context.Background(), fs, "dist/Edge.app", AppleDMGConfig{OutputPath: "Core.dmg"})
	core.RequireTrue(t, result.OK)
	core.AssertTrue(t, fs.IsFile("Core.dmg"))
}

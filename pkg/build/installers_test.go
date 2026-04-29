package build

import core "dappco.re/go"

func TestInstallers_GenerateInstallerScript_Good(t *core.T) {
	script, err := GenerateInstallerScript(VariantCI, "v1.2.3", "dappcore/core")
	core.RequireNoError(t, err)
	core.AssertContains(t, script, "v1.2.3")
}

func TestInstallers_GenerateInstallerScript_Bad(t *core.T) {
	script, err := GenerateInstallerScript(InstallerVariant("missing"), "v1.2.3", "dappcore/core")
	core.AssertError(t, err)
	core.AssertEqual(t, "", script)
}

func TestInstallers_GenerateInstallerScript_Ugly(t *core.T) {
	script, err := GenerateInstallerScript(VariantGo, "v1.2.3", "dappcore/core.git")
	core.RequireNoError(t, err)
	core.AssertContains(t, script, "core")
}

func TestInstallers_GenerateInstaller_Good(t *core.T) {
	script, err := GenerateInstaller(VariantFull, "v1.2.3", "dappcore/core")
	core.RequireNoError(t, err)
	core.AssertContains(t, script, "v1.2.3")
}

func TestInstallers_GenerateInstaller_Bad(t *core.T) {
	script, err := GenerateInstaller(VariantCI, "bad version!", "dappcore/core")
	core.AssertError(t, err)
	core.AssertEqual(t, "", script)
}

func TestInstallers_GenerateInstaller_Ugly(t *core.T) {
	script, err := GenerateInstaller(VariantAgentic, "v1.2.3", "")
	core.RequireNoError(t, err)
	core.AssertContains(t, script, "v1.2.3")
}

func TestInstallers_GenerateAllInstallerScripts_Good(t *core.T) {
	scripts, err := GenerateAllInstallerScripts("v1.2.3", "dappcore/core")
	core.RequireNoError(t, err)
	core.AssertContains(t, scripts, "setup.sh")
}

func TestInstallers_GenerateAllInstallerScripts_Bad(t *core.T) {
	scripts, err := GenerateAllInstallerScripts("bad version!", "dappcore/core")
	core.AssertError(t, err)
	core.AssertNil(t, scripts)
}

func TestInstallers_GenerateAllInstallerScripts_Ugly(t *core.T) {
	scripts, err := GenerateAllInstallerScripts("v1.2.3", "")
	core.RequireNoError(t, err)
	core.AssertContains(t, scripts, "agent.sh")
}

func TestInstallers_GenerateAll_Good(t *core.T) {
	scripts, err := GenerateAll("v1.2.3", "dappcore/core")
	core.RequireNoError(t, err)
	core.AssertContains(t, scripts, "go.sh")
}

func TestInstallers_GenerateAll_Bad(t *core.T) {
	scripts, err := GenerateAll("bad version!", "dappcore/core")
	core.AssertError(t, err)
	core.AssertNil(t, scripts)
}

func TestInstallers_GenerateAll_Ugly(t *core.T) {
	scripts, err := GenerateAll("v1.2.3", "owner/repo.git")
	core.RequireNoError(t, err)
	core.AssertContains(t, scripts["ci.sh"], "repo")
}

func TestInstallers_InstallerVariants_Good(t *core.T) {
	variants := InstallerVariants()
	core.AssertContains(t, variants, VariantFull)
	core.AssertContains(t, variants, VariantCI)
}

func TestInstallers_InstallerVariants_Bad(t *core.T) {
	variants := InstallerVariants()
	variants[0] = InstallerVariant("mutated")
	core.AssertNotEqual(t, InstallerVariant("mutated"), InstallerVariants()[0])
}

func TestInstallers_InstallerVariants_Ugly(t *core.T) {
	variants := InstallerVariants()
	core.AssertEqual(t, VariantDev, variants[len(variants)-1])
	core.AssertLen(t, variants, 6)
}

func TestInstallers_InstallerOutputName_Good(t *core.T) {
	name := InstallerOutputName(VariantFull)
	core.AssertEqual(t, "setup.sh", name)
	core.AssertContains(t, name, ".sh")
}

func TestInstallers_InstallerOutputName_Bad(t *core.T) {
	name := InstallerOutputName(InstallerVariant("missing"))
	core.AssertEqual(t, "", name)
	core.AssertEmpty(t, name)
}

func TestInstallers_InstallerOutputName_Ugly(t *core.T) {
	name := InstallerOutputName(VariantAgentic)
	core.AssertEqual(t, "agent.sh", name)
	core.AssertContains(t, name, "agent")
}

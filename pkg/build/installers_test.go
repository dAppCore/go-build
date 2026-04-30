package build

import core "dappco.re/go"

func TestInstallers_GenerateInstallerScript_Good(t *core.T) {
	result := GenerateInstallerScript(VariantCI, "v1.2.3", "dappcore/core")
	core.RequireTrue(t, result.OK)
	script := result.Value.(string)
	core.AssertContains(t, script, "v1.2.3")
}

func TestInstallers_GenerateInstallerScript_Bad(t *core.T) {
	result := GenerateInstallerScript(InstallerVariant("missing"), "v1.2.3", "dappcore/core")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "unknown")
}

func TestInstallers_GenerateInstallerScript_Ugly(t *core.T) {
	result := GenerateInstallerScript(VariantGo, "v1.2.3", "dappcore/core.git")
	core.RequireTrue(t, result.OK)
	script := result.Value.(string)
	core.AssertContains(t, script, "core")
}

func TestInstallers_GenerateInstaller_Good(t *core.T) {
	result := GenerateInstaller(VariantFull, "v1.2.3", "dappcore/core")
	core.RequireTrue(t, result.OK)
	script := result.Value.(string)
	core.AssertContains(t, script, "v1.2.3")
}

func TestInstallers_GenerateInstaller_Bad(t *core.T) {
	result := GenerateInstaller(VariantCI, "bad version!", "dappcore/core")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "version")
}

func TestInstallers_GenerateInstaller_Ugly(t *core.T) {
	result := GenerateInstaller(VariantAgentic, "v1.2.3", "")
	core.RequireTrue(t, result.OK)
	script := result.Value.(string)
	core.AssertContains(t, script, "v1.2.3")
}

func TestInstallers_GenerateAllInstallerScripts_Good(t *core.T) {
	result := GenerateAllInstallerScripts("v1.2.3", "dappcore/core")
	core.RequireTrue(t, result.OK)
	scripts := result.Value.(map[string]string)
	core.AssertContains(t, scripts, "setup.sh")
}

func TestInstallers_GenerateAllInstallerScripts_Bad(t *core.T) {
	result := GenerateAllInstallerScripts("bad version!", "dappcore/core")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "version")
}

func TestInstallers_GenerateAllInstallerScripts_Ugly(t *core.T) {
	result := GenerateAllInstallerScripts("v1.2.3", "")
	core.RequireTrue(t, result.OK)
	scripts := result.Value.(map[string]string)
	core.AssertContains(t, scripts, "agent.sh")
}

func TestInstallers_GenerateAll_Good(t *core.T) {
	result := GenerateAll("v1.2.3", "dappcore/core")
	core.RequireTrue(t, result.OK)
	scripts := result.Value.(map[string]string)
	core.AssertContains(t, scripts, "go.sh")
}

func TestInstallers_GenerateAll_Bad(t *core.T) {
	result := GenerateAll("bad version!", "dappcore/core")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "version")
}

func TestInstallers_GenerateAll_Ugly(t *core.T) {
	result := GenerateAll("v1.2.3", "owner/repo.git")
	core.RequireTrue(t, result.OK)
	scripts := result.Value.(map[string]string)
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

package build

import core "dappco.re/go"

func TestLinuxkitTemplates_LinuxKitBaseImages_Good(t *core.T) {
	images := LinuxKitBaseImages()
	core.AssertLen(t, images, 3)
	core.AssertEqual(t, "core-dev", images[0].Name)
}

func TestLinuxkitTemplates_LinuxKitBaseImages_Bad(t *core.T) {
	images := LinuxKitBaseImages()
	images[0].DefaultPackages[0] = "mutated"
	again := LinuxKitBaseImages()
	core.AssertNotEqual(t, "mutated", again[0].DefaultPackages[0])
}

func TestLinuxkitTemplates_LinuxKitBaseImages_Ugly(t *core.T) {
	images := LinuxKitBaseImages()
	core.AssertEqual(t, "core-minimal", images[2].Name)
	core.AssertContains(t, images[2].DefaultPackages, "go")
}

func TestLinuxkitTemplates_LookupLinuxKitBaseImage_Good(t *core.T) {
	image, ok := LookupLinuxKitBaseImage("core-dev")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "core-dev", image.Name)
}

func TestLinuxkitTemplates_LookupLinuxKitBaseImage_Bad(t *core.T) {
	image, ok := LookupLinuxKitBaseImage("missing")
	core.AssertFalse(t, ok)
	core.AssertEqual(t, "", image.Name)
}

func TestLinuxkitTemplates_LookupLinuxKitBaseImage_Ugly(t *core.T) {
	image, ok := LookupLinuxKitBaseImage("core-minimal")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, []string{"go"}, image.DefaultPackages)
}

func TestLinuxkitTemplates_LinuxKitBaseTemplate_Good(t *core.T) {
	result := LinuxKitBaseTemplate("core-dev")
	core.RequireTrue(t, result.OK)
	template := result.Value.(string)
	core.AssertContains(t, template, "CORE_IMAGE=core-dev")
}

func TestLinuxkitTemplates_LinuxKitBaseTemplate_Bad(t *core.T) {
	result := LinuxKitBaseTemplate("missing")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "missing")
}

func TestLinuxkitTemplates_LinuxKitBaseTemplate_Ugly(t *core.T) {
	result := LinuxKitBaseTemplate("core-minimal")
	core.RequireTrue(t, result.OK)
	template := result.Value.(string)
	core.AssertContains(t, template, "core-minimal")
}

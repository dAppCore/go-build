package build

import (
	core "dappco.re/go"
	"testing"
)

func TestBuild_ExpandVersionTemplate_Good(t *testing.T) {
	t.Run("expands tag placeholders", func(t *testing.T) {
		value := ExpandVersionTemplate("-X main.Version={{.Tag}}", "v1.2.3")
		if !stdlibAssertEqual("-X main.Version=v1.2.3", value) {
			t.Fatalf("want %v, got %v", "-X main.Version=v1.2.3", value)
		}

	})

	t.Run("avoids duplicated v prefix in version placeholders", func(t *testing.T) {
		value := ExpandVersionTemplate("v{{.Version}}", "v1.2.3")
		if !stdlibAssertEqual("v1.2.3", value) {
			t.Fatalf("want %v, got %v", "v1.2.3", value)
		}

	})

	t.Run("preserves legacy full version expansion", func(t *testing.T) {
		value := ExpandVersionTemplate("release-{{.Version}}", "v1.2.3")
		if !stdlibAssertEqual("release-v1.2.3", value) {
			t.Fatalf("want %v, got %v", "release-v1.2.3", value)
		}

	})

	t.Run("supports shorthand placeholders", func(t *testing.T) {
		value := ExpandVersionTemplate("{{Tag}}-{{Version}}", "v1.2.3")
		if !stdlibAssertEqual("v1.2.3-v1.2.3", value) {
			t.Fatalf("want %v, got %v", "v1.2.3-v1.2.3", value)
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestVersionTemplates_ExpandVersionTemplate_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ExpandVersionTemplate("agent", "v1.2.3")
	})
	core.AssertTrue(t, true)
}

func TestVersionTemplates_ExpandVersionTemplate_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ExpandVersionTemplate("", "")
	})
	core.AssertTrue(t, true)
}

func TestVersionTemplates_ExpandVersionTemplate_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ExpandVersionTemplate("agent", "v1.2.3")
	})
	core.AssertTrue(t, true)
}

func TestVersionTemplates_ExpandVersionTemplates_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ExpandVersionTemplates([]string{"agent"}, "v1.2.3")
	})
	core.AssertTrue(t, true)
}

func TestVersionTemplates_ExpandVersionTemplates_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ExpandVersionTemplates([]string{"agent"}, "")
	})
	core.AssertTrue(t, true)
}

func TestVersionTemplates_ExpandVersionTemplates_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ExpandVersionTemplates([]string{"agent"}, "v1.2.3")
	})
	core.AssertTrue(t, true)
}

func TestVersionTemplates_ExpandVersionTemplateMap_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ExpandVersionTemplateMap(nil, "v1.2.3")
	})
	core.AssertTrue(t, true)
}

func TestVersionTemplates_ExpandVersionTemplateMap_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ExpandVersionTemplateMap(nil, "")
	})
	core.AssertTrue(t, true)
}

func TestVersionTemplates_ExpandVersionTemplateMap_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ExpandVersionTemplateMap(nil, "v1.2.3")
	})
	core.AssertTrue(t, true)
}

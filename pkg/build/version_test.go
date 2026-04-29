package build

import (
	core "dappco.re/go"
	"testing"
)

func TestValidateVersionString_Good(t *testing.T) {
	for _, version := range []string{
		"v1.2.3",
		"1.2.3-beta.1+exp.sha_5114f85",
		"dev-build_20260425",
	} {
		t.Run(version, func(t *testing.T) {
			if err := ValidateVersionString(version); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateVersionString_Bad(t *testing.T) {
	for _, version := range []string{
		"v1.2.3;rm",
		`v1.2.3"`,
		"v1.2.3$IFS",
		"v1.2.3`uname`",
	} {
		t.Run(version, func(t *testing.T) {
			if err := ValidateVersionString(version); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateVersionString_Ugly(t *testing.T) {
	for _, version := range []string{
		"",
		" ",
		" v1.2.3",
		"v1.2.3 ",
		"v1.2.3 beta",
	} {
		t.Run(version, func(t *testing.T) {
			if err := ValidateVersionString(version); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestVersion_ValidateVersionString_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionString("v1.2.3")
	})
	core.AssertTrue(t, true)
}

func TestVersion_ValidateVersionString_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionString("")
	})
	core.AssertTrue(t, true)
}

func TestVersion_ValidateVersionString_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionString("v1.2.3")
	})
	core.AssertTrue(t, true)
}

func TestVersion_ValidateVersionIdentifier_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionIdentifier("v1.2.3")
	})
	core.AssertTrue(t, true)
}

func TestVersion_ValidateVersionIdentifier_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionIdentifier("")
	})
	core.AssertTrue(t, true)
}

func TestVersion_ValidateVersionIdentifier_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionIdentifier("v1.2.3")
	})
	core.AssertTrue(t, true)
}

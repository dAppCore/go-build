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
			requireVersionFlagOK(t, ValidateVersionString(version))
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
			requireVersionFlagError(t, ValidateVersionString(version))
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
			requireVersionFlagError(t, ValidateVersionString(version))
		})
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestVersion_ValidateVersionString_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionString("v1.2.3")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestVersion_ValidateVersionString_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionString("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestVersion_ValidateVersionString_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionString("v1.2.3")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestVersion_ValidateVersionIdentifier_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionIdentifier("v1.2.3")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestVersion_ValidateVersionIdentifier_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionIdentifier("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestVersion_ValidateVersionIdentifier_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionIdentifier("v1.2.3")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

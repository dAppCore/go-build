package build

import "testing"

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

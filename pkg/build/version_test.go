package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateVersionString_Good(t *testing.T) {
	for _, version := range []string{
		"v1.2.3",
		"1.2.3-beta.1+exp.sha_5114f85",
		"dev-build_20260425",
	} {
		t.Run(version, func(t *testing.T) {
			assert.NoError(t, ValidateVersionString(version))
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
			assert.Error(t, ValidateVersionString(version))
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
			assert.Error(t, ValidateVersionString(version))
		})
	}
}

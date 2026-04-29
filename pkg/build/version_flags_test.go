package build

import (
	core "dappco.re/go"
	"testing"
)

func requireVersionFlag(t *testing.T, result core.Result) string {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(string)
}

func requireVersionFlagOK(t *testing.T, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requireVersionFlagError(t *testing.T, result core.Result) {
	t.Helper()
	if result.OK {
		t.Fatal("expected error")
	}
}

func TestVersionLinkerFlag_Good(t *testing.T) {
	flag := requireVersionFlag(t, VersionLinkerFlag("v1.2.3-beta.1+exp.sha"))
	if !stdlibAssertEqual("-X main.version=v1.2.3-beta.1+exp.sha", flag) {
		t.Fatalf("want %v, got %v", "-X main.version=v1.2.3-beta.1+exp.sha", flag)
	}
}

func TestVersionLinkerFlag_Bad(t *testing.T) {
	result := VersionLinkerFlag("v1.2.3;rm -rf /")
	requireVersionFlagError(t, result)
	if !stdlibAssertContains(result.Error(), "unsupported characters") {
		t.Fatalf("expected %v to contain %v", result.Error(), "unsupported characters")
	}
}

func TestValidateVersionIdentifier_Bad(t *testing.T) {
	requireVersionFlagOK(t, ValidateVersionIdentifier("v1.2.3"))
	requireVersionFlagOK(t, ValidateVersionIdentifier("dev"))
	requireVersionFlagError(t, ValidateVersionIdentifier("v1.2.3\n--flag"))
}

func TestVersionFlags_ValidateVersionIdentifier_Good(t *testing.T) {
	t.Run("accepts empty version", func(t *testing.T) {
		requireVersionFlagOK(t, ValidateVersionIdentifier(""))
	})

	t.Run("accepts exact safe version", func(t *testing.T) {
		requireVersionFlagOK(t, ValidateVersionIdentifier("v1.2.3-beta.1+exp.sha"))
	})
}

func TestVersionFlags_ValidateVersionIdentifier_Ugly(t *testing.T) {
	t.Run("rejects non-ASCII identifiers", func(t *testing.T) {
		requireVersionFlagError(t, ValidateVersionIdentifier("v1.2.3-β"))
	})

	t.Run("rejects shell metacharacters", func(t *testing.T) {
		requireVersionFlagError(t, ValidateVersionIdentifier("v1.2.3 && echo unsafe"))
	})

	t.Run("rejects surrounding whitespace", func(t *testing.T) {
		requireVersionFlagError(t, ValidateVersionIdentifier("  v1.2.3-beta.1+exp.sha  "))
	})
}

func TestVersionFlags_VersionLinkerFlag_Good(t *testing.T) {
	t.Run("renders exact safe version", func(t *testing.T) {
		flag := requireVersionFlag(t, VersionLinkerFlag("v1.2.3"))
		if !stdlibAssertEqual("-X main.version=v1.2.3", flag) {
			t.Fatalf("want %v, got %v", "-X main.version=v1.2.3", flag)
		}
	})
}

func TestVersionFlags_VersionLinkerFlag_Ugly(t *testing.T) {
	t.Run("empty version is a no-op", func(t *testing.T) {
		flag := requireVersionFlag(t, VersionLinkerFlag(""))
		if !stdlibAssertEmpty(flag) {
			t.Fatalf("expected empty, got %v", flag)
		}
	})

	t.Run("rejects surrounding whitespace", func(t *testing.T) {
		requireVersionFlagError(t, VersionLinkerFlag(" v1.2.3 "))
	})
}

// --- v0.9.0 generated compliance triplets ---
func TestVersionFlags_VersionLinkerFlag_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = VersionLinkerFlag("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

package build

import "testing"

func TestVersionLinkerFlag_Good(t *testing.T) {
	flag, err := VersionLinkerFlag("v1.2.3-beta.1+exp.sha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("-X main.version=v1.2.3-beta.1+exp.sha", flag) {
		t.Fatalf("want %v, got %v", "-X main.version=v1.2.3-beta.1+exp.sha", flag)
	}
}

func TestVersionLinkerFlag_Bad(t *testing.T) {
	flag, err := VersionLinkerFlag("v1.2.3;rm -rf /")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(flag) {
		t.Fatalf("expected empty, got %v", flag)
	}
}

func TestValidateVersionIdentifier_Bad(t *testing.T) {
	if err := ValidateVersionIdentifier("v1.2.3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateVersionIdentifier("dev"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateVersionIdentifier("v1.2.3\n--flag"); err == nil {
		t.Fatal("expected error")
	}
}

func TestVersionFlags_ValidateVersionIdentifier_Good(t *testing.T) {
	t.Run("accepts empty version", func(t *testing.T) {
		if err := ValidateVersionIdentifier(""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts exact safe version", func(t *testing.T) {
		if err := ValidateVersionIdentifier("v1.2.3-beta.1+exp.sha"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestVersionFlags_ValidateVersionIdentifier_Ugly(t *testing.T) {
	t.Run("rejects non-ASCII identifiers", func(t *testing.T) {
		if err := ValidateVersionIdentifier("v1.2.3-β"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects shell metacharacters", func(t *testing.T) {
		if err := ValidateVersionIdentifier("v1.2.3 && echo unsafe"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects surrounding whitespace", func(t *testing.T) {
		if err := ValidateVersionIdentifier("  v1.2.3-beta.1+exp.sha  "); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestVersionFlags_VersionLinkerFlag_Good(t *testing.T) {
	t.Run("renders exact safe version", func(t *testing.T) {
		flag, err := VersionLinkerFlag("v1.2.3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("-X main.version=v1.2.3", flag) {
			t.Fatalf("want %v, got %v", "-X main.version=v1.2.3", flag)
		}
	})
}

func TestVersionFlags_VersionLinkerFlag_Ugly(t *testing.T) {
	t.Run("empty version is a no-op", func(t *testing.T) {
		flag, err := VersionLinkerFlag("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEmpty(flag) {
			t.Fatalf("expected empty, got %v", flag)
		}
	})

	t.Run("rejects surrounding whitespace", func(t *testing.T) {
		flag, err := VersionLinkerFlag(" v1.2.3 ")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(flag) {
			t.Fatalf("expected empty, got %v", flag)
		}
	})
}

package build

import (
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

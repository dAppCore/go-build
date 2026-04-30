package sdk

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

func writeFakePHP(t *testing.T, dir string) string {
	t.Helper()

	phpPath := ax.Join(dir, "php")
	script := `#!/bin/sh
set -eu
if [ "$1" != "artisan" ] || [ "$2" != "scramble:export" ]; then
  exit 64
fi
output_path="api.json"
shift 2
while [ "$#" -gt 0 ]; do
  case "$1" in
    --path=*)
      output_path="${1#--path=}"
      ;;
    --path)
      shift
      output_path="$1"
      ;;
  esac
  shift
done
printf '{"openapi":"3.1.0"}\n' > "$output_path"
`
	if result := ax.WriteFile(phpPath, []byte(script), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	return phpPath
}

func requireSDKDetectSpec(t *testing.T, sdk *SDK) string {
	t.Helper()
	result := sdk.DetectSpec()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(string)
}

func TestDetect_DetectSpecConfigPathGood(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "api", "spec.yaml")
	if result := ax.MkdirAll(ax.Dir(specPath), 0755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	if result := ax.WriteFile(specPath, []byte("openapi: 3.0.0"), 0644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	sdk := New(tmpDir, &Config{Spec: "api/spec.yaml"})
	got := requireSDKDetectSpec(t, sdk)
	if !stdlibAssertEqual(specPath, got) {
		t.Fatalf("want %v, got %v", specPath, got)
	}

}

func TestDetect_DetectSpecCommonPathGood(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "openapi.yaml")
	if result := ax.WriteFile(specPath, []byte("openapi: 3.0.0"), 0644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	sdk := New(tmpDir, nil)
	got := requireSDKDetectSpec(t, sdk)
	if !stdlibAssertEqual(specPath, got) {
		t.Fatalf("want %v, got %v", specPath, got)
	}

}

func TestDetect_DetectSpecCommonYAMLPathGood(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "openapi.yml")
	if result := ax.WriteFile(specPath, []byte("openapi: 3.0.0"), 0644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	sdk := New(tmpDir, nil)
	got := requireSDKDetectSpec(t, sdk)
	if !stdlibAssertEqual(specPath, got) {
		t.Fatalf("want %v, got %v", specPath, got)
	}

}

func TestDetect_DetectSpecDocsOpenAPIPathGood(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "docs", "openapi.yaml")
	if result := ax.MkdirAll(ax.Dir(specPath), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(specPath, []byte("openapi: 3.0.0"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	sdk := New(tmpDir, nil)
	got := requireSDKDetectSpec(t, sdk)
	if !stdlibAssertEqual(specPath, got) {
		t.Fatalf("want %v, got %v", specPath, got)
	}

}

func TestDetect_DetectSpecNotFoundBad(t *testing.T) {
	tmpDir := t.TempDir()
	sdk := New(tmpDir, nil)
	result := sdk.DetectSpec()
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "no OpenAPI spec found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "no OpenAPI spec found")
	}

}

func TestDetect_DetectSpecConfigNotFoundBad(t *testing.T) {
	tmpDir := t.TempDir()
	sdk := New(tmpDir, &Config{Spec: "non-existent.yaml"})
	result := sdk.DetectSpec()
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "configured spec not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "configured spec not found")
	}

}

func TestDetect_ContainsScrambleGood(t *testing.T) {
	tests := []struct {
		data     string
		expected bool
	}{
		{`{"require": {"dedoc/scramble": "^0.1"}}`, true},
		{`{"require": {"scramble": "^0.1"}}`, true},
		{`{"require": {"laravel/framework": "^11.0"}}`, false},
	}

	for _, tt := range tests {
		if !stdlibAssertEqual(tt.expected, containsScramble(tt.data)) {
			t.Fatalf("want %v, got %v", tt.expected, containsScramble(tt.data))
		}

	}
}

func TestDetect_DetectScrambleBad(t *testing.T) {
	t.Run("no composer.json", func(t *testing.T) {
		sdk := New(t.TempDir(), nil)
		result := sdk.detectScramble()
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "no composer.json") {
			t.Fatalf("expected %v to contain %v", result.Error(), "no composer.json")
		}

	})

	t.Run("no scramble in composer.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		if result := ax.WriteFile(ax.Join(tmpDir, "composer.json"), []byte(`{}`), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		sdk := New(tmpDir, nil)
		result := sdk.detectScramble()
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "scramble not found") {
			t.Fatalf("expected %v to contain %v", result.Error(), "scramble not found")
		}

	})
}

func TestDetect_DetectSpecScrambleGood(t *testing.T) {
	tmpDir := t.TempDir()
	if result := ax.WriteFile(ax.Join(tmpDir, "composer.json"), []byte(`{"require":{"dedoc/scramble":"^0.1"}}`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	phpDir := t.TempDir()
	writeFakePHP(t, phpDir)
	t.Setenv("PATH", phpDir)

	sdk := New(tmpDir, nil)
	got := requireSDKDetectSpec(t, sdk)
	if !stdlibAssertEqual(ax.Join(tmpDir, "api.json"), got) {
		t.Fatalf("want %v, got %v", ax.Join(tmpDir, "api.json"), got)
	}

	dataResult := ax.ReadFile(got)
	if !dataResult.OK {
		t.Fatalf("unexpected error: %v", dataResult.Error())
	}
	data := dataResult.Value.([]byte)
	if !stdlibAssertContains(string(data), `"openapi":"3.1.0"`) {
		t.Fatalf("expected %v to contain %v", string(data), `"openapi":"3.1.0"`)
	}

}

func TestDetect_DetectSpecScrambleOverwritesExistingSpecGood(t *testing.T) {
	tmpDir := t.TempDir()
	if result := ax.WriteFile(ax.Join(tmpDir, "composer.json"), []byte(`{"require":{"dedoc/scramble":"^0.1"}}`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(tmpDir, "api.json"), []byte(`{"openapi":"3.0.0","info":{"title":"stale"}}`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	phpDir := t.TempDir()
	writeFakePHP(t, phpDir)
	t.Setenv("PATH", phpDir)

	sdk := New(tmpDir, nil)
	got := requireSDKDetectSpec(t, sdk)
	if !stdlibAssertEqual(ax.Join(tmpDir, "api.json"), got) {
		t.Fatalf("want %v, got %v", ax.Join(tmpDir, "api.json"), got)
	}

	dataResult := ax.ReadFile(got)
	if !dataResult.OK {
		t.Fatalf("unexpected error: %v", dataResult.Error())
	}
	data := dataResult.Value.([]byte)
	if stdlibAssertContains(string(data), "stale") {
		t.Fatalf("expected %v not to contain %v", string(data), "stale")
	}
	if !stdlibAssertContains(string(data), `"openapi":"3.1.0"`) {
		t.Fatalf("expected %v to contain %v", string(data), `"openapi":"3.1.0"`)
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestDetect_SDK_DetectSpec_Good(t *core.T) {
	subject := &SDK{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.DetectSpec()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDetect_SDK_DetectSpec_Bad(t *core.T) {
	subject := &SDK{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.DetectSpec()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDetect_SDK_DetectSpec_Ugly(t *core.T) {
	subject := &SDK{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.DetectSpec()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

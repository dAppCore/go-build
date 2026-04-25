package ci

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/release"
)

func TestCI_runCIReleaseInitInDir_Good(t *testing.T) {
	projectDir := t.TempDir()

	err := runCIReleaseInitInDir(projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configPath := release.ConfigPath(projectDir)
	content, err := ax.ReadFile(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(content), "sdk:") {
		t.Fatalf("expected %v to contain %v", string(content), "sdk:")
	}
	if !stdlibAssertContains(string(content), "spec: api/openapi.yaml") {
		t.Fatalf("expected %v to contain %v", string(content), "spec: api/openapi.yaml")
	}
	if !stdlibAssertContains(string(content), "languages:") {
		t.Fatalf("expected %v to contain %v", string(content), "languages:")
	}
	if !stdlibAssertContains(string(content), "- typescript") {
		t.Fatalf("expected %v to contain %v", string(content), "- typescript")
	}

}

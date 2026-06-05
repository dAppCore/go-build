package buildcmd

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
)

// artifactNoun is singular only for exactly one artifact.
func TestBuildCmd_artifactNoun_Good(t *testing.T) {
	for n, want := range map[int]string{0: "artifacts", 1: "artifact", 2: "artifacts", 9: "artifacts"} {
		if got := artifactNoun(n); got != want {
			t.Fatalf("artifactNoun(%d) = %q, want %q", n, got, want)
		}
	}
}

// buildArtifactsDir reports where the artifacts actually landed (bin/),
// relative to the project — not the configured OUTPUT_DIR (dist/).
func TestBuildCmd_buildArtifactsDir_UsesArtifactDir_Good(t *testing.T) {
	projectDir := "/project"
	artifacts := []build.Artifact{{Path: ax.Join(projectDir, "bin", "app")}}

	if got := buildArtifactsDir(artifacts, ax.Join(projectDir, "dist"), projectDir); got != "bin" {
		t.Fatalf("expected the real artifact dir 'bin', got %q", got)
	}
}

// With no artifacts to point at, it falls back to the output dir (relative).
func TestBuildCmd_buildArtifactsDir_FallsBackToOutputDir_Bad(t *testing.T) {
	projectDir := "/project"

	if got := buildArtifactsDir(nil, ax.Join(projectDir, "dist"), projectDir); got != "dist" {
		t.Fatalf("expected fallback 'dist', got %q", got)
	}
}

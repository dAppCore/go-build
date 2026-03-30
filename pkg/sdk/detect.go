package sdk

import (
	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// commonSpecPaths are checked in order when no spec is configured.
var commonSpecPaths = []string{
	"api/openapi.yaml",
	"api/openapi.json",
	"openapi.yaml",
	"openapi.json",
	"docs/api.yaml",
	"docs/api.json",
	"swagger.yaml",
	"swagger.json",
}

// DetectSpec finds the OpenAPI spec file.
// Priority: config path -> common paths -> Laravel Scramble.
// Usage example: call value.DetectSpec(...) from integrating code.
func (s *SDK) DetectSpec() (string, error) {
	// 1. Check configured path
	if s.config.Spec != "" {
		specPath := ax.Join(s.projectDir, s.config.Spec)
		if ax.IsFile(specPath) {
			return specPath, nil
		}
		return "", coreerr.E("sdk.DetectSpec", "configured spec not found: "+s.config.Spec, nil)
	}

	// 2. Check common paths
	for _, p := range commonSpecPaths {
		specPath := ax.Join(s.projectDir, p)
		if ax.IsFile(specPath) {
			return specPath, nil
		}
	}

	// 3. Try Laravel Scramble detection
	specPath, err := s.detectScramble()
	if err == nil {
		return specPath, nil
	}

	return "", coreerr.E("sdk.DetectSpec", "no OpenAPI spec found (checked config, common paths, Scramble)", nil)
}

// detectScramble checks for Laravel Scramble and exports the spec.
func (s *SDK) detectScramble() (string, error) {
	composerPath := ax.Join(s.projectDir, "composer.json")
	if !ax.IsFile(composerPath) {
		return "", coreerr.E("sdk.detectScramble", "no composer.json", nil)
	}

	// Check for scramble in composer.json
	data, err := ax.ReadFile(composerPath)
	if err != nil {
		return "", err
	}

	// Simple check for scramble package
	if !containsScramble(string(data)) {
		return "", coreerr.E("sdk.detectScramble", "scramble not found in composer.json", nil)
	}

	// TODO: Run php artisan scramble:export
	return "", coreerr.E("sdk.detectScramble", "scramble export not implemented", nil)
}

// containsScramble checks if composer.json includes scramble.
func containsScramble(content string) bool {
	return core.Contains(content, "dedoc/scramble") ||
		core.Contains(content, "\"scramble\"")
}

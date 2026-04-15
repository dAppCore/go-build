package sdk

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// commonSpecPaths are checked in order when no spec is configured.
var commonSpecPaths = []string{
	"api/openapi.yaml",
	"api/openapi.yml",
	"api/openapi.json",
	"openapi.yaml",
	"openapi.yml",
	"openapi.json",
	"docs/openapi.yaml",
	"docs/openapi.yml",
	"docs/openapi.json",
	"docs/api.yaml",
	"docs/api.yml",
	"docs/api.json",
	"swagger.yaml",
	"swagger.yml",
	"swagger.json",
}

// DetectSpec finds the OpenAPI spec file.
// Priority: config path -> common paths -> Laravel Scramble.
//
// path, err := s.DetectSpec() // → "api/openapi.yaml", nil
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
	composerPath := ax.Join(s.projectDir, "composer.json")
	if ax.IsFile(composerPath) {
		data, err := ax.ReadFile(composerPath)
		if err != nil {
			return "", err
		}

		if containsScramble(string(data)) {
			specPath, err := s.detectScramble()
			if err != nil {
				return "", err
			}
			return specPath, nil
		}
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

	scrambleSpecPath := ax.Join(s.projectDir, "api.json")

	phpCommand, err := resolvePHPCli()
	if err != nil {
		return "", coreerr.E("sdk.detectScramble", "php CLI not found", err)
	}

	if err := ax.ExecDir(context.Background(), s.projectDir, phpCommand, "artisan", "scramble:export", "--path=api.json"); err != nil {
		return "", coreerr.E("sdk.detectScramble", "scramble export failed", err)
	}

	if !ax.IsFile(scrambleSpecPath) {
		return "", coreerr.E("sdk.detectScramble", "scramble export did not create api.json", nil)
	}

	return scrambleSpecPath, nil
}

// containsScramble checks if composer.json includes scramble.
func containsScramble(content string) bool {
	return core.Contains(content, "dedoc/scramble") ||
		core.Contains(content, "\"scramble\"")
}

func resolvePHPCli() (string, error) {
	return ax.ResolveCommand("php",
		"/usr/bin/php",
		"/usr/local/bin/php",
		"/opt/homebrew/bin/php",
	)
}

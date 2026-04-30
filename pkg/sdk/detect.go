package sdk

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
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
func (s *SDK) DetectSpec() core.Result {
	if s == nil {
		return core.Fail(core.E("sdk.DetectSpec", "sdk is nil", nil))
	}
	if s.config == nil {
		return core.Fail(core.E("sdk.DetectSpec", "sdk config is nil", nil))
	}

	// 1. Check configured path
	if s.config.Spec != "" {
		specPath := ax.Join(s.projectDir, s.config.Spec)
		if ax.IsFile(specPath) {
			return core.Ok(specPath)
		}
		return core.Fail(core.E("sdk.DetectSpec", "configured spec not found: "+s.config.Spec, nil))
	}

	// 2. Check common paths
	for _, p := range commonSpecPaths {
		specPath := ax.Join(s.projectDir, p)
		if ax.IsFile(specPath) {
			return core.Ok(specPath)
		}
	}

	// 3. Try Laravel Scramble detection
	composerPath := ax.Join(s.projectDir, "composer.json")
	if ax.IsFile(composerPath) {
		data := ax.ReadFile(composerPath)
		if !data.OK {
			return data
		}

		if containsScramble(string(data.Value.([]byte))) {
			specPath := s.detectScramble()
			if !specPath.OK {
				return specPath
			}
			return specPath
		}
	}

	return core.Fail(core.E("sdk.DetectSpec", "no OpenAPI spec found (checked config, common paths, Scramble)", nil))
}

// detectScramble checks for Laravel Scramble and exports the spec.
func (s *SDK) detectScramble() core.Result {
	composerPath := ax.Join(s.projectDir, "composer.json")
	if !ax.IsFile(composerPath) {
		return core.Fail(core.E("sdk.detectScramble", "no composer.json", nil))
	}

	// Check for scramble in composer.json
	data := ax.ReadFile(composerPath)
	if !data.OK {
		return data
	}

	// Simple check for scramble package
	if !containsScramble(string(data.Value.([]byte))) {
		return core.Fail(core.E("sdk.detectScramble", "scramble not found in composer.json", nil))
	}

	scrambleSpecPath := ax.Join(s.projectDir, "api.json")

	phpCommand := resolvePHPCli()
	if !phpCommand.OK {
		return core.Fail(core.E("sdk.detectScramble", "php CLI not found", core.NewError(phpCommand.Error())))
	}

	exported := ax.ExecDir(context.Background(), s.projectDir, phpCommand.Value.(string), "artisan", "scramble:export", "--path=api.json")
	if !exported.OK {
		return core.Fail(core.E("sdk.detectScramble", "scramble export failed", core.NewError(exported.Error())))
	}

	if !ax.IsFile(scrambleSpecPath) {
		return core.Fail(core.E("sdk.detectScramble", "scramble export did not create api.json", nil))
	}

	return core.Ok(scrambleSpecPath)
}

// containsScramble checks if composer.json includes scramble.
func containsScramble(content string) bool {
	return core.Contains(content, "dedoc/scramble") ||
		core.Contains(content, "\"scramble\"")
}

func resolvePHPCli() core.Result {
	return ax.ResolveCommand("php",
		"/usr/bin/php",
		"/usr/local/bin/php",
		"/opt/homebrew/bin/php",
	)
}

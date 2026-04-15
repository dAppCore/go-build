package sdk

import (
	"dappco.re/go/core"
	coreerr "dappco.re/go/core/log"
	"github.com/oasdiff/kin-openapi/openapi3"
	"github.com/oasdiff/oasdiff/checker"
	"github.com/oasdiff/oasdiff/diff"
	"github.com/oasdiff/oasdiff/load"
)

// DiffResult holds the result of comparing two OpenAPI specs.
//
// result, err := sdk.Diff("docs/openapi.v1.yaml", "docs/openapi.yaml")
type DiffResult struct {
	// Breaking is true if breaking changes were detected.
	Breaking bool
	// Changes is the list of breaking changes.
	Changes []string
	// HasWarnings is true if warning-level changes were detected.
	HasWarnings bool
	// Warnings is the list of warning-level changes.
	Warnings []string
	// Summary is a human-readable summary.
	Summary string
}

// DiffOptions controls the change levels included in the diff result.
type DiffOptions struct {
	// MinimumLevel selects the lowest severity to include.
	// Defaults to checker.ERR to preserve breaking-only behaviour.
	MinimumLevel checker.Level
}

// Diff compares two OpenAPI specs and detects breaking changes.
//
// result, err := sdk.Diff("docs/openapi.v1.yaml", "docs/openapi.yaml")
func Diff(basePath, revisionPath string) (*DiffResult, error) {
	return DiffWithOptions(basePath, revisionPath, DiffOptions{MinimumLevel: checker.ERR})
}

// DiffWithOptions compares two OpenAPI specs and includes changes at or above
// the requested severity level.
func DiffWithOptions(basePath, revisionPath string, opts DiffOptions) (*DiffResult, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	// Load specs
	baseSpec, err := load.NewSpecInfo(loader, load.NewSource(basePath))
	if err != nil {
		return nil, coreerr.E("sdk.Diff", "failed to load base spec", err)
	}

	revSpec, err := load.NewSpecInfo(loader, load.NewSource(revisionPath))
	if err != nil {
		return nil, coreerr.E("sdk.Diff", "failed to load revision spec", err)
	}

	// Compute diff with operations sources map for better error reporting
	diffResult, operationsSources, err := diff.GetWithOperationsSourcesMap(diff.NewConfig(), baseSpec, revSpec)
	if err != nil {
		return nil, coreerr.E("sdk.Diff", "failed to compute diff", err)
	}

	// Check for breaking changes
	config := checker.NewConfig(checker.GetAllChecks())
	changes := checker.CheckBackwardCompatibilityUntilLevel(
		config,
		diffResult,
		operationsSources,
		resolveDiffLevel(opts.MinimumLevel),
	)

	// Build result
	result := &DiffResult{
		Breaking: len(changes) > 0 && changes.HasLevelOrHigher(checker.ERR),
		Changes:  make([]string, 0, len(changes)),
		Warnings: make([]string, 0, len(changes)),
	}

	localizer := checker.NewDefaultLocalizer()
	for _, change := range changes {
		// GetUncolorizedText uses US spelling — upstream oasdiff API.
		text := change.GetUncolorizedText(localizer)
		switch change.GetLevel() {
		case checker.ERR:
			result.Changes = append(result.Changes, text)
		case checker.WARN:
			result.HasWarnings = true
			result.Warnings = append(result.Warnings, text)
		}
	}

	result.Summary = diffSummary(result, resolveDiffLevel(opts.MinimumLevel))

	return result, nil
}

// DiffExitCode returns the exit code for CI integration.
// 0 = no breaking changes, 1 = breaking changes, 2 = error.
//
// os.Exit(sdk.DiffExitCode(sdk.Diff("old.yaml", "new.yaml")))
func DiffExitCode(result *DiffResult, err error) int {
	if err != nil {
		return 2
	}
	if result.Breaking {
		return 1
	}
	return 0
}

func resolveDiffLevel(level checker.Level) checker.Level {
	switch level {
	case checker.WARN, checker.INFO, checker.ERR:
		return level
	default:
		return checker.ERR
	}
}

func diffSummary(result *DiffResult, level checker.Level) string {
	if result == nil {
		return "No breaking changes"
	}

	if level == checker.ERR {
		if result.Breaking {
			return core.Sprintf("%d breaking change(s) detected", len(result.Changes))
		}
		return "No breaking changes"
	}

	switch {
	case result.Breaking && result.HasWarnings:
		return core.Sprintf("%d breaking change(s), %d warning(s) detected", len(result.Changes), len(result.Warnings))
	case result.Breaking:
		return core.Sprintf("%d breaking change(s) detected", len(result.Changes))
	case result.HasWarnings:
		return core.Sprintf("%d warning(s) detected", len(result.Warnings))
	default:
		return "No warnings or breaking changes"
	}
}

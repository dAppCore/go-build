package builders

import (
	"strings"
	"unicode"

	"dappco.re/go"
	"dappco.re/go/build/pkg/build"
	coreerr "dappco.re/go/log"
)

// resolveDenoBuildCommand returns the Deno build invocation using the action-style
// environment override first, then the persisted build config, then the default task.
func resolveDenoBuildCommand(cfg *build.Config, resolveDeno func(...string) (string, error)) (string, []string, error) {
	override := core.Trim(core.Env("DENO_BUILD"))
	if override == "" && cfg != nil {
		override = core.Trim(cfg.DenoBuild)
	}
	if override != "" {
		args, err := splitCommandLine(override)
		if err != nil {
			return "", nil, coreerr.E("builders.resolveDenoBuildCommand", "invalid DENO_BUILD command", err)
		}
		if len(args) == 0 {
			return "", nil, coreerr.E("builders.resolveDenoBuildCommand", "DENO_BUILD command is empty", nil)
		}
		return args[0], args[1:], nil
	}

	command, err := resolveDeno()
	if err != nil {
		return "", nil, err
	}
	return command, []string{"task", "build"}, nil
}

// resolveNpmBuildCommand returns the npm build invocation using the action-style
// environment override first, then the persisted build config, then the default task.
func resolveNpmBuildCommand(cfg *build.Config, resolveNpm func(...string) (string, error)) (string, []string, error) {
	override := core.Trim(core.Env("NPM_BUILD"))
	if override == "" && cfg != nil {
		override = core.Trim(cfg.NpmBuild)
	}
	if override != "" {
		args, err := splitCommandLine(override)
		if err != nil {
			return "", nil, coreerr.E("builders.resolveNpmBuildCommand", "invalid NPM_BUILD command", err)
		}
		if len(args) == 0 {
			return "", nil, coreerr.E("builders.resolveNpmBuildCommand", "NPM_BUILD command is empty", nil)
		}
		return args[0], args[1:], nil
	}

	command, err := resolveNpm()
	if err != nil {
		return "", nil, err
	}
	return command, []string{"run", "build"}, nil
}

// splitCommandLine tokenises a command string with basic shell-style quoting.
func splitCommandLine(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, nil
	}

	var (
		args    []string
		current strings.Builder
		quote   rune
		escape  bool
	)

	flush := func() {
		if current.Len() == 0 {
			return
		}
		args = append(args, current.String())
		current.Reset()
	}

	for _, r := range command {
		switch {
		case escape:
			current.WriteRune(r)
			escape = false
		case r == '\\' && quote != '\'':
			escape = true
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case r == '"' || r == '\'':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if escape {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, coreerr.E("builders.splitCommandLine", "unterminated quote in command", nil)
	}

	flush()
	return args, nil
}

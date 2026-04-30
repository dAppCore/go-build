package builders

import (
	"unicode"

	"dappco.re/go"
	"dappco.re/go/build/pkg/build"
)

type commandSpec struct {
	command string
	args    []string
}

// resolveDenoBuildCommand returns the Deno build invocation using the action-style
// environment override first, then the persisted build config, then the default task.
func resolveDenoBuildCommand(cfg *build.Config, resolveDeno func(...string) core.Result) core.Result {
	override := core.Trim(core.Env("DENO_BUILD"))
	if override == "" && cfg != nil {
		override = core.Trim(cfg.DenoBuild)
	}
	if override != "" {
		argsResult := splitCommandLine(override)
		if !argsResult.OK {
			return core.Fail(core.E("builders.resolveDenoBuildCommand", "invalid DENO_BUILD command", core.NewError(argsResult.Error())))
		}
		args := argsResult.Value.([]string)
		if len(args) == 0 {
			return core.Fail(core.E("builders.resolveDenoBuildCommand", "DENO_BUILD command is empty", nil))
		}
		return core.Ok(commandSpec{command: args[0], args: args[1:]})
	}

	command := resolveDeno()
	if !command.OK {
		return command
	}
	return core.Ok(commandSpec{command: command.Value.(string), args: []string{"task", "build"}})
}

// resolveNpmBuildCommand returns the npm build invocation using the action-style
// environment override first, then the persisted build config, then the default task.
func resolveNpmBuildCommand(cfg *build.Config, resolveNpm func(...string) core.Result) core.Result {
	override := core.Trim(core.Env("NPM_BUILD"))
	if override == "" && cfg != nil {
		override = core.Trim(cfg.NpmBuild)
	}
	if override != "" {
		argsResult := splitCommandLine(override)
		if !argsResult.OK {
			return core.Fail(core.E("builders.resolveNpmBuildCommand", "invalid NPM_BUILD command", core.NewError(argsResult.Error())))
		}
		args := argsResult.Value.([]string)
		if len(args) == 0 {
			return core.Fail(core.E("builders.resolveNpmBuildCommand", "NPM_BUILD command is empty", nil))
		}
		return core.Ok(commandSpec{command: args[0], args: args[1:]})
	}

	command := resolveNpm()
	if !command.OK {
		return command
	}
	return core.Ok(commandSpec{command: command.Value.(string), args: []string{"run", "build"}})
}

// splitCommandLine tokenises a command string with basic shell-style quoting.
func splitCommandLine(command string) core.Result {
	command = core.Trim(command)
	if command == "" {
		return core.Ok([]string(nil))
	}

	var (
		args   []string
		quote  rune
		escape bool
	)
	current := core.NewBuilder()

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
		return core.Fail(core.E("builders.splitCommandLine", "unterminated quote in command", nil))
	}

	flush()
	return core.Ok(args)
}

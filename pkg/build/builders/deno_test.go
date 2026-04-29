package builders

import (
	core "dappco.re/go"
	"testing"

	"dappco.re/go/build/pkg/build"
)

func requireDenoCommandSpec(t *testing.T, result core.Result) commandSpec {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(commandSpec)
}

func requireDenoArgs(t *testing.T, result core.Result) []string {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.([]string)
}

func TestDeno_ResolveDenoBuildCommandGood(t *testing.T) {
	t.Run("environment override takes precedence over config and default", func(t *testing.T) {
		t.Setenv("DENO_BUILD", `deno task "build docs" --watch`)

		cfg := &build.Config{DenoBuild: "deno task ignored"}

		spec := requireDenoCommandSpec(t, resolveDenoBuildCommand(cfg, func(...string) core.Result {
			t.Fatal("resolver should not be called when DENO_BUILD is set")
			return core.Ok("")
		}))
		if !stdlibAssertEqual("deno", spec.command) {
			t.Fatalf("want %v, got %v", "deno", spec.command)
		}
		if !stdlibAssertEqual([]string{"task", "build docs", "--watch"}, spec.args) {
			t.Fatalf("want %v, got %v", []string{"task", "build docs", "--watch"}, spec.args)
		}

	})

	t.Run("config override is used when environment override is absent", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "")

		cfg := &build.Config{DenoBuild: `deno task "bundle app"`}

		spec := requireDenoCommandSpec(t, resolveDenoBuildCommand(cfg, func(...string) core.Result {
			t.Fatal("resolver should not be called when config override is set")
			return core.Ok("")
		}))
		if !stdlibAssertEqual("deno", spec.command) {
			t.Fatalf("want %v, got %v", "deno", spec.command)
		}
		if !stdlibAssertEqual([]string{"task", "bundle app"}, spec.args) {
			t.Fatalf("want %v, got %v", []string{"task", "bundle app"}, spec.args)
		}

	})

	t.Run("falls back to the resolver default when no override exists", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "")

		spec := requireDenoCommandSpec(t, resolveDenoBuildCommand(&build.Config{}, func(...string) core.Result {
			return core.Ok("deno")
		}))
		if !stdlibAssertEqual("deno", spec.command) {
			t.Fatalf("want %v, got %v", "deno", spec.command)
		}
		if !stdlibAssertEqual([]string{"task", "build"}, spec.args) {
			t.Fatalf("want %v, got %v", []string{"task", "build"}, spec.args)
		}

	})
}

func TestDeno_ResolveDenoBuildCommandBad(t *testing.T) {
	t.Run("invalid shell quoting is rejected", func(t *testing.T) {
		t.Setenv("DENO_BUILD", `deno task "unterminated`)

		result := resolveDenoBuildCommand(&build.Config{}, func(...string) core.Result {
			t.Fatal("resolver should not be called when parsing fails")
			return core.Ok("")
		})
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "invalid DENO_BUILD command") {
			t.Fatalf("expected %v to contain %v", result.Error(), "invalid DENO_BUILD command")
		}

	})

	t.Run("resolver errors are surfaced when no override exists", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "")

		result := resolveDenoBuildCommand(nil, func(...string) core.Result {
			return core.Fail(core.NewError("deno not found"))
		})
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "deno not found") {
			t.Fatalf("expected %v to contain %v", result.Error(), "deno not found")
		}

	})
}

func TestDeno_ResolveDenoBuildCommandUgly(t *testing.T) {
	t.Run("trimmed empty command falls through to the default resolver", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "   ")

		spec := requireDenoCommandSpec(t, resolveDenoBuildCommand(&build.Config{}, func(...string) core.Result {
			return core.Ok("deno")
		}))
		if !stdlibAssertEqual("deno", spec.command) {
			t.Fatalf("want %v, got %v", "deno", spec.command)
		}
		if !stdlibAssertEqual([]string{"task", "build"}, spec.args) {
			t.Fatalf("want %v, got %v", []string{"task", "build"}, spec.args)
		}

	})
}

func TestDeno_ResolveNpmBuildCommandGood(t *testing.T) {
	t.Run("environment override takes precedence over config and default", func(t *testing.T) {
		t.Setenv("NPM_BUILD", `npm run "build docs" -- --watch`)

		cfg := &build.Config{NpmBuild: "npm run ignored"}

		spec := requireDenoCommandSpec(t, resolveNpmBuildCommand(cfg, func(...string) core.Result {
			t.Fatal("resolver should not be called when NPM_BUILD is set")
			return core.Ok("")
		}))
		if !stdlibAssertEqual("npm", spec.command) {
			t.Fatalf("want %v, got %v", "npm", spec.command)
		}
		if !stdlibAssertEqual([]string{"run", "build docs", "--", "--watch"}, spec.args) {
			t.Fatalf("want %v, got %v", []string{"run", "build docs", "--", "--watch"}, spec.args)
		}

	})

	t.Run("config override is used when environment override is absent", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "")

		cfg := &build.Config{NpmBuild: `npm run "bundle app"`}

		spec := requireDenoCommandSpec(t, resolveNpmBuildCommand(cfg, func(...string) core.Result {
			t.Fatal("resolver should not be called when config override is set")
			return core.Ok("")
		}))
		if !stdlibAssertEqual("npm", spec.command) {
			t.Fatalf("want %v, got %v", "npm", spec.command)
		}
		if !stdlibAssertEqual([]string{"run", "bundle app"}, spec.args) {
			t.Fatalf("want %v, got %v", []string{"run", "bundle app"}, spec.args)
		}

	})

	t.Run("falls back to the resolver default when no override exists", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "")

		spec := requireDenoCommandSpec(t, resolveNpmBuildCommand(&build.Config{}, func(...string) core.Result {
			return core.Ok("npm")
		}))
		if !stdlibAssertEqual("npm", spec.command) {
			t.Fatalf("want %v, got %v", "npm", spec.command)
		}
		if !stdlibAssertEqual([]string{"run", "build"}, spec.args) {
			t.Fatalf("want %v, got %v", []string{"run", "build"}, spec.args)
		}

	})
}

func TestDeno_ResolveNpmBuildCommandBad(t *testing.T) {
	t.Run("invalid shell quoting is rejected", func(t *testing.T) {
		t.Setenv("NPM_BUILD", `npm run "unterminated`)

		result := resolveNpmBuildCommand(&build.Config{}, func(...string) core.Result {
			t.Fatal("resolver should not be called when parsing fails")
			return core.Ok("")
		})
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "invalid NPM_BUILD command") {
			t.Fatalf("expected %v to contain %v", result.Error(), "invalid NPM_BUILD command")
		}

	})

	t.Run("resolver errors are surfaced when no override exists", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "")

		result := resolveNpmBuildCommand(nil, func(...string) core.Result {
			return core.Fail(core.NewError("npm not found"))
		})
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "npm not found") {
			t.Fatalf("expected %v to contain %v", result.Error(), "npm not found")
		}

	})
}

func TestDeno_ResolveNpmBuildCommandUgly(t *testing.T) {
	t.Run("trimmed empty command falls through to the default resolver", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "   ")

		spec := requireDenoCommandSpec(t, resolveNpmBuildCommand(&build.Config{}, func(...string) core.Result {
			return core.Ok("npm")
		}))
		if !stdlibAssertEqual("npm", spec.command) {
			t.Fatalf("want %v, got %v", "npm", spec.command)
		}
		if !stdlibAssertEqual([]string{"run", "build"}, spec.args) {
			t.Fatalf("want %v, got %v", []string{"run", "build"}, spec.args)
		}

	})
}

func TestDeno_SplitCommandLineGood(t *testing.T) {
	t.Run("handles quoted arguments and escaped spaces", func(t *testing.T) {
		args := requireDenoArgs(t, splitCommandLine(`deno task "build docs" --flag value\ with\ spaces`))
		if !stdlibAssertEqual([]string{"deno", "task", "build docs", "--flag", "value with spaces"}, args) {
			t.Fatalf("want %v, got %v", []string{"deno", "task", "build docs", "--flag", "value with spaces"}, args)
		}

	})
}

func TestDeno_SplitCommandLineBad(t *testing.T) {
	t.Run("rejects unterminated quotes", func(t *testing.T) {
		result := splitCommandLine(`deno task "build docs`)
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "unterminated quote") {
			t.Fatalf("expected %v to contain %v", result.Error(), "unterminated quote")
		}

	})
}

func TestDeno_SplitCommandLineUgly(t *testing.T) {
	t.Run("empty input returns no args", func(t *testing.T) {
		args := requireDenoArgs(t, splitCommandLine("   "))
		if !stdlibAssertNil(args) {
			t.Fatalf("expected nil, got %v", args)
		}

	})
}

package builders

import (
	"errors"
	"testing"

	"dappco.re/go/build/pkg/build"
)

func TestDeno_ResolveDenoBuildCommand_Good(t *testing.T) {
	t.Run("environment override takes precedence over config and default", func(t *testing.T) {
		t.Setenv("DENO_BUILD", `deno task "build docs" --watch`)

		cfg := &build.Config{DenoBuild: "deno task ignored"}

		command, args, err := resolveDenoBuildCommand(cfg, func(...string) (string, error) {
			t.Fatal("resolver should not be called when DENO_BUILD is set")
			return "", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("deno", command) {
			t.Fatalf("want %v, got %v", "deno", command)
		}
		if !stdlibAssertEqual([]string{"task", "build docs", "--watch"}, args) {
			t.Fatalf("want %v, got %v", []string{"task", "build docs", "--watch"}, args)
		}

	})

	t.Run("config override is used when environment override is absent", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "")

		cfg := &build.Config{DenoBuild: `deno task "bundle app"`}

		command, args, err := resolveDenoBuildCommand(cfg, func(...string) (string, error) {
			t.Fatal("resolver should not be called when config override is set")
			return "", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("deno", command) {
			t.Fatalf("want %v, got %v", "deno", command)
		}
		if !stdlibAssertEqual([]string{"task", "bundle app"}, args) {
			t.Fatalf("want %v, got %v", []string{"task", "bundle app"}, args)
		}

	})

	t.Run("falls back to the resolver default when no override exists", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "")

		command, args, err := resolveDenoBuildCommand(&build.Config{}, func(...string) (string, error) {
			return "deno", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("deno", command) {
			t.Fatalf("want %v, got %v", "deno", command)
		}
		if !stdlibAssertEqual([]string{"task", "build"}, args) {
			t.Fatalf("want %v, got %v", []string{"task", "build"}, args)
		}

	})
}

func TestDeno_ResolveDenoBuildCommand_Bad(t *testing.T) {
	t.Run("invalid shell quoting is rejected", func(t *testing.T) {
		t.Setenv("DENO_BUILD", `deno task "unterminated`)

		_, _, err := resolveDenoBuildCommand(&build.Config{}, func(...string) (string, error) {
			t.Fatal("resolver should not be called when parsing fails")
			return "", nil
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "invalid DENO_BUILD command") {
			t.Fatalf("expected %v to contain %v", err.Error(), "invalid DENO_BUILD command")
		}

	})

	t.Run("resolver errors are surfaced when no override exists", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "")

		_, _, err := resolveDenoBuildCommand(nil, func(...string) (string, error) {
			return "", errors.New("deno not found")
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "deno not found") {
			t.Fatalf("expected %v to contain %v", err.Error(), "deno not found")
		}

	})
}

func TestDeno_ResolveDenoBuildCommand_Ugly(t *testing.T) {
	t.Run("trimmed empty command falls through to the default resolver", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "   ")

		command, args, err := resolveDenoBuildCommand(&build.Config{}, func(...string) (string, error) {
			return "deno", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("deno", command) {
			t.Fatalf("want %v, got %v", "deno", command)
		}
		if !stdlibAssertEqual([]string{"task", "build"}, args) {
			t.Fatalf("want %v, got %v", []string{"task", "build"}, args)
		}

	})
}

func TestDeno_ResolveNpmBuildCommand_Good(t *testing.T) {
	t.Run("environment override takes precedence over config and default", func(t *testing.T) {
		t.Setenv("NPM_BUILD", `npm run "build docs" -- --watch`)

		cfg := &build.Config{NpmBuild: "npm run ignored"}

		command, args, err := resolveNpmBuildCommand(cfg, func(...string) (string, error) {
			t.Fatal("resolver should not be called when NPM_BUILD is set")
			return "", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("npm", command) {
			t.Fatalf("want %v, got %v", "npm", command)
		}
		if !stdlibAssertEqual([]string{"run", "build docs", "--", "--watch"}, args) {
			t.Fatalf("want %v, got %v", []string{"run", "build docs", "--", "--watch"}, args)
		}

	})

	t.Run("config override is used when environment override is absent", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "")

		cfg := &build.Config{NpmBuild: `npm run "bundle app"`}

		command, args, err := resolveNpmBuildCommand(cfg, func(...string) (string, error) {
			t.Fatal("resolver should not be called when config override is set")
			return "", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("npm", command) {
			t.Fatalf("want %v, got %v", "npm", command)
		}
		if !stdlibAssertEqual([]string{"run", "bundle app"}, args) {
			t.Fatalf("want %v, got %v", []string{"run", "bundle app"}, args)
		}

	})

	t.Run("falls back to the resolver default when no override exists", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "")

		command, args, err := resolveNpmBuildCommand(&build.Config{}, func(...string) (string, error) {
			return "npm", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("npm", command) {
			t.Fatalf("want %v, got %v", "npm", command)
		}
		if !stdlibAssertEqual([]string{"run", "build"}, args) {
			t.Fatalf("want %v, got %v", []string{"run", "build"}, args)
		}

	})
}

func TestDeno_ResolveNpmBuildCommand_Bad(t *testing.T) {
	t.Run("invalid shell quoting is rejected", func(t *testing.T) {
		t.Setenv("NPM_BUILD", `npm run "unterminated`)

		_, _, err := resolveNpmBuildCommand(&build.Config{}, func(...string) (string, error) {
			t.Fatal("resolver should not be called when parsing fails")
			return "", nil
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "invalid NPM_BUILD command") {
			t.Fatalf("expected %v to contain %v", err.Error(), "invalid NPM_BUILD command")
		}

	})

	t.Run("resolver errors are surfaced when no override exists", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "")

		_, _, err := resolveNpmBuildCommand(nil, func(...string) (string, error) {
			return "", errors.New("npm not found")
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "npm not found") {
			t.Fatalf("expected %v to contain %v", err.Error(), "npm not found")
		}

	})
}

func TestDeno_ResolveNpmBuildCommand_Ugly(t *testing.T) {
	t.Run("trimmed empty command falls through to the default resolver", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "   ")

		command, args, err := resolveNpmBuildCommand(&build.Config{}, func(...string) (string, error) {
			return "npm", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("npm", command) {
			t.Fatalf("want %v, got %v", "npm", command)
		}
		if !stdlibAssertEqual([]string{"run", "build"}, args) {
			t.Fatalf("want %v, got %v", []string{"run", "build"}, args)
		}

	})
}

func TestDeno_SplitCommandLine_Good(t *testing.T) {
	t.Run("handles quoted arguments and escaped spaces", func(t *testing.T) {
		args, err := splitCommandLine(`deno task "build docs" --flag value\ with\ spaces`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]string{"deno", "task", "build docs", "--flag", "value with spaces"}, args) {
			t.Fatalf("want %v, got %v", []string{"deno", "task", "build docs", "--flag", "value with spaces"}, args)
		}

	})
}

func TestDeno_SplitCommandLine_Bad(t *testing.T) {
	t.Run("rejects unterminated quotes", func(t *testing.T) {
		args, err := splitCommandLine(`deno task "build docs`)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertNil(args) {
			t.Fatalf("expected nil, got %v", args)
		}
		if !stdlibAssertContains(err.Error(), "unterminated quote") {
			t.Fatalf("expected %v to contain %v", err.Error(), "unterminated quote")
		}

	})
}

func TestDeno_SplitCommandLine_Ugly(t *testing.T) {
	t.Run("empty input returns no args", func(t *testing.T) {
		args, err := splitCommandLine("   ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertNil(args) {
			t.Fatalf("expected nil, got %v", args)
		}

	})
}

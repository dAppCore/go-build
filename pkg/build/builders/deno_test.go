package builders

import (
	"errors"
	"testing"

	"dappco.re/go/build/pkg/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeno_ResolveDenoBuildCommand_Good(t *testing.T) {
	t.Run("environment override takes precedence over config and default", func(t *testing.T) {
		t.Setenv("DENO_BUILD", `deno task "build docs" --watch`)

		cfg := &build.Config{DenoBuild: "deno task ignored"}

		command, args, err := resolveDenoBuildCommand(cfg, func(...string) (string, error) {
			t.Fatal("resolver should not be called when DENO_BUILD is set")
			return "", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "deno", command)
		assert.Equal(t, []string{"task", "build docs", "--watch"}, args)
	})

	t.Run("config override is used when environment override is absent", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "")

		cfg := &build.Config{DenoBuild: `deno task "bundle app"`}

		command, args, err := resolveDenoBuildCommand(cfg, func(...string) (string, error) {
			t.Fatal("resolver should not be called when config override is set")
			return "", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "deno", command)
		assert.Equal(t, []string{"task", "bundle app"}, args)
	})

	t.Run("falls back to the resolver default when no override exists", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "")

		command, args, err := resolveDenoBuildCommand(&build.Config{}, func(...string) (string, error) {
			return "deno", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "deno", command)
		assert.Equal(t, []string{"task", "build"}, args)
	})
}

func TestDeno_ResolveDenoBuildCommand_Bad(t *testing.T) {
	t.Run("invalid shell quoting is rejected", func(t *testing.T) {
		t.Setenv("DENO_BUILD", `deno task "unterminated`)

		_, _, err := resolveDenoBuildCommand(&build.Config{}, func(...string) (string, error) {
			t.Fatal("resolver should not be called when parsing fails")
			return "", nil
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid DENO_BUILD command")
	})

	t.Run("resolver errors are surfaced when no override exists", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "")

		_, _, err := resolveDenoBuildCommand(nil, func(...string) (string, error) {
			return "", errors.New("deno not found")
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "deno not found")
	})
}

func TestDeno_ResolveDenoBuildCommand_Ugly(t *testing.T) {
	t.Run("trimmed empty command falls through to the default resolver", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "   ")

		command, args, err := resolveDenoBuildCommand(&build.Config{}, func(...string) (string, error) {
			return "deno", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "deno", command)
		assert.Equal(t, []string{"task", "build"}, args)
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

		require.NoError(t, err)
		assert.Equal(t, "npm", command)
		assert.Equal(t, []string{"run", "build docs", "--", "--watch"}, args)
	})

	t.Run("config override is used when environment override is absent", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "")

		cfg := &build.Config{NpmBuild: `npm run "bundle app"`}

		command, args, err := resolveNpmBuildCommand(cfg, func(...string) (string, error) {
			t.Fatal("resolver should not be called when config override is set")
			return "", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "npm", command)
		assert.Equal(t, []string{"run", "bundle app"}, args)
	})

	t.Run("falls back to the resolver default when no override exists", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "")

		command, args, err := resolveNpmBuildCommand(&build.Config{}, func(...string) (string, error) {
			return "npm", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "npm", command)
		assert.Equal(t, []string{"run", "build"}, args)
	})
}

func TestDeno_ResolveNpmBuildCommand_Bad(t *testing.T) {
	t.Run("invalid shell quoting is rejected", func(t *testing.T) {
		t.Setenv("NPM_BUILD", `npm run "unterminated`)

		_, _, err := resolveNpmBuildCommand(&build.Config{}, func(...string) (string, error) {
			t.Fatal("resolver should not be called when parsing fails")
			return "", nil
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid NPM_BUILD command")
	})

	t.Run("resolver errors are surfaced when no override exists", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "")

		_, _, err := resolveNpmBuildCommand(nil, func(...string) (string, error) {
			return "", errors.New("npm not found")
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "npm not found")
	})
}

func TestDeno_ResolveNpmBuildCommand_Ugly(t *testing.T) {
	t.Run("trimmed empty command falls through to the default resolver", func(t *testing.T) {
		t.Setenv("NPM_BUILD", "   ")

		command, args, err := resolveNpmBuildCommand(&build.Config{}, func(...string) (string, error) {
			return "npm", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "npm", command)
		assert.Equal(t, []string{"run", "build"}, args)
	})
}

func TestDeno_SplitCommandLine_Good(t *testing.T) {
	t.Run("handles quoted arguments and escaped spaces", func(t *testing.T) {
		args, err := splitCommandLine(`deno task "build docs" --flag value\ with\ spaces`)
		require.NoError(t, err)
		assert.Equal(t, []string{"deno", "task", "build docs", "--flag", "value with spaces"}, args)
	})
}

func TestDeno_SplitCommandLine_Bad(t *testing.T) {
	t.Run("rejects unterminated quotes", func(t *testing.T) {
		args, err := splitCommandLine(`deno task "build docs`)
		require.Error(t, err)
		assert.Nil(t, args)
		assert.Contains(t, err.Error(), "unterminated quote")
	})
}

func TestDeno_SplitCommandLine_Ugly(t *testing.T) {
	t.Run("empty input returns no args", func(t *testing.T) {
		args, err := splitCommandLine("   ")
		require.NoError(t, err)
		assert.Nil(t, args)
	})
}

package generators

import (
	"context"

	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// TypeScriptGenerator generates TypeScript SDKs from OpenAPI specs.
//
// g := generators.NewTypeScriptGenerator()
type TypeScriptGenerator struct{}

// NewTypeScriptGenerator creates a new TypeScript generator.
//
// g := generators.NewTypeScriptGenerator()
func NewTypeScriptGenerator() *TypeScriptGenerator {
	return &TypeScriptGenerator{}
}

// Language returns the generator's target language identifier.
//
// lang := g.Language() // → "typescript"
func (g *TypeScriptGenerator) Language() string {
	return "typescript"
}

// Available checks if generator dependencies are installed.
//
// if g.Available() { err = g.Generate(ctx, opts) }
func (g *TypeScriptGenerator) Available() bool {
	return g.nativeAvailable() || g.npxAvailable() || dockerRuntimeAvailable()
}

// Install returns instructions for installing the generator.
//
// fmt.Println(g.Install()) // → "npm install -g openapi-typescript-codegen"
func (g *TypeScriptGenerator) Install() string {
	return "npm install -g openapi-typescript-codegen"
}

// Generate creates SDK from OpenAPI spec.
//
// err := g.Generate(ctx, generators.Options{SpecPath: "docs/openapi.yaml", OutputDir: "sdk/typescript"})
func (g *TypeScriptGenerator) Generate(ctx context.Context, opts Options) error {
	if err := ctx.Err(); err != nil {
		return coreerr.E("typescript.Generate", "generation cancelled", err)
	}

	if err := ax.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return coreerr.E("typescript.Generate", "failed to create output dir", err)
	}

	if command, err := g.resolveNativeCli(); err == nil {
		return g.generateNative(ctx, opts, command)
	}
	if command, err := g.resolveNpxCli(); err == nil {
		if g.npxAvailableWithContext(ctx, command) {
			return g.generateNpx(ctx, opts, command)
		}
		if err := ctx.Err(); err != nil {
			return coreerr.E("typescript.Generate", "generation cancelled", err)
		}
	}
	if !dockerRuntimeAvailableWithContext(ctx) {
		if err := ctx.Err(); err != nil {
			return coreerr.E("typescript.Generate", "generation cancelled", err)
		}
		return coreerr.E("typescript.Generate", "Docker is required for fallback generation but not available", nil)
	}
	return g.generateDocker(ctx, opts)
}

func (g *TypeScriptGenerator) nativeAvailable() bool {
	_, err := g.resolveNativeCli()
	return err == nil
}

func (g *TypeScriptGenerator) npxAvailable() bool {
	command, err := g.resolveNpxCli()
	if err != nil {
		return false
	}

	ctx, cancel := availabilityProbeContext()
	defer cancel()

	return g.npxAvailableWithContext(ctx, command)
}

func (g *TypeScriptGenerator) npxAvailableWithContext(ctx context.Context, command string) bool {
	_, err := ax.Run(ctx, command, "--version")
	return err == nil
}

func (g *TypeScriptGenerator) resolveNativeCli(paths ...string) (string, error) {
	command, err := ax.ResolveCommand("openapi-typescript-codegen", paths...)
	if err != nil {
		return "", coreerr.E("typescript.resolveNativeCli", "openapi-typescript-codegen not found. Install it with: "+g.Install(), err)
	}
	return command, nil
}

func (g *TypeScriptGenerator) resolveNpxCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/npx",
			"/opt/homebrew/bin/npx",
		}
	}

	command, err := ax.ResolveCommand("npx", paths...)
	if err != nil {
		return "", coreerr.E("typescript.resolveNpxCli", "npx not found. Install Node.js from https://nodejs.org/", err)
	}
	return command, nil
}

func (g *TypeScriptGenerator) generateNative(ctx context.Context, opts Options, command string) error {
	return ax.Exec(ctx, command,
		"--input", opts.SpecPath,
		"--output", opts.OutputDir,
		"--name", opts.PackageName,
	)
}

func (g *TypeScriptGenerator) generateNpx(ctx context.Context, opts Options, command string) error {
	return ax.Exec(ctx, command, "--yes", "openapi-typescript-codegen",
		"--input", opts.SpecPath,
		"--output", opts.OutputDir,
		"--name", opts.PackageName,
	)
}

func (g *TypeScriptGenerator) generateDocker(ctx context.Context, opts Options) error {
	dockerCommand, err := resolveDockerRuntimeCli()
	if err != nil {
		return coreerr.E("typescript.generateDocker", "docker CLI not available", err)
	}

	specDir := ax.Dir(opts.SpecPath)
	specName := ax.Base(opts.SpecPath)

	args := []string{"run", "--rm"}
	args = append(args, dockerUserArgs()...)
	args = append(args,
		"-v", specDir+":/spec",
		"-v", opts.OutputDir+":/out",
		"openapitools/openapi-generator-cli", "generate",
		"-i", "/spec/"+specName,
		"-g", "typescript-fetch",
		"-o", "/out",
		"--additional-properties=npmName="+opts.PackageName,
	)

	if err := ax.Exec(ctx, dockerCommand, args...); err != nil {
		return coreerr.E("typescript.generateDocker", "docker run failed", err)
	}
	return nil
}

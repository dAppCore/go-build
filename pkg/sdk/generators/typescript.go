package generators

import (
	"context"

	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// TypeScriptGenerator generates TypeScript SDKs from OpenAPI specs.
// Usage example: declare a value of type generators.TypeScriptGenerator in integrating code.
type TypeScriptGenerator struct{}

// NewTypeScriptGenerator creates a new TypeScript generator.
// Usage example: call generators.NewTypeScriptGenerator(...) from integrating code.
func NewTypeScriptGenerator() *TypeScriptGenerator {
	return &TypeScriptGenerator{}
}

// Language returns the generator's target language identifier.
// Usage example: call value.Language(...) from integrating code.
func (g *TypeScriptGenerator) Language() string {
	return "typescript"
}

// Available checks if generator dependencies are installed.
// Usage example: call value.Available(...) from integrating code.
func (g *TypeScriptGenerator) Available() bool {
	return g.nativeAvailable() || g.npxAvailable()
}

// Install returns instructions for installing the generator.
// Usage example: call value.Install(...) from integrating code.
func (g *TypeScriptGenerator) Install() string {
	return "npm install -g openapi-typescript-codegen"
}

// Generate creates SDK from OpenAPI spec.
// Usage example: call value.Generate(...) from integrating code.
func (g *TypeScriptGenerator) Generate(ctx context.Context, opts Options) error {
	if err := ax.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return coreerr.E("typescript.Generate", "failed to create output dir", err)
	}

	if g.nativeAvailable() {
		return g.generateNative(ctx, opts)
	}
	if g.npxAvailable() {
		return g.generateNpx(ctx, opts)
	}
	if !dockerRuntimeAvailable() {
		return coreerr.E("typescript.Generate", "Docker is required for fallback generation but not available", nil)
	}
	return g.generateDocker(ctx, opts)
}

func (g *TypeScriptGenerator) nativeAvailable() bool {
	_, err := ax.LookPath("openapi-typescript-codegen")
	return err == nil
}

func (g *TypeScriptGenerator) npxAvailable() bool {
	_, err := ax.Run(context.Background(), "npx", "--version")
	return err == nil
}

func (g *TypeScriptGenerator) generateNative(ctx context.Context, opts Options) error {
	return ax.Exec(ctx, "openapi-typescript-codegen",
		"--input", opts.SpecPath,
		"--output", opts.OutputDir,
		"--name", opts.PackageName,
	)
}

func (g *TypeScriptGenerator) generateNpx(ctx context.Context, opts Options) error {
	return ax.Exec(ctx, "npx", "--yes", "openapi-typescript-codegen",
		"--input", opts.SpecPath,
		"--output", opts.OutputDir,
		"--name", opts.PackageName,
	)
}

func (g *TypeScriptGenerator) generateDocker(ctx context.Context, opts Options) error {
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

	if err := ax.Exec(ctx, "docker", args...); err != nil {
		return coreerr.E("typescript.generateDocker", "docker run failed", err)
	}
	return nil
}

package generators

import (
	"context"

	"dappco.re/go/build/internal/ax"
	coreerr "dappco.re/go/log"
)

// PHPGenerator generates PHP SDKs from OpenAPI specs.
//
// g := generators.NewPHPGenerator()
type PHPGenerator struct{}

// NewPHPGenerator creates a new PHP generator.
//
// g := generators.NewPHPGenerator()
func NewPHPGenerator() *PHPGenerator {
	return &PHPGenerator{}
}

// Language returns the generator's target language identifier.
//
// lang := g.Language() // → "php"
func (g *PHPGenerator) Language() string {
	return "php"
}

// Available checks if generator dependencies are installed (requires Docker).
//
// if g.Available() { err = g.Generate(ctx, opts) }
func (g *PHPGenerator) Available() bool {
	return dockerRuntimeAvailable()
}

// Install returns instructions for installing the generator.
//
// fmt.Println(g.Install()) // → "Docker is required for PHP SDK generation"
func (g *PHPGenerator) Install() string {
	return "Docker is required for PHP SDK generation"
}

// Generate creates SDK from OpenAPI spec (requires Docker).
//
// err := g.Generate(ctx, generators.Options{SpecPath: "docs/openapi.yaml", OutputDir: "sdk/php"})
func (g *PHPGenerator) Generate(ctx context.Context, opts Options) error {
	if err := ctx.Err(); err != nil {
		return coreerr.E("php.Generate", "generation cancelled", err)
	}

	if !dockerRuntimeAvailableWithContext(ctx) {
		if err := ctx.Err(); err != nil {
			return coreerr.E("php.Generate", "generation cancelled", err)
		}
		return coreerr.E("php.Generate", "Docker is required but not available", nil)
	}

	dockerCommand, err := resolveDockerRuntimeCli()
	if err != nil {
		return coreerr.E("php.Generate", "docker CLI not available", err)
	}

	if err := ax.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return coreerr.E("php.Generate", "failed to create output dir", err)
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
		"-g", "php",
		"-o", "/out",
		"--additional-properties=invokerPackage="+opts.PackageName,
	)

	if err := ax.Exec(ctx, dockerCommand, args...); err != nil {
		return coreerr.E("php.Generate", "docker run failed", err)
	}
	return nil
}

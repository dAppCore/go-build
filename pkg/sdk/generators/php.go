package generators

import (
	"context"

	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// PHPGenerator generates PHP SDKs from OpenAPI specs.
// Usage example: declare a value of type generators.PHPGenerator in integrating code.
type PHPGenerator struct{}

// NewPHPGenerator creates a new PHP generator.
// Usage example: call generators.NewPHPGenerator(...) from integrating code.
func NewPHPGenerator() *PHPGenerator {
	return &PHPGenerator{}
}

// Language returns the generator's target language identifier.
// Usage example: call value.Language(...) from integrating code.
func (g *PHPGenerator) Language() string {
	return "php"
}

// Available checks if generator dependencies are installed.
// Usage example: call value.Available(...) from integrating code.
func (g *PHPGenerator) Available() bool {
	return dockerRuntimeAvailable()
}

// Install returns instructions for installing the generator.
// Usage example: call value.Install(...) from integrating code.
func (g *PHPGenerator) Install() string {
	return "Docker is required for PHP SDK generation"
}

// Generate creates SDK from OpenAPI spec.
// Usage example: call value.Generate(...) from integrating code.
func (g *PHPGenerator) Generate(ctx context.Context, opts Options) error {
	if !g.Available() {
		return coreerr.E("php.Generate", "Docker is required but not available", nil)
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

	if err := ax.Exec(ctx, "docker", args...); err != nil {
		return coreerr.E("php.Generate", "docker run failed", err)
	}
	return nil
}

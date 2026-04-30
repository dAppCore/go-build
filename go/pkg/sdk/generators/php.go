package generators

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
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
func (g *PHPGenerator) Generate(ctx context.Context, opts Options) core.Result {
	if err := ctx.Err(); err != nil {
		return core.Fail(core.E("php.Generate", "generation cancelled", err))
	}

	if !dockerRuntimeAvailableWithContext(ctx) {
		if err := ctx.Err(); err != nil {
			return core.Fail(core.E("php.Generate", "generation cancelled", err))
		}
		return core.Fail(core.E("php.Generate", "Docker is required but not available", nil))
	}

	dockerCommand := resolveDockerRuntimeCli()
	if !dockerCommand.OK {
		return core.Fail(core.E("php.Generate", "docker CLI not available", core.NewError(dockerCommand.Error())))
	}

	created := ax.MkdirAll(opts.OutputDir, 0o755)
	if !created.OK {
		return core.Fail(core.E("php.Generate", "failed to create output dir", core.NewError(created.Error())))
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

	run := ax.Exec(ctx, dockerCommand.Value.(string), args...)
	if !run.OK {
		return core.Fail(core.E("php.Generate", "docker run failed", core.NewError(run.Error())))
	}
	return core.Ok(nil)
}

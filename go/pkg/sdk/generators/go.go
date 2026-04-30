package generators

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

// GoGenerator generates Go SDKs from OpenAPI specs.
//
// g := generators.NewGoGenerator()
type GoGenerator struct{}

// NewGoGenerator creates a new Go generator.
//
// g := generators.NewGoGenerator()
func NewGoGenerator() *GoGenerator {
	return &GoGenerator{}
}

// Language returns the generator's target language identifier.
//
// lang := g.Language() // → "go"
func (g *GoGenerator) Language() string {
	return "go"
}

// Available checks if generator dependencies are installed.
//
// if g.Available() { err = g.Generate(ctx, opts) }
func (g *GoGenerator) Available() bool {
	return g.resolveNativeCli().OK || dockerRuntimeAvailable()
}

// Install returns instructions for installing the generator.
//
// fmt.Println(g.Install()) // → "go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest"
func (g *GoGenerator) Install() string {
	return "go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest"
}

// Generate creates SDK from OpenAPI spec.
//
// err := g.Generate(ctx, generators.Options{SpecPath: "docs/openapi.yaml", OutputDir: "sdk/go"})
func (g *GoGenerator) Generate(ctx context.Context, opts Options) core.Result {
	if err := ctx.Err(); err != nil {
		return core.Fail(core.E("go.Generate", "generation cancelled", err))
	}

	created := ax.MkdirAll(opts.OutputDir, 0o755)
	if !created.OK {
		return core.Fail(core.E("go.Generate", "failed to create output dir", core.NewError(created.Error())))
	}

	if command := g.resolveNativeCli(); command.OK {
		return g.generateNative(ctx, opts, command.Value.(string))
	}
	if !dockerRuntimeAvailableWithContext(ctx) {
		if err := ctx.Err(); err != nil {
			return core.Fail(core.E("go.Generate", "generation cancelled", err))
		}
		return core.Fail(core.E("go.Generate", "Docker is required for fallback generation but not available", nil))
	}
	return g.generateDocker(ctx, opts)
}

func (g *GoGenerator) resolveNativeCli(paths ...string) core.Result {
	command := ax.ResolveCommand("oapi-codegen", paths...)
	if !command.OK {
		return core.Fail(core.E("go.resolveNativeCli", "oapi-codegen not found. Install it with: "+g.Install(), core.NewError(command.Error())))
	}
	return command
}

func (g *GoGenerator) generateNative(ctx context.Context, opts Options, command string) core.Result {
	outputFile := ax.Join(opts.OutputDir, "client.go")

	generated := ax.Exec(ctx, command,
		"-package", opts.PackageName,
		"-generate", "types,client",
		"-o", outputFile,
		opts.SpecPath,
	)
	if !generated.OK {
		return core.Fail(core.E("go.generateNative", "oapi-codegen failed", core.NewError(generated.Error())))
	}

	goMod := core.Sprintf("module %s\n\ngo 1.21\n", opts.PackageName)
	return ax.WriteString(ax.Join(opts.OutputDir, "go.mod"), goMod, 0o644)
}

func (g *GoGenerator) generateDocker(ctx context.Context, opts Options) core.Result {
	dockerCommand := resolveDockerRuntimeCli()
	if !dockerCommand.OK {
		return core.Fail(core.E("go.generateDocker", "docker CLI not available", core.NewError(dockerCommand.Error())))
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
		"-g", "go",
		"-o", "/out",
		"--additional-properties=packageName="+opts.PackageName,
	)

	return ax.Exec(ctx, dockerCommand.Value.(string), args...)
}

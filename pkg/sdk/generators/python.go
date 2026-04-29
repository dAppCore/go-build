package generators

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

// PythonGenerator generates Python SDKs from OpenAPI specs.
//
// g := generators.NewPythonGenerator()
type PythonGenerator struct{}

// NewPythonGenerator creates a new Python generator.
//
// g := generators.NewPythonGenerator()
func NewPythonGenerator() *PythonGenerator {
	return &PythonGenerator{}
}

// Language returns the generator's target language identifier.
//
// lang := g.Language() // → "python"
func (g *PythonGenerator) Language() string {
	return "python"
}

// Available checks if generator dependencies are installed.
//
// if g.Available() { err = g.Generate(ctx, opts) }
func (g *PythonGenerator) Available() bool {
	return g.resolveNativeCli().OK || dockerRuntimeAvailable()
}

// Install returns instructions for installing the generator.
//
// fmt.Println(g.Install()) // → "pip install openapi-python-client"
func (g *PythonGenerator) Install() string {
	return "pip install openapi-python-client"
}

// Generate creates SDK from OpenAPI spec.
//
// err := g.Generate(ctx, generators.Options{SpecPath: "docs/openapi.yaml", OutputDir: "sdk/python"})
func (g *PythonGenerator) Generate(ctx context.Context, opts Options) core.Result {
	if err := ctx.Err(); err != nil {
		return core.Fail(core.E("python.Generate", "generation cancelled", err))
	}

	created := ax.MkdirAll(opts.OutputDir, 0o755)
	if !created.OK {
		return core.Fail(core.E("python.Generate", "failed to create output dir", core.NewError(created.Error())))
	}

	if command := g.resolveNativeCli(); command.OK {
		return g.generateNative(ctx, opts, command.Value.(string))
	}
	if !dockerRuntimeAvailableWithContext(ctx) {
		if err := ctx.Err(); err != nil {
			return core.Fail(core.E("python.Generate", "generation cancelled", err))
		}
		return core.Fail(core.E("python.Generate", "Docker is required for fallback generation but not available", nil))
	}
	return g.generateDocker(ctx, opts)
}

func (g *PythonGenerator) resolveNativeCli(paths ...string) core.Result {
	command := ax.ResolveCommand("openapi-python-client", paths...)
	if !command.OK {
		return core.Fail(core.E("python.resolveNativeCli", "openapi-python-client not found. Install it with: "+g.Install(), core.NewError(command.Error())))
	}
	return command
}

func (g *PythonGenerator) generateNative(ctx context.Context, opts Options, command string) core.Result {
	parentDir := ax.Dir(opts.OutputDir)

	return ax.ExecDir(ctx, parentDir, command, "generate",
		"--path", opts.SpecPath,
		"--output-path", opts.OutputDir,
	)
}

func (g *PythonGenerator) generateDocker(ctx context.Context, opts Options) core.Result {
	dockerCommand := resolveDockerRuntimeCli()
	if !dockerCommand.OK {
		return core.Fail(core.E("python.generateDocker", "docker CLI not available", core.NewError(dockerCommand.Error())))
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
		"-g", "python",
		"-o", "/out",
		"--additional-properties=packageName="+opts.PackageName,
	)

	return ax.Exec(ctx, dockerCommand.Value.(string), args...)
}

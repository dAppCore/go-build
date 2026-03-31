package generators

import (
	"context"

	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
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
	_, err := g.resolveNativeCli()
	return err == nil || dockerRuntimeAvailable()
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
func (g *PythonGenerator) Generate(ctx context.Context, opts Options) error {
	if err := ctx.Err(); err != nil {
		return coreerr.E("python.Generate", "generation cancelled", err)
	}

	if err := ax.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return coreerr.E("python.Generate", "failed to create output dir", err)
	}

	if command, err := g.resolveNativeCli(); err == nil {
		return g.generateNative(ctx, opts, command)
	}
	if !dockerRuntimeAvailableWithContext(ctx) {
		if err := ctx.Err(); err != nil {
			return coreerr.E("python.Generate", "generation cancelled", err)
		}
		return coreerr.E("python.Generate", "Docker is required for fallback generation but not available", nil)
	}
	return g.generateDocker(ctx, opts)
}

func (g *PythonGenerator) resolveNativeCli(paths ...string) (string, error) {
	command, err := ax.ResolveCommand("openapi-python-client", paths...)
	if err != nil {
		return "", coreerr.E("python.resolveNativeCli", "openapi-python-client not found. Install it with: "+g.Install(), err)
	}
	return command, nil
}

func (g *PythonGenerator) generateNative(ctx context.Context, opts Options, command string) error {
	parentDir := ax.Dir(opts.OutputDir)

	return ax.ExecDir(ctx, parentDir, command, "generate",
		"--path", opts.SpecPath,
		"--output-path", opts.OutputDir,
	)
}

func (g *PythonGenerator) generateDocker(ctx context.Context, opts Options) error {
	dockerCommand, err := resolveDockerRuntimeCli()
	if err != nil {
		return coreerr.E("python.generateDocker", "docker CLI not available", err)
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

	return ax.Exec(ctx, dockerCommand, args...)
}

package generators

import (
	"context"

	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// PythonGenerator generates Python SDKs from OpenAPI specs.
// Usage example: declare a value of type generators.PythonGenerator in integrating code.
type PythonGenerator struct{}

// NewPythonGenerator creates a new Python generator.
// Usage example: call generators.NewPythonGenerator(...) from integrating code.
func NewPythonGenerator() *PythonGenerator {
	return &PythonGenerator{}
}

// Language returns the generator's target language identifier.
// Usage example: call value.Language(...) from integrating code.
func (g *PythonGenerator) Language() string {
	return "python"
}

// Available checks if generator dependencies are installed.
// Usage example: call value.Available(...) from integrating code.
func (g *PythonGenerator) Available() bool {
	_, err := ax.LookPath("openapi-python-client")
	return err == nil
}

// Install returns instructions for installing the generator.
// Usage example: call value.Install(...) from integrating code.
func (g *PythonGenerator) Install() string {
	return "pip install openapi-python-client"
}

// Generate creates SDK from OpenAPI spec.
// Usage example: call value.Generate(...) from integrating code.
func (g *PythonGenerator) Generate(ctx context.Context, opts Options) error {
	if err := ax.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return coreerr.E("python.Generate", "failed to create output dir", err)
	}

	if g.Available() {
		return g.generateNative(ctx, opts)
	}
	if !dockerRuntimeAvailable() {
		return coreerr.E("python.Generate", "Docker is required for fallback generation but not available", nil)
	}
	return g.generateDocker(ctx, opts)
}

func (g *PythonGenerator) generateNative(ctx context.Context, opts Options) error {
	parentDir := ax.Dir(opts.OutputDir)

	return ax.ExecDir(ctx, parentDir, "openapi-python-client", "generate",
		"--path", opts.SpecPath,
		"--output-path", opts.OutputDir,
	)
}

func (g *PythonGenerator) generateDocker(ctx context.Context, opts Options) error {
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

	return ax.Exec(ctx, "docker", args...)
}

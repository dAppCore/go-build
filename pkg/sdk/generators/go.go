package generators

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// GoGenerator generates Go SDKs from OpenAPI specs.
// Usage example: declare a value of type generators.GoGenerator in integrating code.
type GoGenerator struct{}

// NewGoGenerator creates a new Go generator.
// Usage example: call generators.NewGoGenerator(...) from integrating code.
func NewGoGenerator() *GoGenerator {
	return &GoGenerator{}
}

// Language returns the generator's target language identifier.
// Usage example: call value.Language(...) from integrating code.
func (g *GoGenerator) Language() string {
	return "go"
}

// Available checks if generator dependencies are installed.
// Usage example: call value.Available(...) from integrating code.
func (g *GoGenerator) Available() bool {
	_, err := ax.LookPath("oapi-codegen")
	return err == nil
}

// Install returns instructions for installing the generator.
// Usage example: call value.Install(...) from integrating code.
func (g *GoGenerator) Install() string {
	return "go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest"
}

// Generate creates SDK from OpenAPI spec.
// Usage example: call value.Generate(...) from integrating code.
func (g *GoGenerator) Generate(ctx context.Context, opts Options) error {
	if err := ax.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return coreerr.E("go.Generate", "failed to create output dir", err)
	}

	if g.Available() {
		return g.generateNative(ctx, opts)
	}
	if !dockerRuntimeAvailable() {
		return coreerr.E("go.Generate", "Docker is required for fallback generation but not available", nil)
	}
	return g.generateDocker(ctx, opts)
}

func (g *GoGenerator) generateNative(ctx context.Context, opts Options) error {
	outputFile := ax.Join(opts.OutputDir, "client.go")

	if err := ax.Exec(ctx, "oapi-codegen",
		"-package", opts.PackageName,
		"-generate", "types,client",
		"-o", outputFile,
		opts.SpecPath,
	); err != nil {
		return coreerr.E("go.generateNative", "oapi-codegen failed", err)
	}

	goMod := core.Sprintf("module %s\n\ngo 1.21\n", opts.PackageName)
	return ax.WriteString(ax.Join(opts.OutputDir, "go.mod"), goMod, 0o644)
}

func (g *GoGenerator) generateDocker(ctx context.Context, opts Options) error {
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

	return ax.Exec(ctx, "docker", args...)
}

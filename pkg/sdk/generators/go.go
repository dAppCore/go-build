package generators

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/build/internal/ax"
	coreerr "dappco.re/go/core/log"
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
	_, err := g.resolveNativeCli()
	return err == nil || dockerRuntimeAvailable()
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
func (g *GoGenerator) Generate(ctx context.Context, opts Options) error {
	if err := ctx.Err(); err != nil {
		return coreerr.E("go.Generate", "generation cancelled", err)
	}

	if err := ax.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return coreerr.E("go.Generate", "failed to create output dir", err)
	}

	if command, err := g.resolveNativeCli(); err == nil {
		return g.generateNative(ctx, opts, command)
	}
	if !dockerRuntimeAvailableWithContext(ctx) {
		if err := ctx.Err(); err != nil {
			return coreerr.E("go.Generate", "generation cancelled", err)
		}
		return coreerr.E("go.Generate", "Docker is required for fallback generation but not available", nil)
	}
	return g.generateDocker(ctx, opts)
}

func (g *GoGenerator) resolveNativeCli(paths ...string) (string, error) {
	command, err := ax.ResolveCommand("oapi-codegen", paths...)
	if err != nil {
		return "", coreerr.E("go.resolveNativeCli", "oapi-codegen not found. Install it with: "+g.Install(), err)
	}
	return command, nil
}

func (g *GoGenerator) generateNative(ctx context.Context, opts Options, command string) error {
	outputFile := ax.Join(opts.OutputDir, "client.go")

	if err := ax.Exec(ctx, command,
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
	dockerCommand, err := resolveDockerRuntimeCli()
	if err != nil {
		return coreerr.E("go.generateDocker", "docker CLI not available", err)
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

	return ax.Exec(ctx, dockerCommand, args...)
}

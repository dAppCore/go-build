package generators

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// PHPGenerator generates PHP SDKs from OpenAPI specs.
type PHPGenerator struct{}

// NewPHPGenerator creates a new PHP generator.
func NewPHPGenerator() *PHPGenerator {
	return &PHPGenerator{}
}

// Language returns the generator's target language identifier.
func (g *PHPGenerator) Language() string {
	return "php"
}

// Available checks if generator dependencies are installed.
func (g *PHPGenerator) Available() bool {
	return dockerRuntimeAvailable()
}

// Install returns instructions for installing the generator.
func (g *PHPGenerator) Install() string {
	return "Docker is required for PHP SDK generation"
}

// Generate creates SDK from OpenAPI spec.
func (g *PHPGenerator) Generate(ctx context.Context, opts Options) error {
	if !g.Available() {
		return coreerr.E("php.Generate", "Docker is required but not available", nil)
	}

	if err := coreio.Local.EnsureDir(opts.OutputDir); err != nil {
		return coreerr.E("php.Generate", "failed to create output dir", err)
	}

	specDir := filepath.Dir(opts.SpecPath)
	specName := filepath.Base(opts.SpecPath)

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

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return coreerr.E("php.Generate", "docker run failed", err)
	}
	return nil
}

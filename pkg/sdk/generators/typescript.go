package generators

import (
	"context"
	"encoding/json"
	stdio "io"
	"strings"

	"dappco.re/go/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// TypeScriptGenerator generates TypeScript SDKs from OpenAPI specs.
//
// g := generators.NewTypeScriptGenerator()
type TypeScriptGenerator struct{}

// NewTypeScriptGenerator creates a new TypeScript generator.
//
// g := generators.NewTypeScriptGenerator()
func NewTypeScriptGenerator() *TypeScriptGenerator {
	return &TypeScriptGenerator{}
}

// Language returns the generator's target language identifier.
//
// lang := g.Language() // → "typescript"
func (g *TypeScriptGenerator) Language() string {
	return "typescript"
}

// Available checks if generator dependencies are installed.
//
// if g.Available() { err = g.Generate(ctx, opts) }
func (g *TypeScriptGenerator) Available() bool {
	return g.nativeAvailable() || g.npxAvailable() || dockerRuntimeAvailable()
}

// Install returns instructions for installing the generator.
//
// fmt.Println(g.Install()) // → "npm install -g openapi-typescript-codegen"
func (g *TypeScriptGenerator) Install() string {
	return "npm install -g openapi-typescript-codegen"
}

// Generate creates SDK from OpenAPI spec.
//
// err := g.Generate(ctx, generators.Options{SpecPath: "docs/openapi.yaml", OutputDir: "sdk/typescript"})
func (g *TypeScriptGenerator) Generate(ctx context.Context, opts Options) error {
	if err := ctx.Err(); err != nil {
		return coreerr.E("typescript.Generate", "generation cancelled", err)
	}

	stagingDir, err := ax.TempDir("core-typescript-sdk-*")
	if err != nil {
		return coreerr.E("typescript.Generate", "failed to create staging dir", err)
	}
	defer func() { _ = ax.RemoveAll(stagingDir) }()

	if command, err := g.resolveNativeCli(); err == nil {
		if err := g.generateNative(ctx, opts, command, stagingDir); err != nil {
			return err
		}
		return finalizeTypeScriptOutput(stagingDir, opts)
	}
	if command, err := g.resolveNpxCli(); err == nil {
		if g.npxAvailableWithContext(ctx, command) {
			if err := g.generateNpx(ctx, opts, command, stagingDir); err != nil {
				return err
			}
			return finalizeTypeScriptOutput(stagingDir, opts)
		}
		if err := ctx.Err(); err != nil {
			return coreerr.E("typescript.Generate", "generation cancelled", err)
		}
	}
	if !dockerRuntimeAvailableWithContext(ctx) {
		if err := ctx.Err(); err != nil {
			return coreerr.E("typescript.Generate", "generation cancelled", err)
		}
		return coreerr.E("typescript.Generate", "Docker is required for fallback generation but not available", nil)
	}
	if err := g.generateDocker(ctx, opts, stagingDir); err != nil {
		return err
	}
	return finalizeTypeScriptOutput(stagingDir, opts)
}

func (g *TypeScriptGenerator) nativeAvailable() bool {
	_, err := g.resolveNativeCli()
	return err == nil
}

func (g *TypeScriptGenerator) npxAvailable() bool {
	command, err := g.resolveNpxCli()
	if err != nil {
		return false
	}

	ctx, cancel := availabilityProbeContext()
	defer cancel()

	return g.npxAvailableWithContext(ctx, command)
}

func (g *TypeScriptGenerator) npxAvailableWithContext(ctx context.Context, command string) bool {
	_, err := ax.Run(ctx, command, "--version")
	return err == nil
}

func (g *TypeScriptGenerator) resolveNativeCli(paths ...string) (string, error) {
	command, err := ax.ResolveCommand("openapi-typescript-codegen", paths...)
	if err != nil {
		return "", coreerr.E("typescript.resolveNativeCli", "openapi-typescript-codegen not found. Install it with: "+g.Install(), err)
	}
	return command, nil
}

func (g *TypeScriptGenerator) resolveNpxCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/npx",
			"/opt/homebrew/bin/npx",
		}
	}

	command, err := ax.ResolveCommand("npx", paths...)
	if err != nil {
		return "", coreerr.E("typescript.resolveNpxCli", "npx not found. Install Node.js from https://nodejs.org/", err)
	}
	return command, nil
}

func (g *TypeScriptGenerator) generateNative(ctx context.Context, opts Options, command string, outputDir string) error {
	return ax.Exec(ctx, command,
		"--input", opts.SpecPath,
		"--output", outputDir,
		"--name", opts.PackageName,
	)
}

func (g *TypeScriptGenerator) generateNpx(ctx context.Context, opts Options, command string, outputDir string) error {
	return ax.Exec(ctx, command, "--yes", "openapi-typescript-codegen",
		"--input", opts.SpecPath,
		"--output", outputDir,
		"--name", opts.PackageName,
	)
}

func (g *TypeScriptGenerator) generateDocker(ctx context.Context, opts Options, outputDir string) error {
	dockerCommand, err := resolveDockerRuntimeCli()
	if err != nil {
		return coreerr.E("typescript.generateDocker", "docker CLI not available", err)
	}

	specDir := ax.Dir(opts.SpecPath)
	specName := ax.Base(opts.SpecPath)

	args := []string{"run", "--rm"}
	args = append(args, dockerUserArgs()...)
	args = append(args,
		"-v", specDir+":/spec",
		"-v", outputDir+":/out",
		"openapitools/openapi-generator-cli", "generate",
		"-i", "/spec/"+specName,
		"-g", "typescript-fetch",
		"-o", "/out",
		"--additional-properties=npmName="+opts.PackageName,
	)

	if err := ax.Exec(ctx, dockerCommand, args...); err != nil {
		return coreerr.E("typescript.generateDocker", "docker run failed", err)
	}
	return nil
}

func finalizeTypeScriptOutput(stagingDir string, opts Options) error {
	if strings.TrimSpace(opts.OutputDir) == "" {
		return coreerr.E("typescript.finalizeOutput", "output dir is required", nil)
	}

	if err := ax.RemoveAll(opts.OutputDir); err != nil && ax.Exists(opts.OutputDir) {
		return coreerr.E("typescript.finalizeOutput", "failed to reset output dir", err)
	}
	if err := ax.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return coreerr.E("typescript.finalizeOutput", "failed to create output dir", err)
	}

	srcDir := ax.Join(opts.OutputDir, "src")
	if err := ax.MkdirAll(srcDir, 0o755); err != nil {
		return coreerr.E("typescript.finalizeOutput", "failed to create src dir", err)
	}

	entries, err := ax.ReadDir(stagingDir)
	if err != nil {
		return coreerr.E("typescript.finalizeOutput", "failed to read staging dir", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		sourcePath := ax.Join(stagingDir, name)

		switch {
		case entry.IsDir() && strings.EqualFold(name, "src"):
			if err := copyTypeScriptDirectoryContents(sourcePath, srcDir); err != nil {
				return err
			}
		case shouldPlaceTypeScriptInSrc(name, entry.IsDir()):
			if err := copyTypeScriptPath(sourcePath, ax.Join(srcDir, name)); err != nil {
				return err
			}
		default:
			if err := copyTypeScriptPath(sourcePath, ax.Join(opts.OutputDir, name)); err != nil {
				return err
			}
		}
	}

	if err := ensureTypeScriptPackageMetadata(opts.OutputDir, opts.PackageName, opts.Version); err != nil {
		return err
	}

	return nil
}

func shouldPlaceTypeScriptInSrc(name string, isDir bool) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return false
	}

	if isDir {
		switch name {
		case "apis", "core", "models", "schemas", "services":
			return true
		default:
			return false
		}
	}

	switch strings.ToLower(ax.Ext(name)) {
	case ".cts", ".mts", ".ts", ".tsx":
		return true
	default:
		return false
	}
}

func ensureTypeScriptPackageMetadata(outputDir, packageName, version string) error {
	manifestPath := ax.Join(outputDir, "package.json")
	manifest := map[string]any{}

	if ax.IsFile(manifestPath) {
		content, err := ax.ReadFile(manifestPath)
		if err != nil {
			return coreerr.E("typescript.ensurePackageMetadata", "failed to read package.json", err)
		}
		if len(strings.TrimSpace(string(content))) > 0 {
			if err := json.Unmarshal(content, &manifest); err != nil {
				return coreerr.E("typescript.ensurePackageMetadata", "failed to parse package.json", err)
			}
		}
	}

	resolvedName := strings.TrimSpace(packageName)
	if resolvedName == "" {
		resolvedName = ax.Base(outputDir)
	}
	manifest["name"] = resolvedName

	if strings.TrimSpace(version) != "" {
		manifest["version"] = strings.TrimSpace(version)
	} else if _, ok := manifest["version"]; !ok {
		manifest["version"] = "0.0.0"
	}

	if _, ok := manifest["type"]; !ok {
		manifest["type"] = "module"
	}
	if _, ok := manifest["files"]; !ok {
		manifest["files"] = []string{"src"}
	}

	indexPath := ax.Join(outputDir, "src", "index.ts")
	if ax.IsFile(indexPath) {
		if _, ok := manifest["types"]; !ok {
			manifest["types"] = "./src/index.ts"
		}
		if _, ok := manifest["exports"]; !ok {
			manifest["exports"] = map[string]any{
				".": "./src/index.ts",
			}
		}
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return coreerr.E("typescript.ensurePackageMetadata", "failed to encode package.json", err)
	}
	if err := ax.WriteFile(manifestPath, append(encoded, '\n'), 0o644); err != nil {
		return coreerr.E("typescript.ensurePackageMetadata", "failed to write package.json", err)
	}

	return nil
}

func copyTypeScriptDirectoryContents(sourceDir, destinationDir string) error {
	entries, err := ax.ReadDir(sourceDir)
	if err != nil {
		return coreerr.E("typescript.copyDirectoryContents", "failed to list source dir", err)
	}

	for _, entry := range entries {
		sourcePath := ax.Join(sourceDir, entry.Name())
		destinationPath := ax.Join(destinationDir, entry.Name())
		if err := copyTypeScriptPath(sourcePath, destinationPath); err != nil {
			return err
		}
	}

	return nil
}

func copyTypeScriptPath(sourcePath, destinationPath string) error {
	info, err := ax.Stat(sourcePath)
	if err != nil {
		return coreerr.E("typescript.copyPath", "failed to stat source path", err)
	}

	if info.IsDir() {
		if err := ax.MkdirAll(destinationPath, 0o755); err != nil {
			return coreerr.E("typescript.copyPath", "failed to create destination dir", err)
		}
		return copyTypeScriptDirectoryContents(sourcePath, destinationPath)
	}

	if err := ax.MkdirAll(ax.Dir(destinationPath), 0o755); err != nil {
		return coreerr.E("typescript.copyPath", "failed to create file parent dir", err)
	}

	sourceFile, err := ax.Open(sourcePath)
	if err != nil {
		return coreerr.E("typescript.copyPath", "failed to open source file", err)
	}
	defer func() { _ = sourceFile.Close() }()

	destinationFile, err := ax.Create(destinationPath)
	if err != nil {
		return coreerr.E("typescript.copyPath", "failed to create destination file", err)
	}
	defer func() { _ = destinationFile.Close() }()

	if _, err := stdio.Copy(destinationFile, sourceFile); err != nil {
		return coreerr.E("typescript.copyPath", "failed to copy file", err)
	}
	if err := ax.Chmod(destinationPath, info.Mode()); err != nil {
		return coreerr.E("typescript.copyPath", "failed to preserve file mode", err)
	}

	return nil
}

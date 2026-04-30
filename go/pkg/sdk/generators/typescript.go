package generators

import (
	"context"
	stdio "io"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
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
func (g *TypeScriptGenerator) Generate(ctx context.Context, opts Options) core.Result {
	if err := ctx.Err(); err != nil {
		return core.Fail(core.E("typescript.Generate", "generation cancelled", err))
	}

	staging := ax.TempDir("core-typescript-sdk-*")
	if !staging.OK {
		return core.Fail(core.E("typescript.Generate", "failed to create staging dir", core.NewError(staging.Error())))
	}
	stagingDir := staging.Value.(string)
	defer func() { _ = ax.RemoveAll(stagingDir) }()

	if command := g.resolveNativeCli(); command.OK {
		generated := g.generateNative(ctx, opts, command.Value.(string), stagingDir)
		if !generated.OK {
			return generated
		}
		return finalizeTypeScriptOutput(stagingDir, opts)
	}
	if command := g.resolveNpxCli(); command.OK {
		if g.npxAvailableWithContext(ctx, command.Value.(string)) {
			generated := g.generateNpx(ctx, opts, command.Value.(string), stagingDir)
			if !generated.OK {
				return generated
			}
			return finalizeTypeScriptOutput(stagingDir, opts)
		}
		if err := ctx.Err(); err != nil {
			return core.Fail(core.E("typescript.Generate", "generation cancelled", err))
		}
	}
	if !dockerRuntimeAvailableWithContext(ctx) {
		if err := ctx.Err(); err != nil {
			return core.Fail(core.E("typescript.Generate", "generation cancelled", err))
		}
		return core.Fail(core.E("typescript.Generate", "Docker is required for fallback generation but not available", nil))
	}
	generated := g.generateDocker(ctx, opts, stagingDir)
	if !generated.OK {
		return generated
	}
	return finalizeTypeScriptOutput(stagingDir, opts)
}

func (g *TypeScriptGenerator) nativeAvailable() bool {
	return g.resolveNativeCli().OK
}

func (g *TypeScriptGenerator) npxAvailable() bool {
	command := g.resolveNpxCli()
	if !command.OK {
		return false
	}

	ctx, cancel := availabilityProbeContext()
	defer cancel()

	return g.npxAvailableWithContext(ctx, command.Value.(string))
}

func (g *TypeScriptGenerator) npxAvailableWithContext(ctx context.Context, command string) bool {
	return ax.Run(ctx, command, "--version").OK
}

func (g *TypeScriptGenerator) resolveNativeCli(paths ...string) core.Result {
	command := ax.ResolveCommand("openapi-typescript-codegen", paths...)
	if !command.OK {
		return core.Fail(core.E("typescript.resolveNativeCli", "openapi-typescript-codegen not found. Install it with: "+g.Install(), core.NewError(command.Error())))
	}
	return command
}

func (g *TypeScriptGenerator) resolveNpxCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/npx",
			"/opt/homebrew/bin/npx",
		}
	}

	command := ax.ResolveCommand("npx", paths...)
	if !command.OK {
		return core.Fail(core.E("typescript.resolveNpxCli", "npx not found. Install Node.js from https://nodejs.org/", core.NewError(command.Error())))
	}
	return command
}

func (g *TypeScriptGenerator) generateNative(ctx context.Context, opts Options, command string, outputDir string) core.Result {
	return ax.Exec(ctx, command,
		"--input", opts.SpecPath,
		"--output", outputDir,
		"--name", opts.PackageName,
	)
}

func (g *TypeScriptGenerator) generateNpx(ctx context.Context, opts Options, command string, outputDir string) core.Result {
	return ax.Exec(ctx, command, "--yes", "openapi-typescript-codegen",
		"--input", opts.SpecPath,
		"--output", outputDir,
		"--name", opts.PackageName,
	)
}

func (g *TypeScriptGenerator) generateDocker(ctx context.Context, opts Options, outputDir string) core.Result {
	dockerCommand := resolveDockerRuntimeCli()
	if !dockerCommand.OK {
		return core.Fail(core.E("typescript.generateDocker", "docker CLI not available", core.NewError(dockerCommand.Error())))
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

	run := ax.Exec(ctx, dockerCommand.Value.(string), args...)
	if !run.OK {
		return core.Fail(core.E("typescript.generateDocker", "docker run failed", core.NewError(run.Error())))
	}
	return core.Ok(nil)
}

func finalizeTypeScriptOutput(stagingDir string, opts Options) core.Result {
	if core.Trim(opts.OutputDir) == "" {
		return core.Fail(core.E("typescript.finalizeOutput", "output dir is required", nil))
	}

	removed := ax.RemoveAll(opts.OutputDir)
	if !removed.OK && ax.Exists(opts.OutputDir) {
		return core.Fail(core.E("typescript.finalizeOutput", "failed to reset output dir", core.NewError(removed.Error())))
	}
	created := ax.MkdirAll(opts.OutputDir, 0o755)
	if !created.OK {
		return core.Fail(core.E("typescript.finalizeOutput", "failed to create output dir", core.NewError(created.Error())))
	}

	srcDir := ax.Join(opts.OutputDir, "src")
	created = ax.MkdirAll(srcDir, 0o755)
	if !created.OK {
		return core.Fail(core.E("typescript.finalizeOutput", "failed to create src dir", core.NewError(created.Error())))
	}

	read := ax.ReadDir(stagingDir)
	if !read.OK {
		return core.Fail(core.E("typescript.finalizeOutput", "failed to read staging dir", core.NewError(read.Error())))
	}

	for _, entry := range read.Value.([]core.FsDirEntry) {
		name := entry.Name()
		sourcePath := ax.Join(stagingDir, name)

		switch {
		case entry.IsDir() && core.Lower(name) == "src":
			copied := copyTypeScriptDirectoryContents(sourcePath, srcDir)
			if !copied.OK {
				return copied
			}
		case shouldPlaceTypeScriptInSrc(name, entry.IsDir()):
			copied := copyTypeScriptPath(sourcePath, ax.Join(srcDir, name))
			if !copied.OK {
				return copied
			}
		default:
			copied := copyTypeScriptPath(sourcePath, ax.Join(opts.OutputDir, name))
			if !copied.OK {
				return copied
			}
		}
	}

	metadata := ensureTypeScriptPackageMetadata(opts.OutputDir, opts.PackageName, opts.Version)
	if !metadata.OK {
		return metadata
	}

	return core.Ok(nil)
}

func shouldPlaceTypeScriptInSrc(name string, isDir bool) bool {
	name = core.Trim(core.Lower(name))
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

	switch core.Lower(ax.Ext(name)) {
	case ".cts", ".mts", ".ts", ".tsx":
		return true
	default:
		return false
	}
}

func ensureTypeScriptPackageMetadata(outputDir, packageName, version string) core.Result {
	manifestPath := ax.Join(outputDir, "package.json")
	manifest := map[string]any{}

	if ax.IsFile(manifestPath) {
		content := ax.ReadFile(manifestPath)
		if !content.OK {
			return core.Fail(core.E("typescript.ensurePackageMetadata", "failed to read package.json", core.NewError(content.Error())))
		}
		data := content.Value.([]byte)
		if len(core.Trim(string(data))) > 0 {
			if decoded := core.JSONUnmarshal(data, &manifest); !decoded.OK {
				return core.Fail(core.E("typescript.ensurePackageMetadata", "failed to parse package.json", core.NewError(decoded.Error())))
			}
		}
	}

	resolvedName := core.Trim(packageName)
	if resolvedName == "" {
		resolvedName = ax.Base(outputDir)
	}
	manifest["name"] = resolvedName

	if core.Trim(version) != "" {
		manifest["version"] = core.Trim(version)
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

	encoded := core.JSONMarshalIndent(manifest, "", "  ")
	if !encoded.OK {
		return core.Fail(core.E("typescript.ensurePackageMetadata", "failed to encode package.json", core.NewError(encoded.Error())))
	}
	written := ax.WriteFile(manifestPath, append(encoded.Value.([]byte), '\n'), 0o644)
	if !written.OK {
		return core.Fail(core.E("typescript.ensurePackageMetadata", "failed to write package.json", core.NewError(written.Error())))
	}

	return core.Ok(nil)
}

func copyTypeScriptDirectoryContents(sourceDir, destinationDir string) core.Result {
	entries := ax.ReadDir(sourceDir)
	if !entries.OK {
		return core.Fail(core.E("typescript.copyDirectoryContents", "failed to list source dir", core.NewError(entries.Error())))
	}

	for _, entry := range entries.Value.([]core.FsDirEntry) {
		sourcePath := ax.Join(sourceDir, entry.Name())
		destinationPath := ax.Join(destinationDir, entry.Name())
		copied := copyTypeScriptPath(sourcePath, destinationPath)
		if !copied.OK {
			return copied
		}
	}

	return core.Ok(nil)
}

func copyTypeScriptPath(sourcePath, destinationPath string) core.Result {
	info := ax.Stat(sourcePath)
	if !info.OK {
		return core.Fail(core.E("typescript.copyPath", "failed to stat source path", core.NewError(info.Error())))
	}
	fileInfo := info.Value.(core.FsFileInfo)

	if fileInfo.IsDir() {
		created := ax.MkdirAll(destinationPath, 0o755)
		if !created.OK {
			return core.Fail(core.E("typescript.copyPath", "failed to create destination dir", core.NewError(created.Error())))
		}
		return copyTypeScriptDirectoryContents(sourcePath, destinationPath)
	}

	created := ax.MkdirAll(ax.Dir(destinationPath), 0o755)
	if !created.OK {
		return core.Fail(core.E("typescript.copyPath", "failed to create file parent dir", core.NewError(created.Error())))
	}

	sourceFile := ax.Open(sourcePath)
	if !sourceFile.OK {
		return core.Fail(core.E("typescript.copyPath", "failed to open source file", core.NewError(sourceFile.Error())))
	}
	source := sourceFile.Value.(core.FsFile)
	defer func() { _ = source.Close() }()

	destinationFile := ax.Create(destinationPath)
	if !destinationFile.OK {
		return core.Fail(core.E("typescript.copyPath", "failed to create destination file", core.NewError(destinationFile.Error())))
	}
	destination := destinationFile.Value.(core.WriteCloser)
	defer func() { _ = destination.Close() }()

	if _, err := stdio.Copy(destination, source); err != nil {
		return core.Fail(core.E("typescript.copyPath", "failed to copy file", err))
	}
	chmod := ax.Chmod(destinationPath, fileInfo.Mode())
	if !chmod.OK {
		return core.Fail(core.E("typescript.copyPath", "failed to preserve file mode", core.NewError(chmod.Error())))
	}

	return core.Ok(nil)
}

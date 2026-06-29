// SPDX-License-Identifier: EUPL-1.2

package build

import (
	"bytes"         // AX-6 intrinsic: gzip kernel decompression buffer.
	"compress/gzip" // AX-6 intrinsic: linuxkit emits a gzip kernel; VZ needs it raw.
	"context"
	stdio "io" // AX-6 intrinsic: stream the decompressed kernel.
	"text/template" // AX-6 intrinsic: no core template primitive.

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	storage "dappco.re/go/build/pkg/storage"
)

// vzGuestKernel is the canonical kernel filename inside a resolved artefact
// directory — matches go-container vz.go vzResolveGuestArtefacts.
const vzGuestKernel = "kernel"

// vzGuestInitrd is the canonical initial-ramdisk filename inside a resolved
// artefact directory (vzResolveGuestArtefacts requires it).
const vzGuestInitrd = "initrd.img"

// vzGuestCmdline is the canonical kernel command-line filename inside a
// resolved artefact directory (optional for the guest; resolve always emits
// it when linuxkit produces one).
const vzGuestCmdline = "cmdline"

// linuxKitResolveDefault is the embedded VZ guest definition resolve renders
// when LinuxKitResolveConfig.Definition is empty. It is shipped under
// images/*.yml but is deliberately absent from linuxKitBaseCatalog, so the
// legacy `core build image` pipeline never sees it.
const linuxKitResolveDefault = "core-dev-vz"

// linuxKitResolveFormat is the linuxkit output format resolve always builds —
// the §4 guest contract (kernel + initrd.img + cmdline).
const linuxKitResolveFormat = "kernel+initrd"

// linuxKitResolveName is the --name passed to linuxkit. The output filenames
// are derived from it (<name>-kernel, <name>-initrd.img, <name>-cmdline),
// confirmed empirically against linuxkit v1.8.2.
const linuxKitResolveName = "vzguest"

// linuxKitResolveCacheFile is the signature sidecar resolve writes into the
// artefact directory so a later call can skip an unchanged build.
const linuxKitResolveCacheFile = ".vzguest-resolve.json"

// LinuxKitResolveTemplateData is the render input for a VZ guest definition.
// Only the staged vzagent binary path varies between builds today; the field
// is exported so callers reading the contract can see what the embedded def
// expects.
type LinuxKitResolveTemplateData struct {
	// VZAgentBinary is the staged path of the cross-compiled vzagent binary,
	// substituted into the definition's files: source.
	VZAgentBinary string
}

// LinuxKitResolveConfig drives a VZ guest-image resolve.
//
//	cfg := build.LinuxKitResolveConfig{
//	    VZAgentBinary: "/path/to/vzagent",   // cross-compiled GOOS=linux GOARCH=arm64
//	    OutputDir:     "/srv/core/vz/guest", // artefact directory the VM boots
//	}
type LinuxKitResolveConfig struct {
	// FS is the filesystem the artefact directory lives on. Nil uses
	// storage.Local.
	FS storage.Medium
	// BaseName names the embedded VZ guest definition to render. Empty uses
	// the default core-dev-vz definition.
	BaseName string
	// Definition is a verbatim linuxkit definition (rendered as a template).
	// When set it overrides BaseName — lets callers/tests supply a definition
	// without embedding one.
	Definition string
	// VZAgentBinary is the cross-compiled vzagent binary path
	// (CGO_ENABLED=0 GOOS=linux GOARCH=arm64). Required: the guest has no
	// control channel without it.
	VZAgentBinary string
	// OutputDir is the artefact directory resolve assembles — the directory
	// passed to the VZProvider as Image.Path. Required.
	OutputDir string
	// Rebuild forces a rebuild even when a matching cached artefact set
	// already exists in OutputDir.
	Rebuild bool
	// ProjectDir is the working directory the linuxkit build runs in (for
	// version derivation parity with the rest of the build system). Empty uses
	// the process working directory.
	ProjectDir string
}

// LinuxKitResolveResult reports a completed resolve.
type LinuxKitResolveResult struct {
	// Dir is the artefact directory satisfying the §4 guest contract — pass it
	// to the VZProvider as Image.Path.
	Dir string
	// Kernel is the resolved kernel path (Dir/kernel).
	Kernel string
	// Initrd is the resolved initial-ramdisk path (Dir/initrd.img).
	Initrd string
	// Cmdline is the resolved kernel-command-line path (Dir/cmdline); "" when
	// linuxkit produced none.
	Cmdline string
	// Cached reports whether the artefacts were reused (no linuxkit build ran).
	Cached bool
}

// linuxKitResolveExec runs `linuxkit build` for resolve. It is a package var
// so unit tests inject a fake without invoking the real CLI. Production resolves
// the linuxkit CLI path and execs it in projectDir; the build writes its three
// outputs (<name>-kernel, <name>-initrd.img, <name>-cmdline) into buildDir.
var linuxKitResolveExec = func(ctx context.Context, projectDir, buildDir, definitionPath, name string) core.Result {
	commandResult := (&linuxKitResolveCli{}).resolve()
	if !commandResult.OK {
		return commandResult
	}
	command := commandResult.Value.(string)
	args := []string{
		"build",
		"--format", linuxKitResolveFormat,
		"--name", name,
		"--dir", buildDir,
		definitionPath,
	}
	executed := ax.ExecWithEnv(ctx, projectDir, nil, command, args...)
	if !executed.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "linuxkit build failed", core.NewError(executed.Error())))
	}
	return core.Ok(nil)
}

// linuxKitResolveCli resolves the linuxkit CLI path, reusing the same search
// behaviour as the LinuxKitBuilder.
type linuxKitResolveCli struct{}

func (c *linuxKitResolveCli) resolve() core.Result {
	command := ax.ResolveCommand("linuxkit",
		"/usr/local/bin/linuxkit",
		"/opt/homebrew/bin/linuxkit",
	)
	if !command.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "linuxkit CLI not found. Install with: brew install linuxkit (macOS) or see https://github.com/linuxkit/linuxkit", core.NewError(command.Error())))
	}
	return command
}

// LinuxKitResolve builds (or returns a cached) VZ guest artefact set and yields
// the artefact directory satisfying go-container's vzResolveGuestArtefacts
// contract (kernel + initrd.img + cmdline). This is the non-stopgap source for
// core/agent's vzResolveImage.
//
// Resolve renders the embedded core-dev-vz definition (or cfg.Definition) with
// the staged vzagent binary, builds the kernel+initrd format with linuxkit,
// then renames linuxkit's <name>-kernel/<name>-initrd.img/<name>-cmdline outputs
// to the canonical kernel/initrd.img/cmdline names in cfg.OutputDir. A signature
// over the definition + the vzagent binary content guards the cache: an
// unchanged input set with kernel + initrd.img already present in OutputDir
// skips the build.
//
//	r := build.LinuxKitResolve(ctx, build.LinuxKitResolveConfig{
//	    VZAgentBinary: "/path/to/vzagent",
//	    OutputDir:     "/srv/core/vz/guest",
//	})
//	res := r.Value.(build.LinuxKitResolveResult)
//	// res.Dir → VZProvider Image.Path
func LinuxKitResolve(ctx context.Context, cfg LinuxKitResolveConfig) core.Result { // Value: LinuxKitResolveResult
	if ctx == nil {
		ctx = context.Background()
	}
	fs := cfg.FS
	if fs == nil {
		fs = storage.Local
	}

	if core.Trim(cfg.OutputDir) == "" {
		return core.Fail(core.E("build.LinuxKitResolve", "output directory is required", nil))
	}
	if core.Trim(cfg.VZAgentBinary) == "" {
		return core.Fail(core.E("build.LinuxKitResolve", "vzagent binary path is required (cross-compile CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/vzagent)", nil))
	}
	if !fs.IsFile(cfg.VZAgentBinary) {
		return core.Fail(core.E("build.LinuxKitResolve", "vzagent binary not found: "+cfg.VZAgentBinary, nil))
	}

	definitionResult := linuxKitResolveDefinition(cfg)
	if !definitionResult.OK {
		return definitionResult
	}
	definition := definitionResult.Value.(string)

	renderedResult := linuxKitResolveRender(definition, cfg.VZAgentBinary)
	if !renderedResult.OK {
		return renderedResult
	}
	rendered := renderedResult.Value.(string)

	binaryHashResult := linuxKitResolveFileHash(fs, cfg.VZAgentBinary)
	if !binaryHashResult.OK {
		return binaryHashResult
	}
	signature := linuxKitResolveSignature(rendered, binaryHashResult.Value.(string))

	result := LinuxKitResolveResult{
		Dir:     cfg.OutputDir,
		Kernel:  ax.Join(cfg.OutputDir, vzGuestKernel),
		Initrd:  ax.Join(cfg.OutputDir, vzGuestInitrd),
		Cmdline: ax.Join(cfg.OutputDir, vzGuestCmdline),
	}

	if !cfg.Rebuild && linuxKitResolveCacheValid(fs, cfg.OutputDir, signature) {
		result.Cached = true
		if !fs.IsFile(result.Cmdline) {
			result.Cmdline = ""
		}
		return core.Ok(result)
	}

	built := linuxKitResolveBuild(ctx, fs, cfg, rendered, signature)
	if !built.OK {
		return built
	}
	return built
}

// linuxKitResolveDefinition returns the verbatim definition to render: an
// explicit cfg.Definition wins, else the embedded base by name (default
// core-dev-vz). The named definition is read straight from the embedded
// images FS, not the catalog, so the VZ guest def stays invisible to the
// legacy image pipeline.
func linuxKitResolveDefinition(cfg LinuxKitResolveConfig) core.Result { // Value: string
	if core.Trim(cfg.Definition) != "" {
		return core.Ok(cfg.Definition)
	}
	name := core.Trim(cfg.BaseName)
	if name == "" {
		name = linuxKitResolveDefault
	}
	content, err := linuxKitBaseTemplateFS.ReadFile("images/" + name + ".yml")
	if err != nil {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to read embedded VZ guest definition: "+name, err))
	}
	return core.Ok(string(content))
}

// linuxKitResolveRender substitutes the staged vzagent binary path into the
// definition's {{ .VZAgentBinary }} placeholder.
func linuxKitResolveRender(definition, vzAgentBinary string) core.Result { // Value: string
	tmpl, parseFailure := template.New("vzguest").Parse(definition)
	if parseFailure != nil {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to parse VZ guest definition", parseFailure))
	}
	rendered := core.NewBuffer()
	if renderFailure := tmpl.Execute(rendered, LinuxKitResolveTemplateData{VZAgentBinary: vzAgentBinary}); renderFailure != nil {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to render VZ guest definition", renderFailure))
	}
	return core.Ok(rendered.String())
}

// linuxKitResolveBuild runs the linuxkit build in a staging directory, then
// assembles the canonical artefact set in cfg.OutputDir. linuxkit does NOT
// create its --dir (verified empirically — a missing dir fails the build), so
// the staging directory is created first.
func linuxKitResolveBuild(ctx context.Context, fs storage.Medium, cfg LinuxKitResolveConfig, rendered, signature string) core.Result { // Value: LinuxKitResolveResult
	stageResult := ax.TempDir("core-build-vzguest-*")
	if !stageResult.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to create build staging directory", core.NewError(stageResult.Error())))
	}
	stageDir := stageResult.Value.(string)
	defer ax.RemoveAll(stageDir)

	buildDir := ax.Join(stageDir, "out")
	if created := ax.MkdirAll(buildDir, 0o755); !created.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to create build output directory", core.NewError(created.Error())))
	}

	definitionPath := ax.Join(stageDir, "vzguest.yml")
	if written := ax.WriteString(definitionPath, rendered, 0o644); !written.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to write rendered VZ guest definition", core.NewError(written.Error())))
	}

	projectDir := cfg.ProjectDir
	if projectDir == "" {
		if wd := ax.Getwd(); wd.OK {
			projectDir = wd.Value.(string)
		}
	}

	if built := linuxKitResolveExec(ctx, projectDir, buildDir, definitionPath, linuxKitResolveName); !built.OK {
		return built
	}

	return linuxKitResolveAssemble(fs, buildDir, cfg.OutputDir, signature)
}

// linuxKitResolveAssemble maps linuxkit's <name>-kernel / <name>-initrd.img /
// <name>-cmdline outputs onto the canonical kernel / initrd.img / cmdline names
// inside outputDir, then writes the cache signature. kernel and initrd.img are
// required; a missing cmdline is tolerated (the guest falls back to its built-in
// default). The mapping is the load-bearing contract — confirmed against
// linuxkit v1.8.2: `build --format kernel+initrd --name N --dir D` emits
// D/N-kernel, D/N-initrd.img, D/N-cmdline.
//
// The kernel is also decompressed: linuxkit's kernel+initrd output is a gzip
// kernel, but go-container's vzResolveGuestArtefacts requires an uncompressed
// arm64 Image and VZLinuxBootLoader does no decompression — an unbootable dir
// otherwise. The initrd stays gzipped (VZ wants it compressed).
func linuxKitResolveAssemble(fs storage.Medium, buildDir, outputDir, signature string) core.Result { // Value: LinuxKitResolveResult
	if created := fs.EnsureDir(outputDir); !created.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to create artefact directory", core.NewError(created.Error())))
	}

	srcKernel := ax.Join(buildDir, linuxKitResolveName+"-kernel")
	srcInitrd := ax.Join(buildDir, linuxKitResolveName+"-initrd.img")
	srcCmdline := ax.Join(buildDir, linuxKitResolveName+"-cmdline")

	if !fs.IsFile(srcKernel) {
		return core.Fail(core.E("build.LinuxKitResolve", "linuxkit did not produce a kernel: "+srcKernel, nil))
	}
	if !fs.IsFile(srcInitrd) {
		return core.Fail(core.E("build.LinuxKitResolve", "linuxkit did not produce an initrd: "+srcInitrd, nil))
	}

	result := LinuxKitResolveResult{
		Dir:    outputDir,
		Kernel: ax.Join(outputDir, vzGuestKernel),
		Initrd: ax.Join(outputDir, vzGuestInitrd),
	}

	if decompressed := linuxKitResolveKernel(fs, srcKernel, result.Kernel); !decompressed.OK {
		return decompressed
	}
	if copied := linuxKitResolveCopy(fs, srcInitrd, result.Initrd); !copied.OK {
		return copied
	}
	if fs.IsFile(srcCmdline) {
		cmdlinePath := ax.Join(outputDir, vzGuestCmdline)
		if copied := linuxKitResolveCopy(fs, srcCmdline, cmdlinePath); !copied.OK {
			return copied
		}
		result.Cmdline = cmdlinePath
	}

	if written := fs.WriteMode(ax.Join(outputDir, linuxKitResolveCacheFile), signature, 0o644); !written.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to write resolve cache signature", core.NewError(written.Error())))
	}

	return core.Ok(result)
}

// linuxKitResolveCopy copies a build output into the artefact directory,
// preserving the source mode where the medium exposes it.
func linuxKitResolveCopy(fs storage.Medium, sourcePath, destinationPath string) core.Result { // Value: nil
	return CopyMediumPath(fs, sourcePath, fs, destinationPath)
}

// linuxKitResolveKernel writes the canonical, uncompressed kernel. linuxkit's
// kernel+initrd output is gzip-compressed (magic 1f 8b), but the §4 guest
// contract is an uncompressed arm64 Image (the Image magic 0x644d5241 sits at
// offset 56) and VZLinuxBootLoader boots it verbatim. A gzip kernel is
// transparently inflated; an already-raw kernel is copied through, so the helper
// is safe whatever linuxkit emits.
func linuxKitResolveKernel(fs storage.Medium, sourcePath, destinationPath string) core.Result { // Value: nil
	content := fs.Read(sourcePath)
	if !content.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to read kernel: "+sourcePath, core.NewError(content.Error())))
	}
	raw := []byte(content.Value.(string))
	if !linuxKitResolveIsGzip(raw) {
		// Already an uncompressed Image — copy through with its mode preserved.
		return linuxKitResolveCopy(fs, sourcePath, destinationPath)
	}

	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return core.Fail(core.E("build.LinuxKitResolve", "open gzip kernel reader", err))
	}
	defer reader.Close()
	decompressed, err := stdio.ReadAll(reader)
	if err != nil {
		return core.Fail(core.E("build.LinuxKitResolve", "decompress kernel", err))
	}

	if written := fs.WriteMode(destinationPath, string(decompressed), 0o644); !written.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "write decompressed kernel", core.NewError(written.Error())))
	}
	return core.Ok(nil)
}

// linuxKitResolveIsGzip reports whether b begins with the gzip magic (1f 8b).
func linuxKitResolveIsGzip(b []byte) bool {
	return len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b
}

// linuxKitResolveCacheValid reports whether outputDir already holds a matching
// artefact set: kernel + initrd.img present and the cache signature equal to
// the recomputed one. A missing or mismatched signature, or a missing kernel /
// initrd, means a rebuild is required.
func linuxKitResolveCacheValid(fs storage.Medium, outputDir, signature string) bool {
	if !fs.IsFile(ax.Join(outputDir, vzGuestKernel)) {
		return false
	}
	if !fs.IsFile(ax.Join(outputDir, vzGuestInitrd)) {
		return false
	}
	cachePath := ax.Join(outputDir, linuxKitResolveCacheFile)
	if !fs.IsFile(cachePath) {
		return false
	}
	content := fs.Read(cachePath)
	if !content.OK {
		return false
	}
	return core.Trim(content.Value.(string)) == signature
}

// linuxKitResolveFileHash returns the SHA-256 hex of a file's contents.
func linuxKitResolveFileHash(fs storage.Medium, path string) core.Result { // Value: string
	content := fs.Read(path)
	if !content.OK {
		return core.Fail(core.E("build.LinuxKitResolve", "failed to read file for signature: "+path, core.NewError(content.Error())))
	}
	return core.Ok(core.SHA256Hex([]byte(content.Value.(string))))
}

// linuxKitResolveSignature derives the cache signature from the rendered
// definition and the vzagent binary hash — the two inputs that determine the
// artefact set. A change in either invalidates the cache.
func linuxKitResolveSignature(rendered, binaryHash string) string {
	return core.SHA256Hex([]byte(core.Join("\n", rendered, binaryHash)))
}

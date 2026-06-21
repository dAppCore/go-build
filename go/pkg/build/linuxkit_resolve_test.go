// SPDX-License-Identifier: EUPL-1.2

package build

import (
	"bytes"
	"compress/gzip"
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	storage "dappco.re/go/build/pkg/storage"
)

// gzipBytes returns body wrapped in a gzip stream — used to mimic linuxkit's
// gzip kernel output in tests.
func gzipBytes(t *testing.T, body string) string {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.String()
}

// fakeLinuxKitResolveExec replaces the real linuxkit build with a stub that
// writes the three <name>-* outputs the kernel+initrd format produces, exactly
// as linuxkit v1.8.2 does. callCount tracks how many builds ran so the caching
// tests can assert no rebuild happened.
type fakeLinuxKitResolveExec struct {
	callCount   int
	kernelBody  string
	initrdBody  string
	cmdlineBody string
	omitCmdline bool
	omitKernel  bool
	failWith    string
}

func (f *fakeLinuxKitResolveExec) install(t *testing.T) {
	t.Helper()
	previous := linuxKitResolveExec
	t.Cleanup(func() { linuxKitResolveExec = previous })
	linuxKitResolveExec = func(_ context.Context, _, buildDir, _, name string) core.Result {
		f.callCount++
		if f.failWith != "" {
			return core.Fail(core.E("build.LinuxKitResolve", f.failWith, nil))
		}
		if !f.omitKernel {
			if w := ax.WriteString(ax.Join(buildDir, name+"-kernel"), f.kernelBody, 0o644); !w.OK {
				return w
			}
		}
		if w := ax.WriteString(ax.Join(buildDir, name+"-initrd.img"), f.initrdBody, 0o644); !w.OK {
			return w
		}
		if !f.omitCmdline {
			if w := ax.WriteString(ax.Join(buildDir, name+"-cmdline"), f.cmdlineBody, 0o644); !w.OK {
				return w
			}
		}
		return core.Ok(nil)
	}
}

func writeFakeVZAgent(t *testing.T, dir, body string) string {
	t.Helper()
	path := ax.Join(dir, "vzagent")
	if w := ax.WriteFile(path, []byte(body), 0o755); !w.OK {
		t.Fatalf("write fake vzagent: %v", w.Error())
	}
	return path
}

func TestBuild_LinuxKitResolve_Good(t *testing.T) {
	tmp := t.TempDir()
	agent := writeFakeVZAgent(t, tmp, "fake-vzagent-binary")
	outputDir := ax.Join(tmp, "guest")

	fake := &fakeLinuxKitResolveExec{
		kernelBody:  "KERNELBYTES",
		initrdBody:  "INITRDBYTES",
		cmdlineBody: "console=hvc0",
	}
	fake.install(t)

	result := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{
		VZAgentBinary: agent,
		OutputDir:     outputDir,
	})
	if !result.OK {
		t.Fatalf("resolve failed: %v", result.Error())
	}
	res := result.Value.(LinuxKitResolveResult)

	if !stdlibAssertEqual(1, fake.callCount) {
		t.Fatalf("want 1 build, got %d", fake.callCount)
	}
	if !stdlibAssertEqual(false, res.Cached) {
		t.Fatalf("first resolve should not be cached")
	}
	if !stdlibAssertEqual(ax.Join(outputDir, "kernel"), res.Kernel) {
		t.Fatalf("kernel path mismatch: %s", res.Kernel)
	}
	if !stdlibAssertEqual(ax.Join(outputDir, "initrd.img"), res.Initrd) {
		t.Fatalf("initrd path mismatch: %s", res.Initrd)
	}
	if !stdlibAssertEqual(ax.Join(outputDir, "cmdline"), res.Cmdline) {
		t.Fatalf("cmdline path mismatch: %s", res.Cmdline)
	}

	// The canonical names must exist on disk with the build's contents — this
	// is the vzResolveGuestArtefacts contract.
	if k := storage.Local.Read(res.Kernel); !k.OK || !stdlibAssertEqual("KERNELBYTES", k.Value.(string)) {
		t.Fatalf("kernel not assembled correctly")
	}
	if i := storage.Local.Read(res.Initrd); !i.OK || !stdlibAssertEqual("INITRDBYTES", i.Value.(string)) {
		t.Fatalf("initrd not assembled correctly")
	}
	if c := storage.Local.Read(res.Cmdline); !c.OK || !stdlibAssertEqual("console=hvc0", c.Value.(string)) {
		t.Fatalf("cmdline not assembled correctly")
	}
}

func TestBuild_LinuxKitResolve_DecompressesGzipKernel_Good(t *testing.T) {
	tmp := t.TempDir()
	agent := writeFakeVZAgent(t, tmp, "fake-vzagent-binary")
	outputDir := ax.Join(tmp, "guest")

	// linuxkit emits a gzip kernel; resolve must inflate it to a raw Image,
	// while the initrd stays gzipped.
	rawKernel := "RAW-ARM64-IMAGE-BYTES"
	fake := &fakeLinuxKitResolveExec{
		kernelBody:  gzipBytes(t, rawKernel),
		initrdBody:  gzipBytes(t, "INITRD-STAYS-GZIPPED"),
		cmdlineBody: "console=hvc0",
	}
	fake.install(t)

	r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: outputDir})
	if !r.OK {
		t.Fatalf("resolve failed: %v", r.Error())
	}
	res := r.Value.(LinuxKitResolveResult)

	kernel := storage.Local.Read(res.Kernel)
	if !kernel.OK {
		t.Fatalf("read assembled kernel: %v", kernel.Error())
	}
	if !stdlibAssertEqual(rawKernel, kernel.Value.(string)) {
		t.Fatalf("kernel was not decompressed: got %q", kernel.Value.(string))
	}
	// The assembled kernel must NOT carry the gzip magic any more.
	if linuxKitResolveIsGzip([]byte(kernel.Value.(string))) {
		t.Fatalf("assembled kernel is still gzip-compressed")
	}
	// The initrd must remain gzipped (VZ wants it compressed).
	initrd := storage.Local.Read(res.Initrd)
	if !initrd.OK || !linuxKitResolveIsGzip([]byte(initrd.Value.(string))) {
		t.Fatalf("initrd should remain gzip-compressed")
	}
}

func TestBuild_LinuxKitResolveKernel_RawPassThrough_Good(t *testing.T) {
	tmp := t.TempDir()
	src := ax.Join(tmp, "vzguest-kernel")
	if w := ax.WriteFile(src, []byte("ALREADY-RAW-IMAGE"), 0o644); !w.OK {
		t.Fatalf("write raw kernel: %v", w.Error())
	}
	dst := ax.Join(tmp, "kernel")
	if r := linuxKitResolveKernel(storage.Local, src, dst); !r.OK {
		t.Fatalf("kernel copy-through failed: %v", r.Error())
	}
	out := storage.Local.Read(dst)
	if !out.OK || !stdlibAssertEqual("ALREADY-RAW-IMAGE", out.Value.(string)) {
		t.Fatalf("raw kernel was not passed through unchanged")
	}
}

func TestBuild_LinuxKitResolveIsGzip_Good(t *testing.T) {
	if !linuxKitResolveIsGzip([]byte{0x1f, 0x8b, 0x08, 0x00}) {
		t.Fatalf("gzip magic not detected")
	}
	if linuxKitResolveIsGzip([]byte{0x41, 0x52, 0x4d, 0x64}) {
		t.Fatalf("raw Image magic must not be read as gzip")
	}
	if linuxKitResolveIsGzip([]byte{0x1f}) {
		t.Fatalf("a single byte must not be read as gzip")
	}
}

func TestBuild_LinuxKitResolve_CachesSecondCall_Good(t *testing.T) {
	tmp := t.TempDir()
	agent := writeFakeVZAgent(t, tmp, "fake-vzagent-binary")
	outputDir := ax.Join(tmp, "guest")

	fake := &fakeLinuxKitResolveExec{kernelBody: "K", initrdBody: "I", cmdlineBody: "C"}
	fake.install(t)

	first := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: outputDir})
	if !first.OK {
		t.Fatalf("first resolve failed: %v", first.Error())
	}

	second := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: outputDir})
	if !second.OK {
		t.Fatalf("second resolve failed: %v", second.Error())
	}
	res := second.Value.(LinuxKitResolveResult)

	if !stdlibAssertEqual(1, fake.callCount) {
		t.Fatalf("second resolve must reuse cache; builds=%d", fake.callCount)
	}
	if !stdlibAssertEqual(true, res.Cached) {
		t.Fatalf("second resolve should be cached")
	}
}

func TestBuild_LinuxKitResolve_RebuildForcesBuild_Good(t *testing.T) {
	tmp := t.TempDir()
	agent := writeFakeVZAgent(t, tmp, "fake-vzagent-binary")
	outputDir := ax.Join(tmp, "guest")

	fake := &fakeLinuxKitResolveExec{kernelBody: "K", initrdBody: "I", cmdlineBody: "C"}
	fake.install(t)

	if r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: outputDir}); !r.OK {
		t.Fatalf("first resolve failed: %v", r.Error())
	}
	r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: outputDir, Rebuild: true})
	if !r.OK {
		t.Fatalf("rebuild resolve failed: %v", r.Error())
	}
	if !stdlibAssertEqual(2, fake.callCount) {
		t.Fatalf("rebuild must run a build; builds=%d", fake.callCount)
	}
	if !stdlibAssertEqual(false, r.Value.(LinuxKitResolveResult).Cached) {
		t.Fatalf("rebuild result should not report cached")
	}
}

func TestBuild_LinuxKitResolve_ChangedBinaryInvalidatesCache_Good(t *testing.T) {
	tmp := t.TempDir()
	agent := writeFakeVZAgent(t, tmp, "vzagent-v1")
	outputDir := ax.Join(tmp, "guest")

	fake := &fakeLinuxKitResolveExec{kernelBody: "K", initrdBody: "I", cmdlineBody: "C"}
	fake.install(t)

	if r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: outputDir}); !r.OK {
		t.Fatalf("first resolve failed: %v", r.Error())
	}

	// Rewriting the binary changes its content hash → signature → rebuild.
	writeFakeVZAgent(t, tmp, "vzagent-v2-different")
	r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: outputDir})
	if !r.OK {
		t.Fatalf("post-change resolve failed: %v", r.Error())
	}
	if !stdlibAssertEqual(2, fake.callCount) {
		t.Fatalf("changed binary must invalidate cache; builds=%d", fake.callCount)
	}
}

func TestBuild_LinuxKitResolve_NoCmdline_Good(t *testing.T) {
	tmp := t.TempDir()
	agent := writeFakeVZAgent(t, tmp, "fake-vzagent-binary")
	outputDir := ax.Join(tmp, "guest")

	fake := &fakeLinuxKitResolveExec{kernelBody: "K", initrdBody: "I", omitCmdline: true}
	fake.install(t)

	r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: outputDir})
	if !r.OK {
		t.Fatalf("resolve failed: %v", r.Error())
	}
	res := r.Value.(LinuxKitResolveResult)
	// A missing cmdline is tolerated — Cmdline blank, kernel + initrd still present.
	if !stdlibAssertEmpty(res.Cmdline) {
		t.Fatalf("cmdline should be empty when linuxkit produced none, got %q", res.Cmdline)
	}
	if !storage.Local.IsFile(res.Kernel) || !storage.Local.IsFile(res.Initrd) {
		t.Fatalf("kernel and initrd must still be assembled without a cmdline")
	}
}

func TestBuild_LinuxKitResolve_MissingOutputDir_Bad(t *testing.T) {
	tmp := t.TempDir()
	agent := writeFakeVZAgent(t, tmp, "x")
	r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent})
	if r.OK {
		t.Fatalf("expected failure when output directory is missing")
	}
	if !stdlibAssertContains(r.Error(), "output directory is required") {
		t.Fatalf("unexpected error: %v", r.Error())
	}
}

func TestBuild_LinuxKitResolve_MissingVZAgent_Bad(t *testing.T) {
	r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{OutputDir: t.TempDir()})
	if r.OK {
		t.Fatalf("expected failure when vzagent binary path is missing")
	}
	if !stdlibAssertContains(r.Error(), "vzagent binary path is required") {
		t.Fatalf("unexpected error: %v", r.Error())
	}
}

func TestBuild_LinuxKitResolve_VZAgentNotFound_Bad(t *testing.T) {
	r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{
		VZAgentBinary: ax.Join(t.TempDir(), "does-not-exist"),
		OutputDir:     t.TempDir(),
	})
	if r.OK {
		t.Fatalf("expected failure when vzagent binary does not exist")
	}
	if !stdlibAssertContains(r.Error(), "vzagent binary not found") {
		t.Fatalf("unexpected error: %v", r.Error())
	}
}

func TestBuild_LinuxKitResolve_BuildProducesNoKernel_Bad(t *testing.T) {
	tmp := t.TempDir()
	agent := writeFakeVZAgent(t, tmp, "x")
	outputDir := ax.Join(tmp, "guest")

	fake := &fakeLinuxKitResolveExec{initrdBody: "I", omitKernel: true}
	fake.install(t)

	r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: outputDir})
	if r.OK {
		t.Fatalf("expected failure when linuxkit produced no kernel")
	}
	if !stdlibAssertContains(r.Error(), "did not produce a kernel") {
		t.Fatalf("unexpected error: %v", r.Error())
	}
}

func TestBuild_LinuxKitResolve_BuildFails_Bad(t *testing.T) {
	tmp := t.TempDir()
	agent := writeFakeVZAgent(t, tmp, "x")

	fake := &fakeLinuxKitResolveExec{failWith: "linuxkit build failed"}
	fake.install(t)

	r := LinuxKitResolve(context.Background(), LinuxKitResolveConfig{VZAgentBinary: agent, OutputDir: ax.Join(tmp, "guest")})
	if r.OK {
		t.Fatalf("expected failure when the linuxkit build fails")
	}
	if !stdlibAssertContains(r.Error(), "linuxkit build failed") {
		t.Fatalf("unexpected error: %v", r.Error())
	}
}

func TestBuild_LinuxKitResolveRender_SubstitutesBinary_Good(t *testing.T) {
	rendered := linuxKitResolveRender("source: \"{{ .VZAgentBinary }}\"\n", "/staged/vzagent")
	if !rendered.OK {
		t.Fatalf("render failed: %v", rendered.Error())
	}
	if !stdlibAssertContains(rendered.Value.(string), "/staged/vzagent") {
		t.Fatalf("rendered definition missing the binary path: %s", rendered.Value.(string))
	}
}

func TestBuild_LinuxKitResolveRender_BadTemplate_Bad(t *testing.T) {
	rendered := linuxKitResolveRender("{{ .Unterminated", "/staged/vzagent")
	if rendered.OK {
		t.Fatalf("expected a parse failure for a malformed template")
	}
}

func TestBuild_LinuxKitResolveSignature_Deterministic_Good(t *testing.T) {
	a := linuxKitResolveSignature("def-A", "hash-1")
	b := linuxKitResolveSignature("def-A", "hash-1")
	c := linuxKitResolveSignature("def-A", "hash-2")
	if !stdlibAssertEqual(a, b) {
		t.Fatalf("identical inputs must produce identical signatures")
	}
	if a == c {
		t.Fatalf("a changed binary hash must change the signature")
	}
}

func TestBuild_LinuxKitResolveDefinition_EmbeddedDefault_Good(t *testing.T) {
	// The embedded core-dev-vz definition must be readable and render with the
	// vzagent placeholder filled — proves the dormant images/*.yml entry is
	// wired to resolve (and never to the legacy catalog).
	def := linuxKitResolveDefinition(LinuxKitResolveConfig{})
	if !def.OK {
		t.Fatalf("embedded VZ guest definition not readable: %v", def.Error())
	}
	if !stdlibAssertContains(def.Value.(string), "{{ .VZAgentBinary }}") {
		t.Fatalf("embedded definition missing the vzagent placeholder")
	}
	if !stdlibAssertContains(def.Value.(string), "virtiofs") {
		t.Fatalf("embedded definition missing the virtio-fs workspace mount")
	}

	rendered := linuxKitResolveRender(def.Value.(string), "/usr/local/bin/vzagent")
	if !rendered.OK {
		t.Fatalf("embedded definition failed to render: %v", rendered.Error())
	}
	if !stdlibAssertContains(rendered.Value.(string), "/usr/local/bin/vzagent") {
		t.Fatalf("rendered embedded definition missing the staged binary path")
	}
	// The default definition must NOT be registered in the legacy catalog.
	if _, ok := LookupLinuxKitBaseImage(linuxKitResolveDefault); ok {
		t.Fatalf("VZ guest definition %q must stay out of linuxKitBaseCatalog", linuxKitResolveDefault)
	}
}

func TestBuild_LinuxKitResolveDefinition_UnknownBase_Bad(t *testing.T) {
	def := linuxKitResolveDefinition(LinuxKitResolveConfig{BaseName: "no-such-vz-image"})
	if def.OK {
		t.Fatalf("expected failure for an unknown embedded definition")
	}
}

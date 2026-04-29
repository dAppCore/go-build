package build

import (
	"context"
	"runtime"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	coreio "dappco.re/go/io"
)

func TestBuiltinResolver_GoBuilder_Name_Good(t *core.T) {
	builder := &builtinGoBuilder{}
	name := builder.Name()
	core.AssertEqual(t, "go", name)
	core.AssertNotEmpty(t, name)
}

func TestBuiltinResolver_GoBuilder_Name_Bad(t *core.T) {
	builder := &builtinGoBuilder{}
	name := builder.Name()
	core.AssertNotEqual(t, "", name)
	core.AssertLen(t, name, 2)
}

func TestBuiltinResolver_GoBuilder_Name_Ugly(t *core.T) {
	var builder *builtinGoBuilder
	name := builder.Name()
	core.AssertEqual(t, "go", name)
	core.AssertNotEmpty(t, name)
}

func TestBuiltinResolver_GoBuilder_Detect_Good(t *core.T) {
	dir := t.TempDir()
	writeBuiltinResolverFile(t, ax.Join(dir, "go.mod"), "module example.com/demo\n")

	detected, err := (&builtinGoBuilder{}).Detect(coreio.Local, dir)
	core.RequireNoError(t, err)
	core.AssertTrue(t, detected)
}

func TestBuiltinResolver_GoBuilder_Detect_Bad(t *core.T) {
	detected, err := (&builtinGoBuilder{}).Detect(coreio.Local, t.TempDir())
	core.RequireNoError(t, err)
	core.AssertFalse(t, detected)
}

func TestBuiltinResolver_GoBuilder_Detect_Ugly(t *core.T) {
	detected, err := (&builtinGoBuilder{}).Detect(nil, "")
	core.RequireNoError(t, err)
	core.AssertFalse(t, detected)
}

func TestBuiltinResolver_GoBuilder_Build_Good(t *core.T) {
	dir := t.TempDir()
	writeBuiltinResolverFile(t, ax.Join(dir, "go.mod"), "module example.com/demo\n\ngo 1.23\n")
	writeBuiltinResolverFile(t, ax.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")

	artifacts, err := (&builtinGoBuilder{}).Build(context.Background(), &Config{
		FS:         coreio.Local,
		ProjectDir: dir,
		OutputDir:  ax.Join(dir, "dist"),
		Name:       "demo",
		Project:    Project{Main: "."},
	}, []Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}})
	core.RequireNoError(t, err)
	core.AssertLen(t, artifacts, 1)
	core.AssertEqual(t, runtime.GOOS+"/"+runtime.GOARCH, artifacts[0].OS+"/"+artifacts[0].Arch)
}

func TestBuiltinResolver_GoBuilder_Build_Bad(t *core.T) {
	artifacts, err := (&builtinGoBuilder{}).Build(context.Background(), nil, nil)
	core.AssertError(t, err)
	core.AssertNil(t, artifacts)
}

func TestBuiltinResolver_GoBuilder_Build_Ugly(t *core.T) {
	dir := t.TempDir()
	artifacts, err := (&builtinGoBuilder{}).Build(context.Background(), &Config{
		FS:         coreio.Local,
		ProjectDir: dir,
		OutputDir:  ax.Join(dir, "dist"),
		Name:       "demo",
	}, []Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}})
	core.AssertError(t, err)
	core.AssertEmpty(t, artifacts)
}

func writeBuiltinResolverFile(t *core.T, path, content string) {
	t.Helper()
	core.RequireNoError(t, ax.MkdirAll(ax.Dir(path), 0o755))
	core.RequireNoError(t, ax.WriteFile(path, []byte(content), 0o644))
}

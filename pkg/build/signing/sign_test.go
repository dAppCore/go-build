package signing

import (
	"context"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func TestSign_SignBinaries_Good(t *core.T) {
	cfg := SignConfig{Enabled: false}
	result := SignBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, []Artifact{{Path: "dist/app", OS: "linux", Arch: "amd64"}})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, false, cfg.Enabled)
}

func TestSign_SignBinaries_Bad(t *core.T) {
	cfg := SignConfig{Enabled: true}
	result := SignBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, nil)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, true, cfg.Enabled)
}

func TestSign_SignBinaries_Ugly(t *core.T) {
	artifacts := []Artifact{{}}
	result := SignBinaries(context.Background(), nil, SignConfig{Enabled: false}, artifacts)
	core.AssertTrue(t, result.OK)
	core.AssertLen(t, artifacts, 1)
}

func TestSign_NotarizeBinaries_Good(t *core.T) {
	cfg := SignConfig{Enabled: false}
	result := NotarizeBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, []Artifact{{Path: "dist/app.zip", OS: "darwin", Arch: "arm64"}})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, false, cfg.Enabled)
}

func TestSign_NotarizeBinaries_Bad(t *core.T) {
	cfg := SignConfig{Enabled: true}
	result := NotarizeBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, nil)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, true, cfg.Enabled)
}

func TestSign_NotarizeBinaries_Ugly(t *core.T) {
	cfg := SignConfig{Enabled: true, MacOS: MacOSConfig{Notarize: false}}
	result := NotarizeBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, []Artifact{{Path: "dist/app.zip", OS: "darwin"}})
	core.AssertTrue(t, result.OK)
	core.AssertFalse(t, cfg.MacOS.Notarize)
}

func TestSign_SignChecksums_Good(t *core.T) {
	cfg := SignConfig{Enabled: false}
	result := SignChecksums(context.Background(), coreio.NewMemoryMedium(), cfg, "CHECKSUMS.txt")
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, false, cfg.Enabled)
}

func TestSign_SignChecksums_Bad(t *core.T) {
	cfg := SignConfig{Enabled: true}
	result := SignChecksums(context.Background(), coreio.NewMemoryMedium(), cfg, "")
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, true, cfg.Enabled)
}

func TestSign_SignChecksums_Ugly(t *core.T) {
	checksumFile := ""
	result := SignChecksums(context.Background(), nil, SignConfig{Enabled: false}, checksumFile)
	core.AssertTrue(t, result.OK)
	core.AssertEmpty(t, checksumFile)
}

package signing

import (
	"context"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func TestSign_SignBinaries_Good(t *core.T) {
	cfg := SignConfig{Enabled: false}
	err := SignBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, []Artifact{{Path: "dist/app", OS: "linux", Arch: "amd64"}})
	core.AssertNoError(t, err)
	core.AssertEqual(t, false, cfg.Enabled)
}

func TestSign_SignBinaries_Bad(t *core.T) {
	cfg := SignConfig{Enabled: true}
	err := SignBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, true, cfg.Enabled)
}

func TestSign_SignBinaries_Ugly(t *core.T) {
	artifacts := []Artifact{{}}
	err := SignBinaries(context.Background(), nil, SignConfig{Enabled: false}, artifacts)
	core.AssertNoError(t, err)
	core.AssertLen(t, artifacts, 1)
}

func TestSign_NotarizeBinaries_Good(t *core.T) {
	cfg := SignConfig{Enabled: false}
	err := NotarizeBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, []Artifact{{Path: "dist/app.zip", OS: "darwin", Arch: "arm64"}})
	core.AssertNoError(t, err)
	core.AssertEqual(t, false, cfg.Enabled)
}

func TestSign_NotarizeBinaries_Bad(t *core.T) {
	cfg := SignConfig{Enabled: true}
	err := NotarizeBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, true, cfg.Enabled)
}

func TestSign_NotarizeBinaries_Ugly(t *core.T) {
	cfg := SignConfig{Enabled: true, MacOS: MacOSConfig{Notarize: false}}
	err := NotarizeBinaries(context.Background(), coreio.NewMemoryMedium(), cfg, []Artifact{{Path: "dist/app.zip", OS: "darwin"}})
	core.AssertNoError(t, err)
	core.AssertFalse(t, cfg.MacOS.Notarize)
}

func TestSign_SignChecksums_Good(t *core.T) {
	cfg := SignConfig{Enabled: false}
	err := SignChecksums(context.Background(), coreio.NewMemoryMedium(), cfg, "CHECKSUMS.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, false, cfg.Enabled)
}

func TestSign_SignChecksums_Bad(t *core.T) {
	cfg := SignConfig{Enabled: true}
	err := SignChecksums(context.Background(), coreio.NewMemoryMedium(), cfg, "")
	core.AssertNoError(t, err)
	core.AssertEqual(t, true, cfg.Enabled)
}

func TestSign_SignChecksums_Ugly(t *core.T) {
	checksumFile := ""
	err := SignChecksums(context.Background(), nil, SignConfig{Enabled: false}, checksumFile)
	core.AssertNoError(t, err)
	core.AssertEmpty(t, checksumFile)
}

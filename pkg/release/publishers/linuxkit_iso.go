package publishers

import (
	"context"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	coreerr "dappco.re/go/log"
)

func (p *LinuxKitPublisher) publishLinuxKitArtifact(ctx context.Context, release *Release, cfg LinuxKitConfig, format, artifactPath string) error {
	switch {
	case isLinuxKitIsoFormat(format):
		return p.publishIso(ctx, release, artifactPath)
	case isLinuxKitQcow2Format(format):
		return p.publishQcow2(ctx, release, cfg, artifactPath)
	case isLinuxKitRawFormat(format):
		return p.publishRaw(ctx, release, artifactPath)
	case format == "aws":
		return p.publishAWS(ctx, release, cfg, artifactPath)
	case format == "gcp":
		return p.publishGCP(ctx, release, cfg, artifactPath)
	default:
		return p.publishLocalLinuxKitArtifact(release, artifactPath, format)
	}
}

func (p *LinuxKitPublisher) publishIso(ctx context.Context, release *Release, artifactPath string) error {
	_ = ctx
	return p.publishLocalLinuxKitArtifact(release, artifactPath, "iso")
}

func (p *LinuxKitPublisher) publishLocalLinuxKitArtifact(release *Release, artifactPath, format string) error {
	if err := p.ensureLinuxKitArtifact(release, artifactPath); err != nil {
		return err
	}

	publisherPrint("Produced LinuxKit %s artifact: %s", format, ax.Base(artifactPath))
	if core.HasSuffix(artifactPath, ".docker.tar") {
		publisherPrint("  Load with: docker load < %s", ax.Base(artifactPath))
	}

	return nil
}

func (p *LinuxKitPublisher) ensureLinuxKitArtifact(release *Release, artifactPath string) error {
	if release == nil || release.FS == nil {
		return coreerr.E("linuxkit.Publish", "release filesystem (FS) is nil", nil)
	}
	if !release.FS.Exists(artifactPath) {
		return coreerr.E("linuxkit.Publish", "artifact not found after build: "+artifactPath, nil)
	}
	return nil
}

func isLinuxKitIsoFormat(format string) bool {
	switch format {
	case "iso", "iso-bios", "iso-efi":
		return true
	default:
		return false
	}
}

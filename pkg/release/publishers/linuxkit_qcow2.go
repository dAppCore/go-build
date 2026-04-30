package publishers

import (
	"context"

	"dappco.re/go"
)

func (p *LinuxKitPublisher) publishQcow2(ctx context.Context, release *Release, cfg LinuxKitConfig, artifactPath string) core.Result {
	published := p.publishLocalLinuxKitArtifact(release, artifactPath, "qcow2")
	if !published.OK {
		return published
	}

	for _, target := range linuxKitCloudTargets(cfg, "aws") {
		uploaded := p.uploadLinuxKitS3(ctx, release, target, artifactPath)
		if !uploaded.OK {
			return uploaded
		}
	}
	for _, target := range linuxKitCloudTargets(cfg, "gcp") {
		uploaded := p.uploadLinuxKitGCS(ctx, release, target, artifactPath)
		if !uploaded.OK {
			return uploaded
		}
	}

	return core.Ok(nil)
}

func isLinuxKitQcow2Format(format string) bool {
	switch format {
	case "qcow2", "qcow2-bios", "qcow2-efi":
		return true
	default:
		return false
	}
}

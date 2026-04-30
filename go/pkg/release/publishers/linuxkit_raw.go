package publishers

import (
	"context"

	"dappco.re/go"
)

func (p *LinuxKitPublisher) publishRaw(ctx context.Context, release *Release, artifactPath string) core.Result {
	_ = ctx
	return p.publishLocalLinuxKitArtifact(release, artifactPath, "raw")
}

func isLinuxKitRawFormat(format string) bool {
	switch format {
	case "raw", "raw-bios", "raw-efi":
		return true
	default:
		return false
	}
}

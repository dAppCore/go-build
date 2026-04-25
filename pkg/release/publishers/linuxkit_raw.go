package publishers

import "context"

func (p *LinuxKitPublisher) publishRaw(ctx context.Context, release *Release, artifactPath string) error {
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

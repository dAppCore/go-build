package publishers

import "context"

func (p *LinuxKitPublisher) publishQcow2(ctx context.Context, release *Release, cfg LinuxKitConfig, artifactPath string) error {
	if err := p.publishLocalLinuxKitArtifact(release, artifactPath, "qcow2"); err != nil {
		return err
	}

	for _, target := range linuxKitCloudTargets(cfg, "aws") {
		if err := p.uploadLinuxKitS3(ctx, release, target, artifactPath); err != nil {
			return err
		}
	}
	for _, target := range linuxKitCloudTargets(cfg, "gcp") {
		if err := p.uploadLinuxKitGCS(ctx, release, target, artifactPath); err != nil {
			return err
		}
	}

	return nil
}

func isLinuxKitQcow2Format(format string) bool {
	switch format {
	case "qcow2", "qcow2-bios", "qcow2-efi":
		return true
	default:
		return false
	}
}

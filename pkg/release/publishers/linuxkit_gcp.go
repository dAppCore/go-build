package publishers

import (
	"context"

	"dappco.re/go/build/internal/ax"
	coreerr "dappco.re/go/log"
)

func (p *LinuxKitPublisher) publishGCP(ctx context.Context, release *Release, cfg LinuxKitConfig, artifactPath string) error {
	if err := p.ensureLinuxKitArtifact(release, artifactPath); err != nil {
		return err
	}

	targets := linuxKitCloudTargets(cfg, "gcp")
	if len(targets) == 0 {
		return coreerr.E("linuxkit.publishGCP", "gcp target bucket is required", nil)
	}

	for _, target := range targets {
		if err := p.uploadLinuxKitGCS(ctx, release, target, artifactPath); err != nil {
			return err
		}
	}

	return nil
}

func (p *LinuxKitPublisher) uploadLinuxKitGCS(ctx context.Context, release *Release, target LinuxKitTarget, artifactPath string) error {
	if target.Bucket == "" {
		return coreerr.E("linuxkit.uploadGCS", "gcp target bucket is required", nil)
	}

	gcloudCommand, err := resolveGCloudCli()
	if err != nil {
		return err
	}

	objectKey := linuxKitObjectKey(target, artifactPath)
	destination := linuxKitCloudURI("gs", target.Bucket, objectKey)
	if err := publisherRun(ctx, release.ProjectDir, nil, gcloudCommand, "storage", "cp", artifactPath, destination); err != nil {
		return coreerr.E("linuxkit.uploadGCS", "failed to upload "+ax.Base(artifactPath)+" to "+destination, err)
	}

	publisherPrint("Uploaded LinuxKit GCP image: %s", destination)
	return nil
}

func resolveGCloudCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/gcloud",
			"/opt/homebrew/bin/gcloud",
			"/usr/bin/gcloud",
		}
	}

	command, err := ax.ResolveCommand("gcloud", paths...)
	if err != nil {
		return "", coreerr.E("linuxkit.resolveGCloudCli", "gcloud CLI not found. Install it from https://cloud.google.com/sdk/docs/install", err)
	}

	return command, nil
}

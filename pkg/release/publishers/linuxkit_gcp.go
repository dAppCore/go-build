package publishers

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

func (p *LinuxKitPublisher) publishGCP(ctx context.Context, release *Release, cfg LinuxKitConfig, artifactPath string) core.Result {
	ensured := p.ensureLinuxKitArtifact(release, artifactPath)
	if !ensured.OK {
		return ensured
	}

	targets := linuxKitCloudTargets(cfg, "gcp")
	if len(targets) == 0 {
		return core.Fail(core.E("linuxkit.publishGCP", "gcp target bucket is required", nil))
	}

	for _, target := range targets {
		uploaded := p.uploadLinuxKitGCS(ctx, release, target, artifactPath)
		if !uploaded.OK {
			return uploaded
		}
	}

	return core.Ok(nil)
}

func (p *LinuxKitPublisher) uploadLinuxKitGCS(ctx context.Context, release *Release, target LinuxKitTarget, artifactPath string) core.Result {
	if target.Bucket == "" {
		return core.Fail(core.E("linuxkit.uploadGCS", "gcp target bucket is required", nil))
	}

	gcloudCommandResult := resolveGCloudCli()
	if !gcloudCommandResult.OK {
		return gcloudCommandResult
	}
	gcloudCommand := gcloudCommandResult.Value.(string)

	objectKey := linuxKitObjectKey(target, artifactPath)
	destination := linuxKitCloudURI("gs", target.Bucket, objectKey)
	uploaded := publisherRun(ctx, release.ProjectDir, nil, gcloudCommand, "storage", "cp", artifactPath, destination)
	if !uploaded.OK {
		return core.Fail(core.E("linuxkit.uploadGCS", "failed to upload "+ax.Base(artifactPath)+" to "+destination, core.NewError(uploaded.Error())))
	}

	publisherPrint("Uploaded LinuxKit GCP image: %s", destination)
	return core.Ok(nil)
}

func resolveGCloudCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/gcloud",
			"/opt/homebrew/bin/gcloud",
			"/usr/bin/gcloud",
		}
	}

	command := ax.ResolveCommand("gcloud", paths...)
	if !command.OK {
		return core.Fail(core.E("linuxkit.resolveGCloudCli", "gcloud CLI not found. Install it from https://cloud.google.com/sdk/docs/install", core.NewError(command.Error())))
	}

	return command
}

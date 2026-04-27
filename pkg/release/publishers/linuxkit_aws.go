package publishers

import (
	"context"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	coreerr "dappco.re/go/log"
)

func (p *LinuxKitPublisher) publishAWS(ctx context.Context, release *Release, cfg LinuxKitConfig, artifactPath string) error {
	if err := p.ensureLinuxKitArtifact(release, artifactPath); err != nil {
		return err
	}

	targets := linuxKitCloudTargets(cfg, "aws")
	if len(targets) == 0 {
		return coreerr.E("linuxkit.publishAWS", "aws target bucket is required", nil)
	}

	for _, target := range targets {
		if err := p.uploadLinuxKitS3(ctx, release, target, artifactPath); err != nil {
			return err
		}
	}

	return nil
}

func (p *LinuxKitPublisher) uploadLinuxKitS3(ctx context.Context, release *Release, target LinuxKitTarget, artifactPath string) error {
	if target.Bucket == "" {
		return coreerr.E("linuxkit.uploadS3", "aws target bucket is required", nil)
	}

	awsCommand, err := resolveAWSCli()
	if err != nil {
		return err
	}

	objectKey := linuxKitObjectKey(target, artifactPath)
	destination := linuxKitCloudURI("s3", target.Bucket, objectKey)
	args := []string{"s3", "cp", artifactPath, destination}
	if target.Region != "" {
		args = append(args, "--region", target.Region)
	}

	if err := publisherRun(ctx, release.ProjectDir, nil, awsCommand, args...); err != nil {
		return coreerr.E("linuxkit.uploadS3", "failed to upload "+ax.Base(artifactPath)+" to "+destination, err)
	}

	publisherPrint("Uploaded LinuxKit AWS image: %s", destination)
	return nil
}

func resolveAWSCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/aws",
			"/opt/homebrew/bin/aws",
		}
	}

	command, err := ax.ResolveCommand("aws", paths...)
	if err != nil {
		return "", coreerr.E("linuxkit.resolveAWSCli", "aws CLI not found. Install it from https://aws.amazon.com/cli/", err)
	}

	return command, nil
}

func appendLinuxKitTargetValue(cfg *LinuxKitConfig, provider string, value any) {
	target, ok := linuxKitTargetFromAny(provider, value)
	if !ok {
		return
	}
	cfg.Targets = append(cfg.Targets, target)
}

func appendLinuxKitBucketTargets(cfg *LinuxKitConfig, ext map[string]any) {
	bucket, ok := ext["bucket"].(string)
	if !ok || bucket == "" {
		return
	}

	for _, provider := range linuxKitCloudProvidersForFormats(cfg.Formats) {
		if linuxKitHasCloudTarget(*cfg, provider) {
			continue
		}
		cfg.Targets = append(cfg.Targets, LinuxKitTarget{
			Provider: provider,
			Bucket:   bucket,
		})
	}
}

func parseLinuxKitTargets(value any) []LinuxKitTarget {
	switch typed := value.(type) {
	case nil:
		return nil
	case []LinuxKitTarget:
		return typed
	case []map[string]any:
		targets := make([]LinuxKitTarget, 0, len(typed))
		for _, item := range typed {
			target, ok := linuxKitTargetFromAny("", item)
			if ok {
				targets = append(targets, target)
			}
		}
		return targets
	case []string:
		targets := make([]LinuxKitTarget, 0, len(typed))
		for _, item := range typed {
			target, ok := linuxKitTargetFromAny("", item)
			if ok {
				targets = append(targets, target)
			}
		}
		return targets
	case []any:
		targets := make([]LinuxKitTarget, 0, len(typed))
		for _, item := range typed {
			target, ok := linuxKitTargetFromAny("", item)
			if ok {
				targets = append(targets, target)
			}
		}
		return targets
	default:
		target, ok := linuxKitTargetFromAny("", value)
		if !ok {
			return nil
		}
		return []LinuxKitTarget{target}
	}
}

func linuxKitTargetFromAny(provider string, value any) (LinuxKitTarget, bool) {
	switch typed := value.(type) {
	case nil:
		return LinuxKitTarget{}, false
	case LinuxKitTarget:
		if typed.Provider == "" {
			typed.Provider = provider
		}
		return typed, linuxKitTargetDefined(typed)
	case string:
		return linuxKitTargetFromString(provider, typed)
	default:
		encoded := core.JSONMarshalString(typed)
		var target LinuxKitTarget
		result := core.JSONUnmarshalString(encoded, &target)
		if !result.OK {
			return LinuxKitTarget{}, false
		}
		if target.Provider == "" {
			target.Provider = provider
		}
		return target, linuxKitTargetDefined(target)
	}
}

func linuxKitTargetFromString(provider, raw string) (LinuxKitTarget, bool) {
	raw = core.Trim(raw)
	if raw == "" {
		return LinuxKitTarget{}, false
	}

	if core.HasPrefix(raw, "{") {
		var target LinuxKitTarget
		result := core.JSONUnmarshalString(raw, &target)
		if !result.OK {
			return LinuxKitTarget{}, false
		}
		if target.Provider == "" {
			target.Provider = provider
		}
		return target, linuxKitTargetDefined(target)
	}

	lower := core.Lower(raw)
	switch {
	case core.HasPrefix(lower, "s3://"):
		bucket, prefix := linuxKitSplitBucketPrefix(raw[len("s3://"):])
		return LinuxKitTarget{Provider: "aws", Bucket: bucket, Prefix: prefix}, bucket != ""
	case core.HasPrefix(lower, "gs://"):
		bucket, prefix := linuxKitSplitBucketPrefix(raw[len("gs://"):])
		return LinuxKitTarget{Provider: "gcp", Bucket: bucket, Prefix: prefix}, bucket != ""
	}

	parts := core.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		candidateProvider := normaliseLinuxKitProvider(parts[0])
		if candidateProvider != "" {
			bucket, prefix := linuxKitSplitBucketPrefix(parts[1])
			return LinuxKitTarget{Provider: candidateProvider, Bucket: bucket, Prefix: prefix}, bucket != ""
		}
	}

	bucket, prefix := linuxKitSplitBucketPrefix(raw)
	return LinuxKitTarget{Provider: provider, Bucket: bucket, Prefix: prefix}, bucket != ""
}

func linuxKitTargetDefined(target LinuxKitTarget) bool {
	return target.Name != "" || target.Type != "" || target.Provider != "" || target.Bucket != "" || target.Prefix != "" || target.Region != ""
}

func linuxKitCloudProvidersForFormats(formats []string) []string {
	providers := make([]string, 0, 2)
	for _, format := range formats {
		switch format {
		case "aws":
			providers = appendLinuxKitProvider(providers, "aws")
		case "gcp":
			providers = appendLinuxKitProvider(providers, "gcp")
		}
	}
	return providers
}

func appendLinuxKitProvider(providers []string, provider string) []string {
	for _, existing := range providers {
		if existing == provider {
			return providers
		}
	}
	return append(providers, provider)
}

func linuxKitCloudTargets(cfg LinuxKitConfig, provider string) []LinuxKitTarget {
	targets := make([]LinuxKitTarget, 0, len(cfg.Targets))
	for _, target := range cfg.Targets {
		if linuxKitTargetProvider(target) == provider {
			targets = append(targets, target)
		}
	}
	return targets
}

func linuxKitHasCloudTarget(cfg LinuxKitConfig, provider string) bool {
	return len(linuxKitCloudTargets(cfg, provider)) > 0
}

func linuxKitTargetProvider(target LinuxKitTarget) string {
	for _, value := range []string{target.Provider, target.Type, target.Name} {
		if provider := normaliseLinuxKitProvider(value); provider != "" {
			return provider
		}
	}
	return ""
}

func normaliseLinuxKitProvider(value string) string {
	switch core.Lower(core.Trim(value)) {
	case "aws", "s3":
		return "aws"
	case "gcp", "gcs", "google", "google-cloud":
		return "gcp"
	default:
		return ""
	}
}

func linuxKitSplitBucketPrefix(value string) (string, string) {
	clean := trimSlashes(value)
	if clean == "" {
		return "", ""
	}
	parts := core.SplitN(clean, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], trimSlashes(parts[1])
}

func linuxKitObjectKey(target LinuxKitTarget, artifactPath string) string {
	base := ax.Base(artifactPath)
	prefix := trimSlashes(target.Prefix)
	if prefix == "" {
		return base
	}
	return core.Join("/", prefix, base)
}

func linuxKitCloudURI(scheme, bucket, objectKey string) string {
	return core.Concat(scheme, "://", trimSlashes(bucket), "/", trimSlashes(objectKey))
}

// Package publishers provides release publishing implementations.
package publishers

import (
	"context"
	"embed"
	"text/template"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

//go:embed templates/scoop/*.tmpl
var scoopTemplates embed.FS

// ScoopConfig holds Scoop-specific configuration.
//
// cfg := publishers.ScoopConfig{Bucket: "host-uk/scoop-bucket"}
type ScoopConfig struct {
	// Bucket is the Scoop bucket repository (e.g., "host-uk/scoop-bucket").
	Bucket string
	// Official config for generating files for official repo PRs.
	Official *OfficialConfig
}

// ScoopPublisher publishes releases to Scoop.
//
// pub := publishers.NewScoopPublisher()
type ScoopPublisher struct{}

// NewScoopPublisher creates a new Scoop publisher.
//
// pub := publishers.NewScoopPublisher()
func NewScoopPublisher() *ScoopPublisher {
	return &ScoopPublisher{}
}

// Name returns the publisher's identifier.
//
// name := pub.Name() // → "scoop"
func (p *ScoopPublisher) Name() string {
	return "scoop"
}

// Validate checks the Scoop publisher configuration before publishing.
func (p *ScoopPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) error {
	_ = ctx
	if err := validatePublisherRelease(p.Name(), release); err != nil {
		return err
	}

	cfg := p.parseConfig(pubCfg, relCfg)
	if cfg.Bucket == "" && (cfg.Official == nil || !cfg.Official.Enabled) {
		return coreerr.E("scoop.Validate", "bucket is required (set publish.scoop.bucket in config)", nil)
	}

	return nil
}

// Supports reports whether the publisher handles the requested target.
func (p *ScoopPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish publishes the release to Scoop.
//
// err := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *ScoopPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	if err := validatePublisherRelease(p.Name(), release); err != nil {
		return err
	}

	cfg := p.parseConfig(pubCfg, relCfg)

	if cfg.Bucket == "" && (cfg.Official == nil || !cfg.Official.Enabled) {
		return coreerr.E("scoop.Publish", "bucket is required (set publish.scoop.bucket in config)", nil)
	}

	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepo, err := detectRepository(ctx, release.ProjectDir)
		if err != nil {
			return coreerr.E("scoop.Publish", "could not determine repository", err)
		}
		repo = detectedRepo
	}

	projectName := ""
	if relCfg != nil {
		projectName = relCfg.GetProjectName()
	}
	if projectName == "" {
		parts := core.Split(repo, "/")
		projectName = parts[len(parts)-1]
	}

	version := core.TrimPrefix(release.Version, "v")
	checksums := buildChecksumMapFromRelease(release)

	data := scoopTemplateData{
		PackageName: projectName,
		Description: core.Sprintf("%s CLI", projectName),
		Repository:  repo,
		Version:     version,
		License:     "MIT",
		BinaryName:  projectName,
		Checksums:   checksums,
	}

	if dryRun {
		return p.dryRunPublish(release.FS, data, cfg)
	}

	return p.executePublish(ctx, release.ProjectDir, data, cfg, release)
}

type scoopTemplateData struct {
	PackageName string
	Description string
	Repository  string
	Version     string
	License     string
	BinaryName  string
	Checksums   ChecksumMap
}

func (p *ScoopPublisher) parseConfig(pubCfg PublisherConfig, relCfg ReleaseConfig) ScoopConfig {
	cfg := ScoopConfig{}

	if ext, ok := pubCfg.Extended.(map[string]any); ok {
		if bucket, ok := ext["bucket"].(string); ok && bucket != "" {
			cfg.Bucket = bucket
		}
		if official, ok := ext["official"].(map[string]any); ok {
			cfg.Official = &OfficialConfig{}
			if enabled, ok := official["enabled"].(bool); ok {
				cfg.Official.Enabled = enabled
			}
			if output, ok := official["output"].(string); ok {
				cfg.Official.Output = output
			}
		}
	}

	return cfg
}

func (p *ScoopPublisher) dryRunPublish(m coreio.Medium, data scoopTemplateData, cfg ScoopConfig) error {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: Scoop Publish ===")
	publisherPrintln()
	publisherPrint("Package:    %s", data.PackageName)
	publisherPrint("Version:    %s", data.Version)
	publisherPrint("Bucket:     %s", cfg.Bucket)
	publisherPrint("Repository: %s", data.Repository)
	publisherPrintln()

	manifest, err := p.renderTemplate(m, "templates/scoop/manifest.json.tmpl", data)
	if err != nil {
		return coreerr.E("scoop.dryRunPublish", "failed to render template", err)
	}
	publisherPrintln("Generated manifest.json:")
	publisherPrintln("---")
	publisherPrintln(manifest)
	publisherPrintln("---")
	publisherPrintln()

	if cfg.Bucket != "" && !scoopOfficialMode(cfg) {
		publisherPrint("Would commit to bucket: %s", cfg.Bucket)
	}
	if scoopOfficialMode(cfg) {
		output := cfg.Official.Output
		if output == "" {
			output = "dist/scoop"
		}
		publisherPrint("Would write files for official PR to: %s", output)
	}
	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return nil
}

func (p *ScoopPublisher) executePublish(ctx context.Context, projectDir string, data scoopTemplateData, cfg ScoopConfig, release *Release) error {
	manifest, err := p.renderTemplate(release.FS, "templates/scoop/manifest.json.tmpl", data)
	if err != nil {
		return coreerr.E("scoop.Publish", "failed to render manifest", err)
	}

	// If official config is enabled, write to output directory
	if scoopOfficialMode(cfg) {
		output := cfg.Official.Output
		if output == "" {
			output = ax.Join(projectDir, "dist", "scoop")
		} else if !ax.IsAbs(output) {
			output = ax.Join(projectDir, output)
		}

		if err := release.FS.EnsureDir(output); err != nil {
			return coreerr.E("scoop.Publish", "failed to create output directory", err)
		}

		manifestPath := ax.Join(output, core.Sprintf("%s.json", data.PackageName))
		if err := release.FS.Write(manifestPath, manifest); err != nil {
			return coreerr.E("scoop.Publish", "failed to write manifest", err)
		}
		publisherPrint("Wrote Scoop manifest for official PR: %s", manifestPath)
	}

	// Official repo mode generates PR-ready files and does not publish directly.
	if cfg.Bucket != "" && !scoopOfficialMode(cfg) {
		if err := p.commitToBucket(ctx, cfg.Bucket, data, manifest); err != nil {
			return err
		}
	}

	return nil
}

func scoopOfficialMode(cfg ScoopConfig) bool {
	return cfg.Official != nil && cfg.Official.Enabled
}

func (p *ScoopPublisher) commitToBucket(ctx context.Context, bucket string, data scoopTemplateData, manifest string) error {
	tmpDir, err := ax.TempDir("scoop-bucket-*")
	if err != nil {
		return coreerr.E("scoop.commitToBucket", "failed to create temp directory", err)
	}
	defer func() { _ = ax.RemoveAll(tmpDir) }()

	publisherPrint("Cloning bucket %s...", bucket)
	if err := publisherRun(ctx, "", nil, "gh", "repo", "clone", bucket, tmpDir, "--", "--depth=1"); err != nil {
		return coreerr.E("scoop.commitToBucket", "failed to clone bucket", err)
	}

	// Ensure bucket directory exists
	bucketDir := ax.Join(tmpDir, "bucket")
	if !ax.IsDir(bucketDir) {
		bucketDir = tmpDir // Some repos put manifests in root
	}

	manifestPath := ax.Join(bucketDir, core.Sprintf("%s.json", data.PackageName))
	if err := ax.WriteString(manifestPath, manifest, 0o644); err != nil {
		return coreerr.E("scoop.commitToBucket", "failed to write manifest", err)
	}

	commitMsg := core.Sprintf("Update %s to %s", data.PackageName, data.Version)

	if err := ax.ExecDir(ctx, tmpDir, "git", "add", "."); err != nil {
		return coreerr.E("scoop.commitToBucket", "git add failed", err)
	}

	if err := publisherRun(ctx, tmpDir, nil, "git", "commit", "-m", commitMsg); err != nil {
		return coreerr.E("scoop.commitToBucket", "git commit failed", err)
	}

	if err := publisherRun(ctx, tmpDir, nil, "git", "push"); err != nil {
		return coreerr.E("scoop.commitToBucket", "git push failed", err)
	}

	publisherPrint("Updated Scoop bucket: %s", bucket)
	return nil
}

func (p *ScoopPublisher) renderTemplate(m coreio.Medium, name string, data scoopTemplateData) (string, error) {
	var content []byte
	var err error

	// Try custom template from medium
	customPath := ax.Join(".core", name)
	if m != nil && m.IsFile(customPath) {
		customContent, err := m.Read(customPath)
		if err != nil {
			return "", coreerr.E("scoop.renderTemplate", "failed to read custom template "+customPath, err)
		}
		content = []byte(customContent)
	}

	// Fallback to embedded template
	if content == nil {
		content, err = scoopTemplates.ReadFile(name)
		if err != nil {
			return "", coreerr.E("scoop.renderTemplate", "failed to read template "+name, err)
		}
	}

	tmpl, err := template.New(ax.Base(name)).Funcs(publisherTemplateFuncs()).Parse(string(content))
	if err != nil {
		return "", coreerr.E("scoop.renderTemplate", "failed to parse template "+name, err)
	}

	buf := core.NewBuffer()
	if err := tmpl.Execute(buf, data); err != nil {
		return "", coreerr.E("scoop.renderTemplate", "failed to execute template "+name, err)
	}

	return buf.String(), nil
}

// Package publishers provides release publishing implementations.
package publishers

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"forge.lthn.ai/core/go-build/pkg/build"
	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
)

//go:embed templates/scoop/*.tmpl
var scoopTemplates embed.FS

// ScoopConfig holds Scoop-specific configuration.
type ScoopConfig struct {
	// Bucket is the Scoop bucket repository (e.g., "host-uk/scoop-bucket").
	Bucket string
	// Official config for generating files for official repo PRs.
	Official *OfficialConfig
}

// ScoopPublisher publishes releases to Scoop.
type ScoopPublisher struct{}

// NewScoopPublisher creates a new Scoop publisher.
func NewScoopPublisher() *ScoopPublisher {
	return &ScoopPublisher{}
}

// Name returns the publisher's identifier.
func (p *ScoopPublisher) Name() string {
	return "scoop"
}

// Publish publishes the release to Scoop.
func (p *ScoopPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	cfg := p.parseConfig(pubCfg, relCfg)

	if cfg.Bucket == "" && (cfg.Official == nil || !cfg.Official.Enabled) {
		return coreerr.E("scoop.Publish", "bucket is required (set publish.scoop.bucket in config)", nil)
	}

	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepo, err := detectRepository(release.ProjectDir)
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
		parts := strings.Split(repo, "/")
		projectName = parts[len(parts)-1]
	}

	version := strings.TrimPrefix(release.Version, "v")
	checksums := buildChecksumMap(release.Artifacts)

	data := scoopTemplateData{
		PackageName: projectName,
		Description: fmt.Sprintf("%s CLI", projectName),
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
	fmt.Println()
	fmt.Println("=== DRY RUN: Scoop Publish ===")
	fmt.Println()
	fmt.Printf("Package:    %s\n", data.PackageName)
	fmt.Printf("Version:    %s\n", data.Version)
	fmt.Printf("Bucket:     %s\n", cfg.Bucket)
	fmt.Printf("Repository: %s\n", data.Repository)
	fmt.Println()

	manifest, err := p.renderTemplate(m, "templates/scoop/manifest.json.tmpl", data)
	if err != nil {
		return coreerr.E("scoop.dryRunPublish", "failed to render template", err)
	}
	fmt.Println("Generated manifest.json:")
	fmt.Println("---")
	fmt.Println(manifest)
	fmt.Println("---")
	fmt.Println()

	if cfg.Bucket != "" {
		fmt.Printf("Would commit to bucket: %s\n", cfg.Bucket)
	}
	if cfg.Official != nil && cfg.Official.Enabled {
		output := cfg.Official.Output
		if output == "" {
			output = "dist/scoop"
		}
		fmt.Printf("Would write files for official PR to: %s\n", output)
	}
	fmt.Println()
	fmt.Println("=== END DRY RUN ===")

	return nil
}

func (p *ScoopPublisher) executePublish(ctx context.Context, projectDir string, data scoopTemplateData, cfg ScoopConfig, release *Release) error {
	manifest, err := p.renderTemplate(release.FS, "templates/scoop/manifest.json.tmpl", data)
	if err != nil {
		return coreerr.E("scoop.Publish", "failed to render manifest", err)
	}

	// If official config is enabled, write to output directory
	if cfg.Official != nil && cfg.Official.Enabled {
		output := cfg.Official.Output
		if output == "" {
			output = filepath.Join(projectDir, "dist", "scoop")
		} else if !filepath.IsAbs(output) {
			output = filepath.Join(projectDir, output)
		}

		if err := release.FS.EnsureDir(output); err != nil {
			return coreerr.E("scoop.Publish", "failed to create output directory", err)
		}

		manifestPath := filepath.Join(output, fmt.Sprintf("%s.json", data.PackageName))
		if err := release.FS.Write(manifestPath, manifest); err != nil {
			return coreerr.E("scoop.Publish", "failed to write manifest", err)
		}
		fmt.Printf("Wrote Scoop manifest for official PR: %s\n", manifestPath)
	}

	// If bucket is configured, commit to it
	if cfg.Bucket != "" {
		if err := p.commitToBucket(ctx, cfg.Bucket, data, manifest); err != nil {
			return err
		}
	}

	return nil
}

func (p *ScoopPublisher) commitToBucket(ctx context.Context, bucket string, data scoopTemplateData, manifest string) error {
	tmpDir, err := os.MkdirTemp("", "scoop-bucket-*")
	if err != nil {
		return coreerr.E("scoop.commitToBucket", "failed to create temp directory", err)
	}
	defer func() { _ = coreio.Local.DeleteAll(tmpDir) }()

	fmt.Printf("Cloning bucket %s...\n", bucket)
	cmd := exec.CommandContext(ctx, "gh", "repo", "clone", bucket, tmpDir, "--", "--depth=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return coreerr.E("scoop.commitToBucket", "failed to clone bucket", err)
	}

	// Ensure bucket directory exists
	bucketDir := filepath.Join(tmpDir, "bucket")
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		bucketDir = tmpDir // Some repos put manifests in root
	}

	manifestPath := filepath.Join(bucketDir, fmt.Sprintf("%s.json", data.PackageName))
	if err := coreio.Local.Write(manifestPath, manifest); err != nil {
		return coreerr.E("scoop.commitToBucket", "failed to write manifest", err)
	}

	commitMsg := fmt.Sprintf("Update %s to %s", data.PackageName, data.Version)

	cmd = exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return coreerr.E("scoop.commitToBucket", "git add failed", err)
	}

	cmd = exec.CommandContext(ctx, "git", "commit", "-m", commitMsg)
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return coreerr.E("scoop.commitToBucket", "git commit failed", err)
	}

	cmd = exec.CommandContext(ctx, "git", "push")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return coreerr.E("scoop.commitToBucket", "git push failed", err)
	}

	fmt.Printf("Updated Scoop bucket: %s\n", bucket)
	return nil
}

func (p *ScoopPublisher) renderTemplate(m coreio.Medium, name string, data scoopTemplateData) (string, error) {
	var content []byte
	var err error

	// Try custom template from medium
	customPath := filepath.Join(".core", name)
	if m != nil && m.IsFile(customPath) {
		customContent, err := m.Read(customPath)
		if err == nil {
			content = []byte(customContent)
		}
	}

	// Fallback to embedded template
	if content == nil {
		content, err = scoopTemplates.ReadFile(name)
		if err != nil {
			return "", coreerr.E("scoop.renderTemplate", "failed to read template "+name, err)
		}
	}

	tmpl, err := template.New(filepath.Base(name)).Parse(string(content))
	if err != nil {
		return "", coreerr.E("scoop.renderTemplate", "failed to parse template "+name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", coreerr.E("scoop.renderTemplate", "failed to execute template "+name, err)
	}

	return buf.String(), nil
}

// Ensure build package is used
var _ = build.Artifact{}

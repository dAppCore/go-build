// Package build provides project type detection and cross-compilation for the Core build system.
// This file handles CI environment detection and GitHub Actions output formatting.
package build

import (
	"context"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	io_interface "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// CIContext holds environment information detected from a GitHub Actions run.
//
//	ci := build.DetectCI()
//	if ci != nil {
//	    fmt.Println(ci.ShortSHA) // "abc1234"
//	}
type CIContext struct {
	// Ref is the full git ref (GITHUB_REF).
	//   ci.Ref // "refs/tags/v1.2.3"
	Ref string
	// SHA is the full commit hash (GITHUB_SHA).
	//   ci.SHA // "abc1234def5678..."
	SHA string
	// ShortSHA is the first 7 characters of SHA.
	//   ci.ShortSHA // "abc1234"
	ShortSHA string
	// Tag is the tag name when the ref is a tag ref.
	//   ci.Tag // "v1.2.3"
	Tag string
	// IsTag is true when the ref is a tag ref (refs/tags/...).
	//   ci.IsTag // true
	IsTag bool
	// Branch is the branch name when the ref is a branch ref.
	//   ci.Branch // "main"
	Branch string
	// Repo is the owner/repo string (GITHUB_REPOSITORY).
	//   ci.Repo // "dappcore/core"
	Repo string
	// Owner is the repository owner derived from Repo.
	//   ci.Owner // "dappcore"
	Owner string
}

// artifactMeta is the structure written to artifact_meta.json.
type artifactMeta struct {
	Name   string `json:"name"`
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Ref    string `json:"ref,omitempty"`
	SHA    string `json:"sha,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Branch string `json:"branch,omitempty"`
	IsTag  bool   `json:"is_tag"`
	Repo   string `json:"repo,omitempty"`
}

// FormatGitHubAnnotation formats a build message as a GitHub Actions annotation.
//
//	s := build.FormatGitHubAnnotation("error", "main.go", 42, "undefined: foo")
//	// "::error file=main.go,line=42::undefined: foo"
//
//	s := build.FormatGitHubAnnotation("warning", "pkg/build/ci.go", 10, "unused import")
//	// "::warning file=pkg/build/ci.go,line=10::unused import"
func FormatGitHubAnnotation(level, file string, line int, message string) string {
	return core.Sprintf(
		"::%s file=%s,line=%d::%s",
		escapeGitHubAnnotationValue(level),
		escapeGitHubAnnotationValue(file),
		line,
		escapeGitHubAnnotationValue(message),
	)
}

func escapeGitHubAnnotationValue(value string) string {
	value = core.Replace(value, "%", "%25")
	value = core.Replace(value, "\r", "%0D")
	value = core.Replace(value, "\n", "%0A")
	return value
}

// DetectCI reads GitHub Actions environment variables and returns a populated CIContext.
// Returns nil if GITHUB_ACTIONS is not set or GITHUB_SHA is empty, which indicates
// the process is not running inside GitHub Actions.
//
//	ci := build.DetectCI()
//	if ci == nil {
//	    // running locally, skip CI-specific output
//	}
//	if ci != nil && ci.IsTag {
//	    // upload release assets
//	}
func DetectCI() *CIContext {
	return detectGitHubContext(true)
}

// DetectGitHubMetadata returns GitHub CI metadata when the standard environment
// variables are present, even if GITHUB_ACTIONS is unset.
//
// This is useful for metadata emission paths that only need the GitHub ref/SHA
// shape and should not be coupled to a specific runner environment.
func DetectGitHubMetadata() *CIContext {
	return detectGitHubContext(false)
}

func detectLocalGitMetadata(dir string) *CIContext {
	dir = core.Trim(dir)
	if dir == "" {
		return nil
	}

	sha, err := runGitMetadataCommand(dir, "rev-parse", "HEAD")
	if err != nil || sha == "" {
		return nil
	}

	ctx := &CIContext{SHA: sha}

	if tag, err := runGitMetadataCommand(dir, "describe", "--tags", "--exact-match", "HEAD"); err == nil && tag != "" {
		ctx.Ref = "refs/tags/" + tag
	} else if branch, err := runGitMetadataCommand(dir, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil && branch != "" {
		ctx.Ref = "refs/heads/" + branch
	}

	if remoteURL, err := runGitMetadataCommand(dir, "remote", "get-url", "origin"); err == nil {
		ctx.Repo, ctx.Owner = parseGitRemote(remoteURL)
	}

	populateGitHubContext(ctx)
	return ctx
}

func detectGitHubContext(requireActions bool) *CIContext {
	if requireActions && core.Env("GITHUB_ACTIONS") == "" {
		return nil
	}

	sha := core.Env("GITHUB_SHA")
	if sha == "" {
		return nil
	}

	ref := core.Env("GITHUB_REF")
	repo := core.Env("GITHUB_REPOSITORY")

	ctx := &CIContext{
		Ref:  ref,
		SHA:  sha,
		Repo: repo,
	}

	populateGitHubContext(ctx)
	return ctx
}

func populateGitHubContext(ctx *CIContext) {
	if ctx == nil {
		return
	}

	// ShortSHA is first 7 chars of SHA.
	runes := []rune(ctx.SHA)
	if len(runes) >= 7 {
		ctx.ShortSHA = string(runes[:7])
	} else {
		ctx.ShortSHA = ctx.SHA
	}

	// Derive owner from "owner/repo" format.
	if ctx.Repo != "" {
		parts := core.SplitN(ctx.Repo, "/", 2)
		if len(parts) == 2 {
			ctx.Owner = parts[0]
		}
	}

	// Classify ref as tag or branch.
	const tagPrefix = "refs/tags/"
	const branchPrefix = "refs/heads/"

	if core.HasPrefix(ctx.Ref, tagPrefix) {
		ctx.IsTag = true
		ctx.Tag = core.TrimPrefix(ctx.Ref, tagPrefix)
	} else if core.HasPrefix(ctx.Ref, branchPrefix) {
		ctx.Branch = core.TrimPrefix(ctx.Ref, branchPrefix)
	}
}

func runGitMetadataCommand(dir string, args ...string) (string, error) {
	output, err := ax.RunDir(context.Background(), dir, "git", args...)
	if err != nil {
		return "", err
	}
	return core.Trim(output), nil
}

func parseGitRemote(raw string) (string, string) {
	raw = core.Trim(raw)
	if raw == "" {
		return "", ""
	}

	path := remoteRepositoryPath(raw)
	if path == "" {
		return "", ""
	}

	path = core.Replace(path, "\\", "/")
	parts := core.Split(path, "/")
	if len(parts) < 2 {
		return "", ""
	}

	owner := parts[len(parts)-2]
	repo := core.TrimSuffix(parts[len(parts)-1], ".git")
	if owner == "" || repo == "" {
		return "", ""
	}

	value := owner + "/" + repo
	return value, owner
}

func remoteRepositoryPath(raw string) string {
	if splitURL := core.SplitN(raw, "://", 2); len(splitURL) == 2 && splitURL[0] != "" {
		raw = splitURL[1]
		pathParts := core.SplitN(raw, "/", 2)
		if len(pathParts) != 2 {
			return ""
		}
		return core.Trim(core.SplitN(pathParts[1], "?", 2)[0], "/")
	}

	if splitSCM := core.SplitN(raw, ":", 2); len(splitSCM) == 2 && splitSCM[0] != "" && core.Contains(splitSCM[0], "@") {
		return core.Trim(splitSCM[1], "/")
	}

	return core.Trim(raw, "/")
}

// ArtifactName generates a canonical artifact filename from the build name, CI context, and target.
// Format: {name}_{OS}_{ARCH}_{TAG|SHORT_SHA}
// When ci is nil or has no tag or SHA, only the name and target are used.
//
//	name := build.ArtifactName("core", ci, build.Target{OS: "linux", Arch: "amd64"})
//	// "core_linux_amd64_v1.2.3"  (when ci.IsTag)
//	// "core_linux_amd64_abc1234" (when ci != nil, not a tag)
//	// "core_linux_amd64"         (when ci is nil)
func ArtifactName(buildName string, ci *CIContext, target Target) string {
	base := core.Join("_", buildName, target.OS, target.Arch)

	if ci == nil {
		return base
	}

	var version string
	if ci.IsTag && ci.Tag != "" {
		version = ci.Tag
	} else if ci.ShortSHA != "" {
		version = ci.ShortSHA
	}

	if version == "" {
		return base
	}

	return core.Concat(base, "_", version)
}

// WriteArtifactMeta writes an artifact_meta.json file to path.
// The file contains the build name, target OS/arch, and CI metadata if available.
//
//	err := build.WriteArtifactMeta(io.Local, "dist/artifact_meta.json", "core", build.Target{OS: "linux", Arch: "amd64"}, ci)
//	// writes: {"name":"core","os":"linux","arch":"amd64","tag":"v1.2.3","is_tag":true,...}
func WriteArtifactMeta(fs io_interface.Medium, path string, buildName string, target Target, ci *CIContext) error {
	meta := artifactMeta{
		Name: buildName,
		OS:   target.OS,
		Arch: target.Arch,
	}

	if ci != nil {
		meta.Ref = ci.Ref
		meta.SHA = ci.SHA
		meta.Tag = ci.Tag
		meta.Branch = ci.Branch
		meta.IsTag = ci.IsTag
		meta.Repo = ci.Repo
	}

	encodedData := core.JSONMarshal(meta)
	if !encodedData.OK {
		return coreerr.E("build.WriteArtifactMeta", "failed to marshal artifact meta", encodedData.Error())
	}

	if err := fs.Write(path, string(encodedData.Value.([]byte))); err != nil {
		return coreerr.E("build.WriteArtifactMeta", "failed to write artifact meta", err)
	}

	return nil
}

// CIArtifactPath returns the CI-stamped artifact path for a build output.
// The filename keeps the original packaging suffix, such as `.tar.gz`, `.zip`,
// `.exe`, or `.app`.
//
//	path := build.CIArtifactPath("core", ci, build.Artifact{
//	    Path: "/tmp/dist/linux_amd64/core.tar.gz",
//	    OS: "linux",
//	    Arch: "amd64",
//	})
func CIArtifactPath(buildName string, ci *CIContext, artifact Artifact) string {
	if ci == nil || artifact.Path == "" || artifact.OS == "" || artifact.Arch == "" {
		return artifact.Path
	}

	return replaceArtifactBaseName(artifact.Path, ArtifactName(buildName, ci, Target{
		OS:   artifact.OS,
		Arch: artifact.Arch,
	}))
}

func replaceArtifactBaseName(artifactPath, replacement string) string {
	if artifactPath == "" || replacement == "" {
		return artifactPath
	}

	baseName := ax.Base(artifactPath)
	suffix := artifactPathSuffix(baseName)
	if suffix == "" {
		return ax.Join(ax.Dir(artifactPath), replacement)
	}

	return ax.Join(ax.Dir(artifactPath), replacement+suffix)
}

func artifactPathSuffix(fileName string) string {
	switch {
	case core.HasSuffix(fileName, ".tar.gz"):
		return ".tar.gz"
	case core.HasSuffix(fileName, ".tar.xz"):
		return ".tar.xz"
	case core.HasSuffix(fileName, ".tar.zst"):
		return ".tar.zst"
	case core.HasSuffix(fileName, ".tar.bz2"):
		return ".tar.bz2"
	case core.HasSuffix(fileName, ".tgz"):
		return ".tgz"
	case core.HasSuffix(fileName, ".txz"):
		return ".txz"
	case core.HasSuffix(fileName, ".zip"):
		return ".zip"
	case core.HasSuffix(fileName, ".exe"):
		return ".exe"
	case core.HasSuffix(fileName, ".dmg"):
		return ".dmg"
	case core.HasSuffix(fileName, ".app"):
		return ".app"
	default:
		parts := core.Split(fileName, ".")
		if len(parts) <= 1 || (len(parts) == 2 && parts[0] == "") {
			return ""
		}

		return "." + parts[len(parts)-1]
	}
}

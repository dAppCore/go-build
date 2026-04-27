package publishers

import (
	"strings"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
)

// ChecksumMap holds resolved release asset names and SHA-256 checksums for the
// platform archives publishers reference in generated package metadata.
type ChecksumMap struct {
	DarwinAmd64      string
	DarwinArm64      string
	LinuxAmd64       string
	LinuxArm64       string
	WindowsAmd64     string
	WindowsArm64     string
	DarwinAmd64File  string
	DarwinArm64File  string
	LinuxAmd64File   string
	LinuxArm64File   string
	WindowsAmd64File string
	WindowsArm64File string
	ChecksumFile     string
}

func buildChecksumMap(artifacts []build.Artifact) ChecksumMap {
	return populateChecksumMap(artifacts, nil)
}

func buildChecksumMapFromRelease(release *Release) ChecksumMap {
	if release == nil {
		return ChecksumMap{}
	}

	checksums := buildChecksumMap(release.Artifacts)
	if checksums.ChecksumFile == "" {
		return checksums
	}

	filesystem := releaseArtifactFS(release)
	if filesystem == nil {
		return checksums
	}

	checksumPath := ""
	for _, artifact := range release.Artifacts {
		if ax.Base(artifact.Path) == checksums.ChecksumFile {
			checksumPath = artifact.Path
			break
		}
	}
	if checksumPath == "" {
		return checksums
	}

	content, err := filesystem.Read(checksumPath)
	if err != nil {
		return checksums
	}

	lookup := parseChecksumFile(content)
	if len(lookup) == 0 {
		return checksums
	}

	return populateChecksumMap(release.Artifacts, lookup)
}

func populateChecksumMap(artifacts []build.Artifact, lookup map[string]string) ChecksumMap {
	checksums := ChecksumMap{}

	for _, artifact := range artifacts {
		name := ax.Base(artifact.Path)
		if name == "" {
			continue
		}

		if isChecksumArtifactName(name) {
			if checksums.ChecksumFile == "" {
				checksums.ChecksumFile = name
			}
			continue
		}
		if isSignatureArtifactName(name) || isMetadataArtifactName(name) {
			continue
		}

		osValue, archValue, ok := artifactPlatform(artifact, name)
		if !ok {
			continue
		}

		checksum := artifact.Checksum
		if checksum == "" && lookup != nil {
			checksum = lookup[name]
		}

		switch osValue + "/" + archValue {
		case "darwin/amd64":
			assignChecksumEntry(&checksums.DarwinAmd64File, &checksums.DarwinAmd64, name, checksum)
		case "darwin/arm64":
			assignChecksumEntry(&checksums.DarwinArm64File, &checksums.DarwinArm64, name, checksum)
		case "linux/amd64":
			assignChecksumEntry(&checksums.LinuxAmd64File, &checksums.LinuxAmd64, name, checksum)
		case "linux/arm64":
			assignChecksumEntry(&checksums.LinuxArm64File, &checksums.LinuxArm64, name, checksum)
		case "windows/amd64":
			assignChecksumEntry(&checksums.WindowsAmd64File, &checksums.WindowsAmd64, name, checksum)
		case "windows/arm64":
			assignChecksumEntry(&checksums.WindowsArm64File, &checksums.WindowsArm64, name, checksum)
		}
	}

	return checksums
}

func assignChecksumEntry(fileName, checksum *string, name, value string) {
	if *fileName == "" {
		*fileName = name
	}
	if *checksum == "" && value != "" {
		*checksum = value
	}
}

func artifactPlatform(artifact build.Artifact, name string) (string, string, bool) {
	if osValue, archValue := normalizePlatformToken(artifact.OS), normalizeArchToken(artifact.Arch); osValue != "" && archValue != "" {
		return osValue, archValue, true
	}

	tokens := strings.FieldsFunc(stripArtifactSuffixes(strings.ToLower(name)), func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	for i := 0; i+1 < len(tokens); i++ {
		osValue := normalizePlatformToken(tokens[i])
		archValue := normalizeArchToken(tokens[i+1])
		if osValue != "" && archValue != "" {
			return osValue, archValue, true
		}
	}

	return "", "", false
}

func stripArtifactSuffixes(name string) string {
	for _, suffix := range []string{
		".tar.gz",
		".tar.xz",
		".zip",
		".app",
		".exe",
		".sig",
		".asc",
	} {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

func normalizePlatformToken(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "darwin", "macos", "mac":
		return "darwin"
	case "linux":
		return "linux"
	case "windows", "win", "win32":
		return "windows"
	default:
		return ""
	}
}

func normalizeArchToken(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "amd64", "x86_64", "x64":
		return "amd64"
	case "arm64", "aarch64":
		return "arm64"
	default:
		return ""
	}
}

func isChecksumArtifactName(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	return strings.HasSuffix(name, ".txt") && strings.Contains(name, "checksum")
}

func isSignatureArtifactName(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	return strings.HasSuffix(name, ".asc") || strings.HasSuffix(name, ".sig")
}

func isMetadataArtifactName(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "artifact_meta.json")
}

func parseChecksumFile(content string) map[string]string {
	lines := strings.Split(content, "\n")
	lookup := make(map[string]string, len(lines))
	for _, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		lookup[ax.Base(fields[len(fields)-1])] = fields[0]
	}
	return lookup
}

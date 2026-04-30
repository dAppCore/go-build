package publishers

import (
	core "dappco.re/go"
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

	contentResult := filesystem.Read(checksumPath)
	if !contentResult.OK {
		return checksums
	}
	content := contentResult.Value.(string)

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

	tokens := splitArtifactTokens(stripArtifactSuffixes(core.Lower(name)))
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
		if core.HasSuffix(name, suffix) {
			return core.TrimSuffix(name, suffix)
		}
	}
	return name
}

func normalizePlatformToken(value string) string {
	switch core.Trim(core.Lower(value)) {
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
	switch core.Trim(core.Lower(value)) {
	case "amd64", "x86_64", "x64":
		return "amd64"
	case "arm64", "aarch64":
		return "arm64"
	default:
		return ""
	}
}

func isChecksumArtifactName(name string) bool {
	name = core.Trim(core.Lower(name))
	return core.HasSuffix(name, ".txt") && core.Contains(name, "checksum")
}

func isSignatureArtifactName(name string) bool {
	name = core.Trim(core.Lower(name))
	return core.HasSuffix(name, ".asc") || core.HasSuffix(name, ".sig")
}

func isMetadataArtifactName(name string) bool {
	return core.Lower(core.Trim(name)) == "artifact_meta.json"
}

func parseChecksumFile(content string) map[string]string {
	lines := core.Split(content, "\n")
	lookup := make(map[string]string, len(lines))
	for _, line := range lines {
		fields := checksumFields(core.Trim(line))
		if len(fields) < 2 {
			continue
		}
		lookup[ax.Base(fields[len(fields)-1])] = fields[0]
	}
	return lookup
}

func splitArtifactTokens(value string) []string {
	return splitDelimitedFields(value, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
}

func checksumFields(value string) []string {
	return splitDelimitedFields(value, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
}

func splitDelimitedFields(value string, separator func(rune) bool) []string {
	var fields []string
	start := -1
	for i, r := range value {
		if separator(r) {
			if start >= 0 {
				fields = append(fields, value[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		fields = append(fields, value[start:])
	}
	return fields
}

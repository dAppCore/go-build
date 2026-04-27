package build

import "dappco.re/go/core"

// ExpandVersionTemplate resolves the RFC-documented version placeholders used
// across build and release config surfaces.
//
// Supported placeholders:
//   - {{.Tag}} / {{Tag}} → v-prefixed version/tag
//   - {{.Version}} / {{Version}} → legacy full version value
//
// The helper also understands v{{.Version}} / v{{Version}} so RFC examples
// that prefix the placeholder do not render a duplicated "v".
func ExpandVersionTemplate(value, version string) string {
	if value == "" || version == "" {
		return value
	}

	trimmedVersion := core.TrimPrefix(version, "v")

	value = core.Replace(value, "v{{.Version}}", "v"+trimmedVersion)
	value = core.Replace(value, "v{{Version}}", "v"+trimmedVersion)
	value = core.Replace(value, "{{.Tag}}", version)
	value = core.Replace(value, "{{Tag}}", version)
	value = core.Replace(value, "{{.Version}}", version)
	value = core.Replace(value, "{{Version}}", version)

	return value
}

// ExpandVersionTemplates resolves version placeholders across a string slice.
func ExpandVersionTemplates(values []string, version string) []string {
	if len(values) == 0 || version == "" {
		return values
	}

	expanded := make([]string, 0, len(values))
	for _, value := range values {
		expanded = append(expanded, ExpandVersionTemplate(value, version))
	}

	return expanded
}

// ExpandVersionTemplateMap resolves version placeholders across a string map.
func ExpandVersionTemplateMap(values map[string]string, version string) map[string]string {
	if len(values) == 0 || version == "" {
		return CloneStringMap(values)
	}

	expanded := make(map[string]string, len(values))
	for key, value := range values {
		expanded[key] = ExpandVersionTemplate(value, version)
	}

	return expanded
}

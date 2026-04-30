// Package locales embeds translation files for this module.
package locales

import "embed"

// Usage example: use locales.FS from package consumers as needed.
//
//go:embed *.json
var FS embed.FS

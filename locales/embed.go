// Package locales embeds translation files for this module.
package locales

import (
	"embed"

	"dappco.re/go/i18n"
)

// Usage example: use locales.FS from package consumers as needed.
//
//go:embed *.json
var FS embed.FS

func init() {
	i18n.RegisterLocales(FS, ".")
}

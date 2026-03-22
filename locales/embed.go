// Package locales embeds translation files for this module.
package locales

import (
	"embed"

	"dappco.re/go/core/i18n"
)

//go:embed *.json
var FS embed.FS

func init() {
	i18n.RegisterLocales(FS, ".")
}

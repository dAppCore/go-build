// Package locales embeds translation files for this module.
package locales

import (
	"embed"

	"forge.lthn.ai/core/go-i18n"
)

//go:embed *.json
var FS embed.FS

func init() {
	i18n.RegisterLocales(FS, ".")
}

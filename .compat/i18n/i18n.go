package i18n

import (
	"io/fs"

	core "dappco.re/go"
)

func RegisterLocales(fs.FS, string) {}

func T(key string, args ...any) string {
	if len(args) == 0 {
		return key
	}
	switch key {
	case "common.error.failed":
		return core.Sprintf("failed to %v", mapValue(args[0], "Action"))
	case "i18n.fail.get":
		return core.Sprintf("failed to get %v", first(args))
	case "i18n.fail.create":
		return core.Sprintf("failed to create %v", first(args))
	case "i18n.fail.generate":
		return core.Sprintf("failed to generate %v", first(args))
	default:
		return core.Sprintf("%s %v", key, first(args))
	}
}

func Label(word string) string {
	if word == "" {
		return ""
	}
	return word + ":"
}

func Title(text string) string {
	if text == "" {
		return ""
	}
	return core.Upper(text[:1]) + text[1:]
}

func ProgressSubject(verb, subject string) string {
	if subject == "" {
		return verb + "..."
	}
	return core.Sprintf("%s %s...", Title(verb), subject)
}

func first(args []any) any {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func mapValue(value any, key string) any {
	if m, ok := value.(map[string]any); ok {
		return m[key]
	}
	return value
}

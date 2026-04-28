package i18n

import (
	"fmt"
	"io/fs"
	"strings"
)

func RegisterLocales(fs.FS, string) {}

func T(key string, args ...any) string {
	if len(args) == 0 {
		return key
	}
	switch key {
	case "common.error.failed":
		return fmt.Sprintf("failed to %v", mapValue(args[0], "Action"))
	case "i18n.fail.get":
		return fmt.Sprintf("failed to get %v", first(args))
	case "i18n.fail.create":
		return fmt.Sprintf("failed to create %v", first(args))
	case "i18n.fail.generate":
		return fmt.Sprintf("failed to generate %v", first(args))
	default:
		return fmt.Sprintf("%s %v", key, first(args))
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
	return strings.ToUpper(text[:1]) + text[1:]
}

func ProgressSubject(verb, subject string) string {
	if subject == "" {
		return verb + "..."
	}
	return fmt.Sprintf("%s %s...", Title(verb), subject)
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

package publishers

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"text/template"
)

func publisherTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"jsonString": jsonString,
		"jsComment":  jsComment,
		"xmlEscape":  xmlEscape,
		"rubyQuote":  rubyQuote,
		"psQuote":    psQuote,
		"shellQuote": shellQuote,
		"printf":     fmt.Sprintf,
	}
}

func jsonString(value string) string {
	quoted, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(quoted)
}

func jsComment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "*/", "*\\/")
	return value
}

func xmlEscape(value string) string {
	return html.EscapeString(value)
}

func rubyQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	return "'" + value + "'"
}

func psQuote(value string) string {
	value = strings.ReplaceAll(value, `'`, `''`)
	return "'" + value + "'"
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

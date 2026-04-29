package publishers

import (
	"html"
	"text/template"

	core "dappco.re/go"
)

func publisherTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"jsonString": jsonString,
		"jsComment":  jsComment,
		"xmlEscape":  xmlEscape,
		"rubyQuote":  rubyQuote,
		"psQuote":    psQuote,
		"shellQuote": shellQuote,
		"printf":     core.Sprintf,
	}
}

func jsonString(value string) string {
	quoted := core.JSONMarshal(value)
	if !quoted.OK {
		return `""`
	}
	return string(quoted.Value.([]byte))
}

func jsComment(value string) string {
	value = core.Trim(value)
	value = core.Replace(value, "\r", " ")
	value = core.Replace(value, "\n", " ")
	value = core.Replace(value, "*/", "*\\/")
	return value
}

func xmlEscape(value string) string {
	return html.EscapeString(value)
}

func rubyQuote(value string) string {
	value = core.Replace(value, `\`, `\\`)
	value = core.Replace(value, `'`, `\'`)
	return "'" + value + "'"
}

func psQuote(value string) string {
	value = core.Replace(value, `'`, `''`)
	return "'" + value + "'"
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + core.Replace(value, "'", `'"'"'`) + "'"
}

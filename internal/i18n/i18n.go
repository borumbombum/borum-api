package i18n

import "strings"

const DefaultLang = "en"

var Supported = []string{"en", "pt"}

func IsValid(lang string) bool {
	for _, l := range Supported {
		if l == lang {
			return true
		}
	}
	return false
}

// URLFor returns the full URL path for a given language and path.
// For English (default), it returns the path as-is.
// For other languages, it prepends the language prefix.
func URLFor(lang, path string) string {
	if lang == DefaultLang {
		return path
	}
	return "/" + lang + path
}

// LangFromPath extracts the language code from a URL path.
// Returns "en" if no language prefix is found.
func LangFromPath(path string) string {
	for _, lang := range Supported {
		if lang == DefaultLang {
			continue
		}
		prefix := "/" + lang + "/"
		if strings.HasPrefix(path, prefix) || path == "/"+lang {
			return lang
		}
	}
	return DefaultLang
}

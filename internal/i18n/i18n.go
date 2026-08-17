package i18n

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

func Prefix(lang string) string {
	if lang == DefaultLang {
		return ""
	}
	return "/" + lang
}

func OtherLang(lang string) string {
	if lang == DefaultLang {
		return "pt"
	}
	return DefaultLang
}

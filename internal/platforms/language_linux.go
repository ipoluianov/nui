package platforms

import (
	"os"
	"strings"
)

// systemLanguage follows the gettext order: LANGUAGE (a list), LC_ALL,
// LC_MESSAGES, LANG. "C" and "POSIX" mean no language.
func systemLanguage() string {
	if list := os.Getenv("LANGUAGE"); list != "" {
		if lang, _, _ := strings.Cut(list, ":"); lang != "" {
			return lang
		}
	}
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(name); value != "" && value != "C" && value != "POSIX" && !strings.HasPrefix(value, "C.") {
			return value
		}
	}
	return ""
}

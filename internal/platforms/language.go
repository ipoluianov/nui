package platforms

// SystemLanguage returns the language of the user interface of the system as
// the system reports it, e.g. "ru-RU", "zh-Hans-CN" or "en_US.UTF-8"; "" if
// unknown.
func SystemLanguage() string {
	return systemLanguage()
}

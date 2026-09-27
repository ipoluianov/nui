package platforms

import "github.com/ebitengine/purego/objc"

// systemLanguage returns the first of the user's preferred languages, e.g.
// "ru-RU" or "zh-Hans-CN".
func systemLanguage() string {
	var lang string
	withAutoreleasePool(func() {
		languages := objc.ID(objc.GetClass("NSLocale")).Send(objc.RegisterName("preferredLanguages"))
		if languages == 0 || objc.Send[int](languages, selCount) == 0 {
			return
		}
		lang = nsStringToGo(objc.Send[objc.ID](languages, selObjectAtIndex, 0))
	})
	return lang
}

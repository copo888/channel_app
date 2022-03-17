package locales

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// init
func init() {
	initEn(language.Make("en"))
}

// initEn will init en support.
func initEn(tag language.Tag) {
	message.SetString(tag, "0", "Success")
}

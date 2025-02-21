package util

import (
	"unicode"
	"unicode/utf8"
)

func CapitalizeFirstLetter(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

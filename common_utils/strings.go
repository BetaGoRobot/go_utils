package commonutils

import "strings"

// RemoveFromStringRune to be filled
//
//	@param s string
//	@param charsToRemove ...rune
//	@return string
//	@update 2025-04-11 11:11:22
func RemoveFromStringRune(s string, charsToRemove ...rune) string {
	for _, r := range charsToRemove {
		s = strings.ReplaceAll(s, string(r), "")
	}
	return s
}

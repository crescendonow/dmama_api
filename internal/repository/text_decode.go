package repository

import (
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// decodeDBText normalizes legacy Windows-874 text to UTF-8.
// It only decodes strings that are not valid UTF-8 to avoid corrupting clean data.
func decodeDBText(s string) string {
	if s == "" || utf8.ValidString(s) {
		return s
	}

	decoded, err := charmap.Windows874.NewDecoder().String(s)
	if err != nil {
		return s
	}
	return decoded
}

func decodeDBTextPtr(s *string) *string {
	if s == nil {
		return nil
	}

	decoded := decodeDBText(*s)
	return &decoded
}

package httpapi

import "strings"

func firstIP(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})
	for _, part := range parts {
		if ip := strings.TrimSpace(part); ip != "" {
			return ip
		}
	}
	return ""
}

func pickNodeEntryIP(raw string, fallback string) string {
	if ip := firstIP(raw); ip != "" {
		return ip
	}
	return strings.TrimSpace(fallback)
}

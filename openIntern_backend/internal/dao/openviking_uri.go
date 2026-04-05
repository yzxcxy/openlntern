package dao

import "strings"

// normalizeVikingURI 清理 OpenViking URI 的 query/fragment，并去掉末尾斜杠。
func normalizeVikingURI(uri string) string {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return ""
	}
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimRight(strings.TrimSpace(trimmed), "/")
}

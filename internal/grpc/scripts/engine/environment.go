package engine

import "strings"

// MergeEnvironment 使用 overrides 覆盖 base 中的同名环境变量。
func MergeEnvironment(base, overrides []string) []string {
	keys := make(map[string]struct{}, len(overrides))
	for _, item := range overrides {
		key, _, _ := strings.Cut(item, "=")
		keys[key] = struct{}{}
	}
	result := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, exists := keys[key]; !exists {
			result = append(result, item)
		}
	}
	return append(result, overrides...)
}

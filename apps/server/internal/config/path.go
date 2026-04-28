package config

import "os"

// ResolvePath 返回第一个存在的候选路径；如果都不存在，则回退到第一个候选。
func ResolvePath(candidates ...string) string {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if len(candidates) == 0 {
		return ""
	}

	return candidates[0]
}

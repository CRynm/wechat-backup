package regex

import "regexp"

// GetTarget 使用正则表达式提取信息
func GetTarget(pattern, content string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

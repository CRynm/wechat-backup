package html

import "strings"

// UnescapeHTML HTML 解转义函数
func UnescapeHTML(str string) string {
	htmlEscapes := map[string]string{
		"&lt;":   "<",
		"&gt;":   ">",
		"&nbsp;": " ",
		"&amp;":  "&",
		"&quot;": "\"",
	}

	result := str
	for escaped, unescaped := range htmlEscapes {
		result = strings.ReplaceAll(result, escaped, unescaped)
	}
	return result
}

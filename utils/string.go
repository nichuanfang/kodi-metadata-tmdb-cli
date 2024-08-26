package utils

import (
	"runtime"
	"strings"
)

// EndsWith 字符以xx结尾
func EndsWith(str, subStr string) bool {
	index := strings.LastIndex(str, subStr)

	return index > 0 && str[index:] == subStr
}

// sanitizeFileName 替换文件名中的非法字符
func SanitizeFileName(fileName string) string {
	// 定义 Windows 和 Linux 不允许的字符
	var invalidChars string
	if runtime.GOOS == "windows" {
		invalidChars = `\/:*?"<>|`
	} else {
		invalidChars = `/\0`
	}

	// 替换非法字符
	for _, char := range invalidChars {
		fileName = strings.ReplaceAll(fileName, string(char), "_")
	}

	return fileName
}

package utils

import (
	"encoding/json"
	"strings"
)

// MaxFileTagCount 限制单个文件可保存的标签数量，避免元数据无限增长。
const MaxFileTagCount = 12

// MaxFileTagLength 限制单个标签的 Unicode 字符数量。
const MaxFileTagLength = 24

// NormalizeFileTags 清理标签列表、去重并应用数量与长度边界。
func NormalizeFileTags(tags []string) []string {
	// normalizedTags 保存按输入顺序保留的有效标签。
	normalizedTags := make([]string, 0, len(tags))
	// seenTags 保存已出现标签的小写形式，用于不区分大小写去重。
	seenTags := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		// normalizedTag 保存去除空白和井号前缀后的标签。
		normalizedTag := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(tag), "#"))
		if normalizedTag == "" {
			continue
		}
		// tagRunes 保存 Unicode 字符，避免按字节截断中文标签。
		tagRunes := []rune(normalizedTag)
		if len(tagRunes) > MaxFileTagLength {
			normalizedTag = string(tagRunes[:MaxFileTagLength])
		}
		// comparisonKey 保存不区分大小写的去重键。
		comparisonKey := strings.ToLower(normalizedTag)
		if _, exists := seenTags[comparisonKey]; exists {
			continue
		}
		seenTags[comparisonKey] = struct{}{}
		normalizedTags = append(normalizedTags, normalizedTag)
		if len(normalizedTags) >= MaxFileTagCount {
			break
		}
	}
	return normalizedTags
}

// ParseFileTags 将逗号、中文逗号或分号分隔的输入转换为规范标签列表。
func ParseFileTags(input string) []string {
	// rawTags 按支持的分隔符切分管理端输入。
	rawTags := strings.FieldsFunc(input, func(character rune) bool {
		return character == ',' || character == '，' || character == ';' || character == '；'
	})
	return NormalizeFileTags(rawTags)
}

// EncodeFileTags 将标签列表编码为稳定 JSON 文本供 SQLite 保存。
func EncodeFileTags(tags []string) string {
	// normalizedTags 保存写入数据库前的规范标签。
	normalizedTags := NormalizeFileTags(tags)
	if len(normalizedTags) == 0 {
		return "[]"
	}
	// encodedTags、encodeErr 保存 JSON 编码结果与错误。
	encodedTags, encodeErr := json.Marshal(normalizedTags)
	if encodeErr != nil {
		return "[]"
	}
	return string(encodedTags)
}

// DecodeFileTags 将 SQLite 中的 JSON 标签文本还原为规范标签列表。
func DecodeFileTags(input string) []string {
	// tags 保存 JSON 解码后的标签列表。
	var tags []string
	if json.Unmarshal([]byte(strings.TrimSpace(input)), &tags) != nil {
		return ParseFileTags(input)
	}
	return NormalizeFileTags(tags)
}

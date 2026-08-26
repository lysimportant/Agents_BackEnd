package content

import (
	"strings"
	"testing"
)

// TestSanitizeArticleContent 保证日常复用的正文白名单会移除脚本并保留常用富文本标签。
func TestSanitizeArticleContent(t *testing.T) {
	sanitizedHTML := SanitizeArticleContent(`<p><strong>hello</strong></p><script>alert(1)</script><ul><li>world</li></ul>`)
	if strings.Contains(strings.ToLower(sanitizedHTML), "script") {
		t.Fatalf("sanitized HTML still contains script: %q", sanitizedHTML)
	}
	for _, expectedTag := range []string{"<p>", "<strong>hello</strong>", "<ul>", "<li>world</li>"} {
		if !strings.Contains(sanitizedHTML, expectedTag) {
			t.Fatalf("sanitized HTML missing %q: %q", expectedTag, sanitizedHTML)
		}
	}
}

// TestExtractPlainText 验证长度校验使用用户可见文字，而不是 HTML 标签字符数。
func TestExtractPlainText(t *testing.T) {
	if got := ExtractPlainText(`<p><strong>你好</strong>，世界</p>`); got != "你好，世界" {
		t.Fatalf("unexpected plain text: %q", got)
	}
}

// TestSanitizeDailyContent 验证日常只保留富文本格式，不允许嵌入媒体节点。
func TestSanitizeDailyContent(t *testing.T) {
	sanitizedHTML := SanitizeDailyContent(`<p><em>记录</em></p><img src="/api/public/files/12/preview" alt="x"><video src="https://example.com/a.mp4"></video>`)
	if strings.Contains(sanitizedHTML, "https://example.com/a.mp4") {
		t.Fatalf("daily HTML should remove external media: %q", sanitizedHTML)
	}
	if !strings.Contains(sanitizedHTML, `<img alt="x" src="/api/public/files/12/preview">`) {
		t.Fatalf("daily HTML should preserve controlled media: %q", sanitizedHTML)
	}
	if !strings.Contains(sanitizedHTML, "<em>记录</em>") {
		t.Fatalf("daily HTML should preserve formatting: %q", sanitizedHTML)
	}
}

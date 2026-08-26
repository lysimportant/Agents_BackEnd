// Package content 提供文章正文清洗与本地媒体地址重写能力，供 C 端公开接口使用。
package content

import (
	"strings"

	"golang.org/x/net/html"
)

// ExtractPlainText 返回 HTML 中用户可见的纯文本，用于日常内容长度和空值校验。
func ExtractPlainText(input string) string {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return ""
	}
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return strings.TrimSpace(builder.String())
}

// allowedTags 记录正文清洗后允许保留的 HTML 标签白名单。
var allowedTags = map[string]bool{
	"a": true, "b": true, "strong": true, "i": true, "em": true, "u": true, "s": true,
	"p": true, "br": true, "hr": true, "blockquote": true, "pre": true, "code": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "li": true, "div": true, "span": true, "figure": true,
	"figcaption": true, "img": true, "video": true, "source": true, "table": true,
	"thead": true, "tbody": true, "tr": true, "th": true, "td": true,
}

// dailyAllowedTags 记录日常富文本可保存的格式标签，不允许嵌入图片、视频等外部媒体。
var dailyAllowedTags = map[string]bool{
	"a": true, "b": true, "strong": true, "i": true, "em": true, "u": true, "s": true,
	"p": true, "br": true, "blockquote": true, "pre": true, "code": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "li": true, "div": true, "span": true,
}

// allowedProtocols 记录允许保留的链接协议白名单，防止危险协议注入。
var allowedProtocols = map[string]bool{
	"http:": true, "https:": true, "mailto:": true, "#": true,
}

// SanitizeArticleContent 清洗文章正文，仅保留白名单标签与安全属性，返回清理后的 HTML。
func SanitizeArticleContent(input string) string {
	return sanitizeHTML(input, allowedTags)
}

// SanitizeDailyContent 清洗日常富文本，仅保留文字格式和安全链接，禁止嵌入外部媒体。
func SanitizeDailyContent(input string) string {
	return sanitizeHTML(input, dailyAllowedTags)
}

// sanitizeHTML 按传入的标签白名单清洗 HTML，并保留各业务允许的安全属性。
func sanitizeHTML(input string, allowedTagSet map[string]bool) string {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return ""
	}
	var builder strings.Builder
	var writeNode func(*html.Node)
	writeNode = func(node *html.Node) {
		switch node.Type {
		case html.TextNode:
			builder.WriteString(node.Data)
		case html.ElementNode:
			if !allowedTagSet[strings.ToLower(node.Data)] {
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					writeNode(child)
				}
				return
			}
			builder.WriteString("<")
			builder.WriteString(node.Data)
			var href string
			var srcAttr string
			for _, attr := range node.Attr {
				name := strings.ToLower(attr.Key)
				value := attr.Val
				switch {
				case node.Data == "a" && name == "href" && isSafeURL(value):
					href = value
				case (node.Data == "img" || node.Data == "video" || node.Data == "source") && name == "src" && isSafeURL(value):
					srcAttr = value
				case node.Data == "img" && (name == "alt" || name == "title" || name == "width" || name == "height"):
					builder.WriteString(" " + attr.Key + "\"" + html.EscapeString(value) + "\"")
				case node.Data == "video" && (name == "controls" || name == "preload"):
					builder.WriteString(" " + attr.Key + "\"" + html.EscapeString(value) + "\"")
				}
			}
			if href != "" {
				builder.WriteString(" href=\"" + html.EscapeString(href) + "\"")
			}
			if srcAttr != "" {
				builder.WriteString(" src=\"" + html.EscapeString(srcAttr) + "\"")
			}
			builder.WriteString(">")
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				writeNode(child)
			}
			builder.WriteString("</")
			builder.WriteString(node.Data)
			builder.WriteString(">")
		}
	}
	for child := doc.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "html" {
			for body := child.FirstChild; body != nil; body = body.NextSibling {
				if body.Type == html.ElementNode && body.Data == "body" {
					for item := body.FirstChild; item != nil; item = item.NextSibling {
						writeNode(item)
					}
				}
			}
		}
	}
	return builder.String()
}

// isSafeURL 判断 URL 是否安全，拦截 javascript 等危险协议与协议相对地址。
func isSafeURL(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(trimmed, "/") {
		return !strings.HasPrefix(lower, "//")
	}
	if colon := strings.Index(trimmed, ":"); colon > 0 {
		return allowedProtocols[strings.ToLower(trimmed[:colon+1])]
	}
	return true
}

// RewriteLocalMedia 将正文中的本地 /api/files/:id/preview 与 /thumbnail 地址重写为公开地址。
func RewriteLocalMedia(input string, resolve func(id string) string) string {
	replaced := rewritePattern(input, "/api/files/", "/preview", resolve)
	replaced = rewritePattern(replaced, "/api/files/", "/thumbnail", resolve)
	return replaced
}

// rewritePattern 查找并替换指定前后缀之间的媒体 id 为公开地址，无法解析时移除该引用。
func rewritePattern(input, prefix, suffix string, resolve func(id string) string) string {
	result := input
	search := 0
	for {
		start := strings.Index(result[search:], prefix)
		if start < 0 {
			break
		}
		start += search
		idEndRel := strings.Index(result[start+len(prefix):], suffix)
		if idEndRel < 0 {
			break
		}
		id := result[start+len(prefix) : start+len(prefix)+idEndRel]
		resolved := resolve(id)
		if resolved == "" {
			result = result[:start] + result[start+len(prefix)+idEndRel+len(suffix):]
			search = start
			continue
		}
		result = result[:start] + resolved + result[start+len(prefix)+idEndRel+len(suffix):]
		search = start + len(resolved)
	}
	return result
}

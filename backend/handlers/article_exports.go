package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"collector-backend/middleware"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

func (h *ArticleHandler) Export(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	articles := make([]models.Article, 0)
	for _, article := range h.store.ListArticles() {
		if canAccessArticle(user, article) {
			articles = append(articles, article)
		}
	}
	stamp := time.Now().Format("20060102-150405")
	switch strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "csv"))) {
	case "csv":
		data, err := buildArticleCSV(articles)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 CSV 失败"})
			return
		}
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="articles-%s.csv"`, stamp))
		c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
	case "pdf":
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="articles-%s.pdf"`, stamp))
		c.Data(http.StatusOK, "application/pdf", buildArticlePDF(articles))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的导出格式，可选 csv 或 pdf"})
	}
}

func buildArticleCSV(articles []models.Article) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\xef\xbb\xbf")
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"ID", "标题", "分类", "作者", "状态", "摘要", "正文", "浏览量", "可见范围", "创建时间", "更新时间"}); err != nil {
		return nil, err
	}
	for _, article := range articles {
		visibility := "公开"
		if article.IsPrivate {
			visibility = "私有"
		}
		record := []string{
			strconv.Itoa(article.ID), article.Title, article.Category, article.Author, article.Status,
			article.Summary, article.Content, strconv.Itoa(article.Views), visibility,
			article.CreatedAt.Format(time.RFC3339), article.UpdatedAt.Format(time.RFC3339),
		}
		for index := range record {
			record[index] = safeSpreadsheetCell(record[index])
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func safeSpreadsheetCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func buildArticlePDF(articles []models.Article) []byte {
	lines := []string{"HuaJian_AI 文章导出", "生成时间: " + time.Now().Format("2006-01-02 15:04:05"), ""}
	for _, article := range articles {
		visibility := "公开"
		if article.IsPrivate {
			visibility = "私有"
		}
		lines = append(lines, wrapPDFText(fmt.Sprintf("#%d %s", article.ID, article.Title), 38)...)
		lines = append(lines, wrapPDFText(fmt.Sprintf("分类: %s  作者: %s", article.Category, article.Author), 38)...)
		lines = append(lines, wrapPDFText(fmt.Sprintf("状态: %s  浏览量: %d  范围: %s", article.Status, article.Views, visibility), 38)...)
		lines = append(lines, wrapPDFText("摘要: "+article.Summary, 38)...)
		lines = append(lines, wrapPDFText("正文: "+article.Content, 38)...)
		lines = append(lines, "")
	}
	return renderCJKPDF(lines)
}

func wrapPDFText(value string, width int) []string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
	result := []string{}
	for _, sourceLine := range strings.Split(value, "\n") {
		runes := []rune(sourceLine)
		if len(runes) == 0 {
			result = append(result, "")
			continue
		}
		for len(runes) > width {
			result = append(result, string(runes[:width]))
			runes = runes[width:]
		}
		result = append(result, string(runes))
	}
	return result
}

func renderCJKPDF(lines []string) []byte {
	const linesPerPage = 50
	if len(lines) == 0 {
		lines = []string{""}
	}
	pageCount := (len(lines) + linesPerPage - 1) / linesPerPage
	objects := make([][]byte, 5+pageCount*2)
	objects[1] = []byte(`<< /Type /Catalog /Pages 2 0 R >>`)
	kids := make([]string, 0, pageCount)
	for page := 0; page < pageCount; page++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", 5+page*2))
	}
	objects[2] = []byte(fmt.Sprintf(`<< /Type /Pages /Kids [%s] /Count %d >>`, strings.Join(kids, " "), pageCount))
	objects[3] = []byte(`<< /Type /Font /Subtype /Type0 /BaseFont /STSong-Light /Encoding /UniGB-UCS2-H /DescendantFonts [4 0 R] >>`)
	objects[4] = []byte(`<< /Type /Font /Subtype /CIDFontType0 /BaseFont /STSong-Light /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 4 >> /DW 1000 >>`)
	for page := 0; page < pageCount; page++ {
		pageID := 5 + page*2
		contentID := pageID + 1
		objects[pageID] = []byte(fmt.Sprintf(`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>`, contentID))
		start := page * linesPerPage
		end := start + linesPerPage
		if end > len(lines) {
			end = len(lines)
		}
		var content strings.Builder
		content.WriteString("BT\n/F1 10 Tf\n50 800 Td\n14 TL\n")
		for _, line := range lines[start:end] {
			content.WriteString("<")
			content.WriteString(pdfUCS2Hex(line))
			content.WriteString("> Tj\nT*\n")
		}
		content.WriteString("ET\n")
		stream := []byte(content.String())
		objects[contentID] = []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}

	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects))
	for id := 1; id < len(objects); id++ {
		offsets[id] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n", id)
		output.Write(objects[id])
		output.WriteString("\nendobj\n")
	}
	xrefOffset := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n", len(objects))
	output.WriteString("0000000000 65535 f \n")
	for id := 1; id < len(objects); id++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[id])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects), xrefOffset)
	return output.Bytes()
}

func pdfUCS2Hex(value string) string {
	var result strings.Builder
	for _, item := range value {
		if item > 0xffff {
			item = '?'
		}
		fmt.Fprintf(&result, "%04X", item)
	}
	return result.String()
}

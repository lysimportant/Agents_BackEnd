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

// Export 实现对应业务逻辑。
func (h *ArticleHandler) Export(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// articles 保存文章。
	articles := make([]models.Article, 0)
	// article 表示当前循环中的索引、键或业务元素。
	for _, article := range h.store.ListArticles() {
		if canAccessArticle(user, article) {
			articles = append(articles, article)
		}
	}
	// stamp 保存时间标记。
	stamp := time.Now().Format("20060102-150405")
	switch strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "csv"))) {
	case "csv":
		// data、err 保存当前操作结果以及可能返回的错误状态。
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

// buildArticleCSV 转换并生成对应业务结果。
func buildArticleCSV(articles []models.Article) ([]byte, error) {
	// buffer 保存变量 buffer。
	var buffer bytes.Buffer
	buffer.WriteString("\xef\xbb\xbf")
	// writer 保存变量 writer。
	writer := csv.NewWriter(&buffer)
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := writer.Write([]string{"ID", "标题", "分类", "作者", "状态", "摘要", "正文", "浏览量", "可见范围", "创建时间", "更新时间"}); err != nil {
		return nil, err
	}
	// article 表示当前循环中的索引、键或业务元素。
	for _, article := range articles {
		// visibility 保存可见状态。
		visibility := "公开"
		if article.IsPrivate {
			visibility = "私有"
		}
		// record 保存记录。
		record := []string{
			strconv.Itoa(article.ID), article.Title, article.Category, article.Author, article.Status,
			article.Summary, article.Content, strconv.Itoa(article.Views), visibility,
			article.CreatedAt.Format(time.RFC3339), article.UpdatedAt.Format(time.RFC3339),
		}
		// index 表示当前循环中的索引、键或业务元素。
		for index := range record {
			record[index] = safeSpreadsheetCell(record[index])
		}
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// safeSpreadsheetCell 实现对应业务逻辑。
func safeSpreadsheetCell(value string) string {
	// trimmed 保存去除空白后的内容。
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

// buildArticlePDF 转换并生成对应业务结果。
func buildArticlePDF(articles []models.Article) []byte {
	// lines 保存变量 lines。
	lines := []string{"HuaJian_AI 文章导出", "生成时间: " + time.Now().Format("2006-01-02 15:04:05"), ""}
	// article 表示当前循环中的索引、键或业务元素。
	for _, article := range articles {
		// visibility 保存可见状态。
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

// wrapPDFText 实现对应业务逻辑。
func wrapPDFText(value string, width int) []string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
	// result 保存操作结果。
	result := []string{}
	// sourceLine 表示当前循环中的索引、键或业务元素。
	for _, sourceLine := range strings.Split(value, "\n") {
		// runes 保存字符序列。
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

// renderCJKPDF 实现对应业务逻辑。
func renderCJKPDF(lines []string) []byte {
	// linesPerPage 保存页码。
	const linesPerPage = 50
	if len(lines) == 0 {
		lines = []string{""}
	}
	// pageCount 保存页码数量。
	pageCount := (len(lines) + linesPerPage - 1) / linesPerPage
	// objects 保存变量 objects。
	objects := make([][]byte, 5+pageCount*2)
	objects[1] = []byte(`<< /Type /Catalog /Pages 2 0 R >>`)
	// kids 保存变量 kids。
	kids := make([]string, 0, pageCount)
	for page := 0; page < pageCount; page++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", 5+page*2))
	}
	objects[2] = []byte(fmt.Sprintf(`<< /Type /Pages /Kids [%s] /Count %d >>`, strings.Join(kids, " "), pageCount))
	objects[3] = []byte(`<< /Type /Font /Subtype /Type0 /BaseFont /STSong-Light /Encoding /UniGB-UCS2-H /DescendantFonts [4 0 R] >>`)
	objects[4] = []byte(`<< /Type /Font /Subtype /CIDFontType0 /BaseFont /STSong-Light /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 4 >> /DW 1000 >>`)
	for page := 0; page < pageCount; page++ {
		// pageID 保存页码标识。
		pageID := 5 + page*2
		// contentID 保存内容标识。
		contentID := pageID + 1
		objects[pageID] = []byte(fmt.Sprintf(`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>`, contentID))
		// start 保存开始位置。
		start := page * linesPerPage
		// end 保存结束位置。
		end := start + linesPerPage
		if end > len(lines) {
			end = len(lines)
		}
		// content 保存内容。
		var content strings.Builder
		content.WriteString("BT\n/F1 10 Tf\n50 800 Td\n14 TL\n")
		// line 表示当前循环中的索引、键或业务元素。
		for _, line := range lines[start:end] {
			content.WriteString("<")
			content.WriteString(pdfUCS2Hex(line))
			content.WriteString("> Tj\nT*\n")
		}
		content.WriteString("ET\n")
		// stream 保存变量 stream。
		stream := []byte(content.String())
		objects[contentID] = []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}

	// output 保存变量 output。
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	// offsets 保存偏移量。
	offsets := make([]int, len(objects))
	for id := 1; id < len(objects); id++ {
		offsets[id] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n", id)
		output.Write(objects[id])
		output.WriteString("\nendobj\n")
	}
	// xrefOffset 保存偏移量。
	xrefOffset := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n", len(objects))
	output.WriteString("0000000000 65535 f \n")
	for id := 1; id < len(objects); id++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[id])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects), xrefOffset)
	return output.Bytes()
}

// pdfUCS2Hex 实现对应业务逻辑。
func pdfUCS2Hex(value string) string {
	// result 保存操作结果。
	var result strings.Builder
	// item 表示当前循环中的索引、键或业务元素。
	for _, item := range value {
		if item > 0xffff {
			item = '?'
		}
		fmt.Fprintf(&result, "%04X", item)
	}
	return result.String()
}

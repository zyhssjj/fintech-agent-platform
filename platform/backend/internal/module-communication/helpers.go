package communication

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/datatypes"

	basedto "wfagent/internal/base/dto"
)

var placeholderPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)
var xmlTagPattern = regexp.MustCompile(`<[^>]+>`)

func mustJSON(value any, fallback string) datatypes.JSON {
	if value == nil {
		return datatypes.JSON([]byte(fallback))
	}
	data, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte(fallback))
	}
	return datatypes.JSON(data)
}

func decodeJSON[T any](value datatypes.JSON, target *T) error {
	if len(value) == 0 {
		return nil
	}
	return json.Unmarshal(value, target)
}

func toRecipients(items []basedto.PersonRef) datatypes.JSON {
	return mustJSON(items, "[]")
}

func fromRecipients(value datatypes.JSON) []basedto.PersonRef {
	var items []basedto.PersonRef
	_ = decodeJSON(value, &items)
	if items == nil {
		return []basedto.PersonRef{}
	}
	return items
}

func fromStringArray(value datatypes.JSON) []string {
	var items []string
	_ = decodeJSON(value, &items)
	if items == nil {
		return []string{}
	}
	return items
}

func renderTemplateString(source string, variables map[string]string) string {
	return placeholderPattern.ReplaceAllStringFunc(source, func(token string) string {
		match := placeholderPattern.FindStringSubmatch(token)
		if len(match) < 2 {
			return token
		}
		return html.EscapeString(strings.TrimSpace(variables[match[1]]))
	})
}

func extractPlaceholderNames(source string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(source, -1)
	seen := map[string]struct{}{}
	result := make([]string, 0, len(matches))
	for _, item := range matches {
		if len(item) < 2 {
			continue
		}
		name := strings.TrimSpace(item[1])
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func summarizeHTML(value string) string {
	text := strings.TrimSpace(html.UnescapeString(htmlTagPattern.ReplaceAllString(value, " ")))
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return truncateString(text, 180)
}

func classifyEmailPriority(subject, body string, tags []string) (int, string, bool, string, string) {
	text := strings.ToLower(subject + " " + body + " " + strings.Join(tags, " "))
	score := 35
	label := "normal"
	needsReply := strings.Contains(text, "回复") || strings.Contains(text, "答复") || strings.Contains(text, "reply") || strings.Contains(text, "请尽快")
	if strings.Contains(text, "紧急") || strings.Contains(text, "urgent") || strings.Contains(text, "asap") || strings.Contains(text, "立即") {
		score += 50
		label = "urgent"
	} else if strings.Contains(text, "重要") || strings.Contains(text, "important") || strings.Contains(text, "审批") || strings.Contains(text, "签报") {
		score += 25
		label = "high"
	}
	summary := summarizeHTML(body)
	replyDraft := ""
	if needsReply {
		replyDraft = "已收到您的邮件，我们正在处理并会尽快回复。"
	}
	return score, label, needsReply, summary, replyDraft
}

func withMailboxSignature(bodyHTML, signatureHTML string) string {
	bodyHTML = strings.TrimSpace(bodyHTML)
	signatureHTML = strings.TrimSpace(signatureHTML)
	if signatureHTML == "" {
		return bodyHTML
	}
	if bodyHTML == "" {
		return signatureHTML
	}
	return bodyHTML + "<br/><br/>" + signatureHTML
}

func truncateString(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	value = strings.ToValidUTF8(value, "")
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

func sanitizeMailSubject(value string) string {
	value = strings.ToValidUTF8(value, "")
	for _, marker := range []string{"邮件正文建议:", "邮件正文：", "邮件正文:", "正文建议：", "正文建议:", "建议正文：", "建议正文:"} {
		if index := strings.Index(value, marker); index >= 0 {
			value = value[:index]
		}
	}
	value = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(value), " ")
	return truncateString(value, 120)
}

func validateExternalDocumentContent(subject, bodyHTML string) error {
	value := cleanUTF8(subject + "\n" + bodyHTML)
	for _, marker := range []string{
		"[发件人姓名]", "[发件人职位]", "[发件人单位]", "[联系方式]",
		"{发件人姓名}", "{发件人职位}", "{发件人单位}", "{联系方式}",
	} {
		if strings.Contains(value, marker) {
			return fmt.Errorf("函件正文包含未填写的“%s”，请在可编辑正文中填写真实信息后重新创建函件", strings.Trim(marker, "[]{}"))
		}
	}
	if strings.Contains(value, "{{") || strings.Contains(value, "}}") {
		return fmt.Errorf("函件正文仍包含未替换的模板字段，请补全字段后重新创建函件")
	}
	return nil
}

func cleanUTF8(value string) string {
	return strings.ToValidUTF8(value, "")
}

func cleanPersonRef(item basedto.PersonRef) basedto.PersonRef {
	item.UserID = cleanUTF8(item.UserID)
	item.Name = cleanUTF8(item.Name)
	item.Email = cleanUTF8(item.Email)
	return item
}

func cleanPersonRefs(items []basedto.PersonRef) []basedto.PersonRef {
	cleaned := make([]basedto.PersonRef, 0, len(items))
	for _, item := range items {
		cleaned = append(cleaned, cleanPersonRef(item))
	}
	return cleaned
}

func encodeBase64(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func decodeBase64(data string) ([]byte, error) {
	if strings.TrimSpace(data) == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(data)
}

func buildDOCXFromTemplate(templateData []byte, variables map[string]string) ([]byte, error) {
	if len(templateData) == 0 {
		return buildFallbackDOCX(variables), nil
	}
	docxVars, err := extractDOCXPlaceholders(templateData)
	if err != nil || len(docxVars) == 0 {
		return buildFallbackDOCX(variables), nil
	}
	readerAt := bytes.NewReader(templateData)
	zr, err := zip.NewReader(readerAt, int64(len(templateData)))
	if err != nil {
		return buildFallbackDOCX(variables), nil
	}
	buffer := &bytes.Buffer{}
	zw := zip.NewWriter(buffer)
	for _, file := range zr.File {
		src, err := file.Open()
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		content, err := io.ReadAll(src)
		_ = src.Close()
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if file.Name == "word/document.xml" {
			text := string(content)
			for key, value := range variables {
				text = strings.ReplaceAll(text, "{{"+key+"}}", html.EscapeString(value))
			}
			content = []byte(text)
		}
		writer, err := zw.CreateHeader(&file.FileHeader)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := writer.Write(content); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func extractDOCXPlaceholders(templateData []byte) ([]string, error) {
	readerAt := bytes.NewReader(templateData)
	zr, err := zip.NewReader(readerAt, int64(len(templateData)))
	if err != nil {
		return nil, err
	}
	for _, file := range zr.File {
		if file.Name != "word/document.xml" {
			continue
		}
		src, err := file.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(src)
		_ = src.Close()
		if err != nil {
			return nil, err
		}
		return extractPlaceholderNames(string(content)), nil
	}
	return nil, errors.New("word/document.xml not found")
}

func extractDOCXText(templateData []byte) (string, []string, error) {
	readerAt := bytes.NewReader(templateData)
	zr, err := zip.NewReader(readerAt, int64(len(templateData)))
	if err != nil {
		return "", nil, err
	}
	for _, file := range zr.File {
		if file.Name != "word/document.xml" {
			continue
		}
		src, err := file.Open()
		if err != nil {
			return "", nil, err
		}
		content, err := io.ReadAll(src)
		_ = src.Close()
		if err != nil {
			return "", nil, err
		}
		xmlText := string(content)
		xmlText = strings.ReplaceAll(xmlText, "</w:p>", "\n")
		xmlText = strings.ReplaceAll(xmlText, "</w:tr>", "\n")
		xmlText = strings.ReplaceAll(xmlText, "</w:tc>", " ")
		plain := strings.TrimSpace(html.UnescapeString(xmlTagPattern.ReplaceAllString(xmlText, " ")))
		plain = regexp.MustCompile(`[\t\r ]+`).ReplaceAllString(plain, " ")
		lines := []string{}
		for _, part := range strings.Split(plain, "\n") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			lines = append(lines, part)
		}
		return strings.Join(lines, "\n"), lines, nil
	}
	return "", nil, errors.New("word/document.xml not found")
}

func validateVariableCoverage(definitions []TemplateVariable, htmlContent string, docxData []byte) error {
	defined := make([]string, 0, len(definitions))
	for _, variable := range definitions {
		name := strings.TrimSpace(variable.VariableName)
		if name == "" {
			continue
		}
		defined = append(defined, name)
	}
	for _, variable := range extractPlaceholderNames(htmlContent) {
		if !slices.Contains(defined, variable) {
			return fmt.Errorf("html template placeholder %s is missing from variable definitions", variable)
		}
	}
	if len(docxData) == 0 {
		return nil
	}
	docxVars, err := extractDOCXPlaceholders(docxData)
	if err != nil || len(docxVars) == 0 {
		return nil
	}
	for _, variable := range docxVars {
		if !slices.Contains(defined, variable) {
			return fmt.Errorf("docx template placeholder %s is missing from variable definitions", variable)
		}
	}
	return nil
}

func ensureVariableValues(definitions []TemplateVariable, raw map[string]any) (map[string]string, error) {
	result := make(map[string]string, len(definitions))
	for _, item := range definitions {
		value := ""
		if raw != nil {
			if got, ok := raw[item.VariableName]; ok && got != nil {
				value = fmt.Sprint(got)
			}
		}
		if strings.TrimSpace(value) == "" {
			value = item.DefaultValue
		}
		if item.IsRequired && strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("missing required variable: %s", item.VariableName)
		}
		result[item.VariableName] = value
	}
	return result, nil
}

func roleIsInternal(roles []string) bool {
	internal := []string{"SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"}
	for _, role := range roles {
		if slices.Contains(internal, role) {
			return true
		}
	}
	return false
}

func formatRFC3339(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func buildFallbackDOCX(variables map[string]string) []byte {
	buffer := &strings.Builder{}
	buffer.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buffer.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	buffer.WriteString(`<w:p><w:r><w:t>` + html.EscapeString(strings.TrimSpace(variables["fund_name"])) + `季度沟通通知</w:t></w:r></w:p>`)
	buffer.WriteString(`<w:p><w:r><w:t>尊敬的` + html.EscapeString(strings.TrimSpace(variables["lp_name"])) + `，您好。</w:t></w:r></w:p>`)
	buffer.WriteString(`<w:p><w:r><w:t>通知日期：` + html.EscapeString(strings.TrimSpace(variables["notice_date"])) + `</w:t></w:r></w:p>`)
	buffer.WriteString(`</w:body></w:document>`)
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`
	return buildStaticDOCX(contentTypes, rels, docRels, buffer.String())
}

func buildHTMLDocumentDOCX(title, bodyHTML string) []byte {
	plainText := html.UnescapeString(htmlTagPattern.ReplaceAllString(bodyHTML, "\n"))
	plainText = strings.ReplaceAll(plainText, "\r\n", "\n")
	plainText = strings.ReplaceAll(plainText, "\r", "\n")
	lines := make([]string, 0)
	if title = strings.TrimSpace(title); title != "" {
		lines = append(lines, title)
	}
	for _, line := range strings.Split(plainText, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		lines = []string{"Communication document"}
	}

	buffer := &strings.Builder{}
	buffer.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buffer.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, line := range lines {
		buffer.WriteString(`<w:p><w:r><w:t xml:space="preserve">`)
		buffer.WriteString(html.EscapeString(line))
		buffer.WriteString(`</w:t></w:r></w:p>`)
	}
	buffer.WriteString(`</w:body></w:document>`)
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`
	return buildStaticDOCX(contentTypes, rels, docRels, buffer.String())
}

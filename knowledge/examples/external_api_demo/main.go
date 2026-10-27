package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL    = "http://127.0.0.1:8201"
	defaultCompany    = "beijing_motian"
	defaultDomainPath = "基金层面/基金设立/工商注册-会商"
	defaultQuestion   = "文档中记录了哪些网络安全专用产品？请列出命中的代表条目。"
)

type requestBody struct {
	Query       string   `json:"query"`
	CompanyCode string   `json:"company_code"`
	DomainPaths []string `json:"domain_paths"`
	TopK        int      `json:"top_k"`
}

type source struct {
	Title      string  `json:"title"`
	SourceFile string  `json:"source_file"`
	DomainPath string  `json:"domain_path"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
}

type apiResponse struct {
	RequestID string   `json:"request_id"`
	Answer    *string  `json:"answer"`
	RoutedTo  []string `json:"routed_to"`
	Sources   []source `json:"sources"`
}

func main() {
	baseURL := flag.String("base-url", envOr("KNOWLEDGE_BASE_URL", defaultBaseURL), "知识库服务地址")
	mode := flag.String("mode", "retrieve", "调用模式：retrieve 或 answer")
	question := flag.String("question", defaultQuestion, "测试问题")
	company := flag.String("company", defaultCompany, "平台代码")
	domain := flag.String("domain", defaultDomainPath, "一级、二级或三级知识域路径")
	flag.Parse()

	apiKey, err := readAPIKey()
	if err != nil {
		fatal(err)
	}
	if *mode != "retrieve" && *mode != "answer" {
		fatal(errors.New("mode 只能是 retrieve 或 answer"))
	}

	payload, err := json.Marshal(requestBody{
		Query:       *question,
		CompanyCode: *company,
		DomainPaths: []string{*domain},
		TopK:        3,
	})
	if err != nil {
		fatal(err)
	}

	endpoint := strings.TrimRight(*baseURL, "/") + "/api/v1/" + *mode
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	started := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		fatal(fmt.Errorf("请求失败: %w", err))
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		fatal(fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data))))
	}

	var result apiResponse
	if err := json.Unmarshal(data, &result); err != nil {
		fatal(fmt.Errorf("解析响应失败: %w", err))
	}

	fmt.Printf("调用成功: HTTP %d，耗时 %s\n", resp.StatusCode, time.Since(started).Round(time.Millisecond))
	fmt.Printf("请求编号: %s\n", result.RequestID)
	routedTo := result.RoutedTo
	if len(routedTo) == 0 {
		routedTo = []string{*domain}
	}
	fmt.Printf("检索范围: %s / %s\n", *company, strings.Join(routedTo, ", "))
	if result.Answer != nil {
		fmt.Printf("\n回答:\n%s\n", *result.Answer)
	}
	fmt.Printf("\n证据数量: %d\n", len(result.Sources))
	for i, item := range result.Sources {
		content := []rune(strings.TrimSpace(item.Content))
		if len(content) > 180 {
			content = append(content[:180], []rune("...")...)
		}
		fmt.Printf("\n[%d] %s\n领域: %s\n分数: %.4f\n片段: %s\n", i+1, firstNonEmpty(item.Title, item.SourceFile, "未命名来源"), item.DomainPath, item.Score, string(content))
	}
}

func readAPIKey() (string, error) {
	if value := strings.TrimSpace(os.Getenv("KNOWLEDGE_API_KEY")); value != "" {
		return value, nil
	}
	fmt.Print("请输入知识库 API Key: ")
	value, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("API Key 不能为空")
	}
	return value, nil
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}

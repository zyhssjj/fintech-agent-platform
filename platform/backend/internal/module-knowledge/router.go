package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wfagent/internal/base/config"
	"wfagent/internal/base/middleware"
	"wfagent/internal/base/response"
)

var errNotConfigured = errors.New("知识库服务未配置，请设置 KNOWLEDGE_SERVICE_BASE_URL 和 KNOWLEDGE_SERVICE_API_KEY")

type QueryRequest struct {
	Query       string   `json:"query" binding:"required,max=2000"`
	CompanyCode string   `json:"company_code" binding:"required,max=64"`
	DomainPaths []string `json:"domain_paths,omitempty"`
	TopK        int      `json:"top_k,omitempty"`
}

type Handler struct{ client *http.Client }

func RegisterRoutes(v1 *gin.RouterGroup, _ *gorm.DB) {
	h := &Handler{}
	group := v1.Group("/agents/knowledge")
	group.Use(middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"))
	group.POST("/search", h.search)
	group.POST("/answer", h.answer)
}

func (h *Handler) search(c *gin.Context) { h.proxy(c, "/api/v1/retrieve") }
func (h *Handler) answer(c *gin.Context) { h.proxy(c, "/api/v1/answer") }

func (h *Handler) proxy(c *gin.Context, path string) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		response.BindError(c, errors.New("请输入检索问题"))
		return
	}
	if req.TopK <= 0 {
		req.TopK = 8
	}
	if req.TopK > 20 {
		req.TopK = 20
	}
	if !canAccessCompany(c, req.CompanyCode) {
		response.Error(c, response.ErrForbidden, "无权访问该公司知识域")
		return
	}
	data, err := h.call(c.Request.Context(), path, req)
	if err != nil {
		response.Error(c, response.ErrUnavailable, err.Error())
		return
	}
	response.Success(c, data)
}

func (h *Handler) call(ctx context.Context, path string, req QueryRequest) (map[string]any, error) {
	cfg := config.C.Knowledge
	if strings.TrimSpace(cfg.ServiceBaseURL) == "" || strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errNotConfigured
	}
	body := map[string]any{"query": req.Query, "company_code": req.CompanyCode, "domain_paths": req.DomainPaths, "top_k": req.TopK}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := h.client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	url := strings.TrimRight(cfg.ServiceBaseURL, "/") + path
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("知识库服务连接失败: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("知识库服务返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var data map[string]any
	if err := json.Unmarshal(responseBody, &data); err != nil {
		return nil, err
	}
	return data, nil
}

var companyCodes = map[string]struct{}{
	"beijing_hangke": {}, "beijing_motian": {}, "neimeng_juyuan": {},
	"hunan_hangtian": {}, "hangmo_group": {}, "future_intelligence": {},
}

func canAccessCompany(c *gin.Context, companyCode string) bool {
	if _, ok := companyCodes[companyCode]; !ok {
		return false
	}
	roles, _ := c.Get("roles")
	userRoles, _ := roles.([]string)
	for _, role := range userRoles {
		if role == "SYS_ADMIN" || role == "*" {
			return true
		}
	}
	// 主系统尚未维护用户所属公司，已鉴权的内部角色暂按公司白名单访问；
	// 后续接入人员公司归属后在此处收紧到个人范围。
	return true
}

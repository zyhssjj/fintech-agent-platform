package communication

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	approval "wfagent/internal/module-approval"
)

type envelope[T any] struct {
	Code int `json:"code"`
	Data T   `json:"data"`
}

func TestCommunicationAndApprovalSmoke(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := approval.AutoMigrate(db); err != nil {
		t.Fatalf("approval migrate: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("communication migrate: %v", err)
	}
	if err := approval.SeedDefaults(db); err != nil {
		t.Fatalf("approval seed: %v", err)
	}
	if err := SeedDemoData(db); err != nil {
		t.Fatalf("communication seed: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "ops_manager")
		c.Set("roles", []string{"OPS_MANAGER"})
		c.Next()
	})
	v1 := router.Group("/v1")
	approval.RegisterRoutes(v1, db)
	RegisterRoutes(v1, db)

	mustStatusOK(t, router, http.MethodGet, "/v1/agents/communication/mailboxes", nil)
	mustStatusOK(t, router, http.MethodGet, "/v1/agents/communication/emails", nil)
	mustStatusOK(t, router, http.MethodGet, "/v1/agents/communication/templates", nil)
	mustStatusOK(t, router, http.MethodGet, "/v1/approvals", nil)

	templateBody := map[string]any{
		"name":                "测试函件模板",
		"category":            "notice",
		"scene":               "lp_communication",
		"subject_template":    "{{fund_name}}季度沟通通知",
		"html_content":        "<h1>{{fund_name}}季度沟通通知</h1><p>{{lp_name}}</p><p>{{notice_date}}</p>",
		"docx_template_name":  "test.docx",
		"docx_template_base64": encodeBase64(sampleDOCXTemplate()),
		"variables": []map[string]any{
			{"variable_name": "fund_name", "variable_label": "基金名称", "variable_type": "text", "is_required": true},
			{"variable_name": "lp_name", "variable_label": "LP名称", "variable_type": "text", "is_required": true},
			{"variable_name": "notice_date", "variable_label": "通知日期", "variable_type": "date", "is_required": true},
		},
	}
	templateResp := mustStatusOK(t, router, http.MethodPost, "/v1/agents/communication/templates", templateBody)
	var createdTemplate envelope[TemplateView]
	if err := json.Unmarshal(templateResp.Body.Bytes(), &createdTemplate); err != nil {
		t.Fatalf("decode template: %v", err)
	}
	if createdTemplate.Data.TemplateID == "" {
		t.Fatalf("template id should not be empty")
	}

	documentBody := map[string]any{
		"template_id":     "1",
		"recipient_name":  "LP测试用户",
		"recipient_email": "lp@example.com",
		"title":           "LP测试通知",
		"document_type":   "notice",
		"visible_user_ids": []string{"1"},
		"variables": map[string]any{
			"fund_name":   "示例基金一期",
			"lp_name":     "LP测试用户",
			"notice_date": "2026-07-23",
		},
	}
	documentResp := mustStatusOK(t, router, http.MethodPost, "/v1/agents/communication/documents", documentBody)
	var createdDocument envelope[DocumentView]
	if err := json.Unmarshal(documentResp.Body.Bytes(), &createdDocument); err != nil {
		t.Fatalf("decode document: %v", err)
	}
	if createdDocument.Data.DocumentID == "" {
		t.Fatalf("document id should not be empty")
	}

	submitResp := mustStatusOK(t, router, http.MethodPost, "/v1/agents/communication/documents/"+createdDocument.Data.DocumentID+"/submit", map[string]any{})
	var submitted envelope[map[string]any]
	if err := json.Unmarshal(submitResp.Body.Bytes(), &submitted); err != nil {
		t.Fatalf("decode submit: %v", err)
	}
	approvalID, _ := submitted.Data["approval_id"].(string)
	if approvalID == "" {
		t.Fatalf("approval id should not be empty")
	}

	mustStatusOK(t, router, http.MethodPost, "/v1/approvals/"+approvalID+"/action", map[string]any{"action": "approve", "comment": "通过"})
	mustStatusOK(t, router, http.MethodGet, "/v1/agents/communication/documents", nil)
}

func mustStatusOK(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code >= 400 {
		t.Fatalf("%s %s failed: %d %s", method, path, recorder.Code, recorder.Body.String())
	}
	return recorder
}

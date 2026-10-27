package communication

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	baseai "wfagent/internal/base/ai"
)

func TestTemplateAIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response baseai.TemplateDraftResponse
		switch r.URL.Path {
		case "/template/chat":
			response = baseai.TemplateDraftResponse{
				Summary:           "chat route ok",
				FollowUpQuestions: []string{"请补充使用场景"},
				SuggestedFields: []baseai.TemplateField{
					{VariableName: "partner_name", VariableLabel: "合作方名称", VariableType: "text", IsRequired: true},
				},
				SubjectTemplate: "关于{{partner_name}}的正式通知",
				HTMLTemplate:    "<p>{{partner_name}}</p>",
			}
		case "/template/import":
			response = baseai.TemplateDraftResponse{
				Summary:           "import route ok",
				FollowUpQuestions: []string{"请确认哪些字段允许编辑"},
				SuggestedFields: []baseai.TemplateField{
					{VariableName: "fund_name", VariableLabel: "基金名称", VariableType: "text", IsRequired: true},
				},
				SubjectTemplate: "关于{{fund_name}}的正式通知",
				HTMLTemplate:    "<p>{{fund_name}}</p>",
			}
		case "/template/render-assist":
			response = baseai.TemplateDraftResponse{
				Summary:         "render route ok",
				SubjectTemplate: "正式函件建议",
				HTMLTemplate:    "<p>建议正文</p>",
			}
		default:
			t.Fatalf("unexpected AI path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	db := openCommunicationTestDB(t)
	handler := &Handler{service: newServiceWithAIClient(db, baseai.New(server.URL, time.Second))}

	router := gin.New()
	group := router.Group("/v1/agents/communication/templates/ai")
	group.POST("/chat", handler.DraftTemplateFromChat)
	group.POST("/import", handler.DraftTemplateFromImport)
	group.POST("/render-assist", handler.AssistTemplateRender)

	chatResp := mustStatusOKRecorder(t, router, http.MethodPost, "/v1/agents/communication/templates/ai/chat", map[string]any{
		"scene":               "strategic_cooperation",
		"goal":                "生成合作通知模板",
		"latest_user_message": "需要正式、专业、可审计",
	})
	var chatEnvelope envelope[TemplateAssistResponse]
	if err := json.Unmarshal(chatResp.Body.Bytes(), &chatEnvelope); err != nil {
		t.Fatalf("decode chat response: %v", err)
	}
	if chatEnvelope.Data.Summary != "chat route ok" {
		t.Fatalf("unexpected chat summary: %s", chatEnvelope.Data.Summary)
	}

	importResp := mustStatusOKRecorder(t, router, http.MethodPost, "/v1/agents/communication/templates/ai/import", map[string]any{
		"file_name":            "import.docx",
		"scene":                "investor_relations",
		"docx_template_base64": encodeBase64(sampleDOCXTemplate()),
	})
	var importEnvelope envelope[TemplateAssistResponse]
	if err := json.Unmarshal(importResp.Body.Bytes(), &importEnvelope); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if importEnvelope.Data.Summary != "import route ok" {
		t.Fatalf("unexpected import summary: %s", importEnvelope.Data.Summary)
	}
	if importEnvelope.Data.PlainText == "" {
		t.Fatalf("expected plain text extracted from docx")
	}

	renderResp := mustStatusOKRecorder(t, router, http.MethodPost, "/v1/agents/communication/templates/ai/render-assist", map[string]any{
		"template_name":  "合作通知",
		"template_scene": "strategic_cooperation",
		"recipient_type": "partner",
		"business_goal":  "生成正式对外沟通正文",
		"known_values": map[string]any{
			"partner_name": "示例合作方",
		},
	})
	var renderEnvelope envelope[TemplateAssistResponse]
	if err := json.Unmarshal(renderResp.Body.Bytes(), &renderEnvelope); err != nil {
		t.Fatalf("decode render response: %v", err)
	}
	if renderEnvelope.Data.HTMLTemplate != "<p>建议正文</p>" {
		t.Fatalf("unexpected render html: %s", renderEnvelope.Data.HTMLTemplate)
	}
}

func mustStatusOKRecorder(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
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

package communication

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	baseai "wfagent/internal/base/ai"
)

func TestDraftTemplateFromChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/template/chat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(baseai.TemplateDraftResponse{
			Summary:           "chat summary",
			FollowUpQuestions: []string{"Q1"},
			SuggestedFields: []baseai.TemplateField{
				{VariableName: "partner_name", VariableLabel: "合作方名称", VariableType: "text", IsRequired: true},
			},
			SubjectTemplate: "关于{{partner_name}}的通知",
			HTMLTemplate:    "<p>{{partner_name}}</p>",
		})
	}))
	defer server.Close()

	service := newServiceWithAIClient(openCommunicationTestDB(t), baseai.New(server.URL, time.Second))
	result, err := service.DraftTemplateFromChat(context.Background(), TemplateChatAssistRequest{
		Scene:             "strategic_cooperation",
		Goal:              "生成合作通知模板",
		LatestUserMessage: "需要正式、简洁",
	})
	if err != nil {
		t.Fatalf("DraftTemplateFromChat: %v", err)
	}
	if result.Summary != "chat summary" {
		t.Fatalf("unexpected summary: %s", result.Summary)
	}
	if len(result.SuggestedFields) != 1 || result.SuggestedFields[0].VariableName != "partner_name" {
		t.Fatalf("unexpected suggested fields: %+v", result.SuggestedFields)
	}
}

func TestDraftTemplateFromImport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/template/import" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(baseai.TemplateDraftResponse{
			Summary:           "import summary",
			FollowUpQuestions: []string{"Q2"},
			SuggestedFields: []baseai.TemplateField{
				{VariableName: "fund_name", VariableLabel: "基金名称", VariableType: "text", IsRequired: true},
			},
			SubjectTemplate: "关于{{fund_name}}的通知",
			HTMLTemplate:    "<p>{{fund_name}}</p>",
		})
	}))
	defer server.Close()

	service := newServiceWithAIClient(openCommunicationTestDB(t), baseai.New(server.URL, time.Second))
	result, err := service.DraftTemplateFromImport(context.Background(), TemplateImportAssistRequest{
		FileName:           "import.docx",
		Scene:              "investor_relations",
		DOCXTemplateBase64: encodeBase64(sampleDOCXTemplate()),
	})
	if err != nil {
		t.Fatalf("DraftTemplateFromImport: %v", err)
	}
	if result.Summary != "import summary" {
		t.Fatalf("unexpected summary: %s", result.Summary)
	}
	if result.PlainText == "" || len(result.Paragraphs) == 0 {
		t.Fatalf("expected extracted docx text, got plain=%q paragraphs=%v", result.PlainText, result.Paragraphs)
	}
}

func TestAssistTemplateRender(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/template/render-assist" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(baseai.TemplateDraftResponse{
			Summary:         "render summary",
			SubjectTemplate: "正式通知",
			HTMLTemplate:    "<p>正文建议</p>",
		})
	}))
	defer server.Close()

	service := newServiceWithAIClient(openCommunicationTestDB(t), baseai.New(server.URL, time.Second))
	result, err := service.AssistTemplateRender(context.Background(), TemplateRenderAssistInput{
		TemplateName:  "合作通知",
		TemplateScene: "strategic_cooperation",
		RecipientType: "partner",
		BusinessGoal:  "生成正式对外通知",
	})
	if err != nil {
		t.Fatalf("AssistTemplateRender: %v", err)
	}
	if result.HTMLTemplate != "<p>正文建议</p>" {
		t.Fatalf("unexpected html template: %s", result.HTMLTemplate)
	}
}

func TestAssistEndpointsRequireConfiguredAIClient(t *testing.T) {
	service := NewService(openCommunicationTestDB(t))
	service.aiClient = nil

	chatResult, err := service.DraftTemplateFromChat(context.Background(), TemplateChatAssistRequest{
		Goal:              "生成模板",
		LatestUserMessage: "请给我正式通知模板",
	})
	if err != nil {
		t.Fatalf("expected chat fallback, got error %v", err)
	}
	if chatResult.HTMLTemplate == "" || chatResult.SubjectTemplate == "" {
		t.Fatalf("expected chat fallback draft, got %+v", chatResult)
	}

	importResult, err := service.DraftTemplateFromImport(context.Background(), TemplateImportAssistRequest{
		FileName:           "test.docx",
		DOCXTemplateBase64: encodeBase64(sampleDOCXTemplate()),
	})
	if err != nil {
		t.Fatalf("expected import fallback, got error %v", err)
	}
	if importResult.HTMLTemplate == "" || importResult.PlainText == "" {
		t.Fatalf("expected import fallback draft, got %+v", importResult)
	}

	renderResult, err := service.AssistTemplateRender(context.Background(), TemplateRenderAssistInput{
		BusinessGoal: "生成正文建议",
	})
	if err != nil {
		t.Fatalf("expected render fallback, got error %v", err)
	}
	if renderResult.HTMLTemplate == "" {
		t.Fatalf("expected render fallback html, got %+v", renderResult)
	}
}

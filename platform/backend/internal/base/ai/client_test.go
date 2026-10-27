package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDraftTemplateFromChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/template/chat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(TemplateDraftResponse{
			Summary: "chat ok",
			SuggestedFields: []TemplateField{
				{VariableName: "partner_name", VariableLabel: "合作方名称", VariableType: "text", IsRequired: true},
			},
		})
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	result, err := client.DraftTemplateFromChat(context.Background(), TemplateChatRequest{
		Scene: "cooperation_notice",
		Goal:  "创建合作通知模板",
	})
	if err != nil {
		t.Fatalf("draft template from chat: %v", err)
	}
	if result.Summary != "chat ok" {
		t.Fatalf("unexpected summary: %s", result.Summary)
	}
	if len(result.SuggestedFields) != 1 || result.SuggestedFields[0].VariableName != "partner_name" {
		t.Fatalf("unexpected fields: %+v", result.SuggestedFields)
	}
}

func TestDraftTemplateFromImport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/template/import" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(TemplateDraftResponse{
			Summary: "import ok",
		})
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	result, err := client.DraftTemplateFromImport(context.Background(), TemplateImportRequest{
		FileName:  "template.docx",
		PlainText: "合作方名称：__________",
		Scene:     "cooperation_notice",
	})
	if err != nil {
		t.Fatalf("draft template from import: %v", err)
	}
	if result.Summary != "import ok" {
		t.Fatalf("unexpected summary: %s", result.Summary)
	}
}

func TestRenderTemplateAssist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/template/render-assist" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(TemplateDraftResponse{
			Summary:         "render ok",
			SubjectTemplate: "关于{{document_title}}的正式通知",
		})
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	result, err := client.RenderTemplateAssist(context.Background(), TemplateRenderAssistRequest{
		TemplateName:  "战略合作沟通模板",
		TemplateScene: "cooperation_notice",
	})
	if err != nil {
		t.Fatalf("render assist: %v", err)
	}
	if result.SubjectTemplate == "" {
		t.Fatalf("expected subject template")
	}
}

func TestClientReturnsReadableErrorWhenAIServiceDoesNotReturnJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	_, err := client.DraftTemplateFromImport(context.Background(), TemplateImportRequest{
		FileName: "template.docx",
	})
	if err == nil {
		t.Fatalf("expected readable error")
	}
	if !strings.Contains(err.Error(), "Internal Server Error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

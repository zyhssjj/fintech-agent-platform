package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"wfagent/internal/base/config"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type TemplateField struct {
	VariableName  string `json:"variable_name"`
	VariableLabel string `json:"variable_label"`
	VariableType  string `json:"variable_type"`
	IsRequired    bool   `json:"is_required"`
}

type TemplateDraftResponse struct {
	Summary           string          `json:"summary"`
	FollowUpQuestions []string        `json:"follow_up_questions"`
	SuggestedFields   []TemplateField `json:"suggested_fields"`
	SubjectTemplate   string          `json:"subject_template"`
	HTMLTemplate      string          `json:"html_template"`
}

type TemplateChatRequest struct {
	Scene             string                   `json:"scene"`
	Goal              string                   `json:"goal"`
	CurrentSummary    string                   `json:"current_summary"`
	LatestUserMessage string                   `json:"latest_user_message"`
	KnownFields       []map[string]interface{} `json:"known_fields"`
}

type TemplateImportRequest struct {
	FileName   string   `json:"file_name"`
	PlainText  string   `json:"plain_text"`
	Paragraphs []string `json:"paragraphs"`
	Scene      string   `json:"scene"`
}

type TemplateRenderAssistRequest struct {
	TemplateName  string                 `json:"template_name"`
	TemplateScene string                 `json:"template_scene"`
	RecipientType string                 `json:"recipient_type"`
	BusinessGoal  string                 `json:"business_goal"`
	KnownValues   map[string]interface{} `json:"known_values"`
}

func NewFromConfig() *Client {
	if config.C == nil {
		return nil
	}
	return New(config.C.AI.ServiceBaseURL, config.C.AI.Timeout)
}

func New(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) DraftTemplateFromChat(ctx context.Context, payload TemplateChatRequest) (TemplateDraftResponse, error) {
	return c.post(ctx, "/template/chat", payload)
}

func (c *Client) DraftTemplateFromImport(ctx context.Context, payload TemplateImportRequest) (TemplateDraftResponse, error) {
	return c.post(ctx, "/template/import", payload)
}

func (c *Client) RenderTemplateAssist(ctx context.Context, payload TemplateRenderAssistRequest) (TemplateDraftResponse, error) {
	return c.post(ctx, "/template/render-assist", payload)
}

func (c *Client) post(ctx context.Context, path string, payload interface{}) (TemplateDraftResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return TemplateDraftResponse{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return TemplateDraftResponse{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return TemplateDraftResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(response.Body)
		message := strings.TrimSpace(string(bodyBytes))
		if message == "" {
			message = fmt.Sprintf("ai service returned http %d", response.StatusCode)
		} else {
			message = fmt.Sprintf("ai service returned http %d: %s", response.StatusCode, message)
		}
		return TemplateDraftResponse{}, fmt.Errorf("%s", message)
	}

	var result TemplateDraftResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return TemplateDraftResponse{}, err
	}
	return result, nil
}

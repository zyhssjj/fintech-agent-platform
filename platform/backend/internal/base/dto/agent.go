package dto

// 本文件定义智能体、待办、知识库、通讯协同接口的传输结构。
type AgentListQuery struct {
	Category string `form:"category"`
	Status   string `form:"status"`
	PaginationQuery
}

type AgentSummary struct {
	AgentID      string   `json:"agent_id"`
	AgentName    string   `json:"agent_name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
	Stats        JSONMap  `json:"stats"`
}

type AgentDetailResponse struct {
	AgentSummary
	Config    JSONMap `json:"config"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type AgentInvokeRequest struct {
	Capability string  `json:"capability" binding:"required"`
	Parameters JSONMap `json:"parameters"`
	Context    JSONMap `json:"context"`
}

type AgentInvokeResponse struct {
	TaskID        string  `json:"task_id"`
	Status        string  `json:"status"`
	Result        JSONMap `json:"result"`
	ExecutionTime int     `json:"execution_time"`
}

type TodoQuery struct {
	Status    string `form:"status"`
	Priority  string `form:"priority"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	PaginationQuery
}

type TodoItem struct {
	TodoID        string    `json:"todo_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	Priority      string    `json:"priority"`
	Status        string    `json:"status"`
	DueDate       string    `json:"due_date"`
	Assignee      PersonRef `json:"assignee"`
	RelatedEntity EntityRef `json:"related_entity"`
	Alerts        []JSONMap `json:"alerts"`
	CreatedAt     string    `json:"created_at"`
}

type CreateTodoRequest struct {
	Title         string    `json:"title" binding:"required"`
	Description   string    `json:"description"`
	Category      string    `json:"category" binding:"required"`
	Priority      string    `json:"priority" binding:"required"`
	DueDate       string    `json:"due_date" binding:"required"`
	AssigneeID    string    `json:"assignee_id"`
	RelatedEntity EntityRef `json:"related_entity"`
}

type UpdateTodoStatusRequest struct {
	Status  string `json:"status" binding:"required"`
	Comment string `json:"comment"`
}

type NotificationRequest struct {
	Title         string    `json:"title" binding:"required"`
	Content       string    `json:"content" binding:"required"`
	Type          string    `json:"type" binding:"required"`
	Recipients    []string  `json:"recipients" binding:"required"`
	Channel       string    `json:"channel"`
	RelatedEntity EntityRef `json:"related_entity"`
}

type KnowledgeDocumentUploadResponse struct {
	DocumentID string `json:"document_id"`
	FileURL    string `json:"file_url"`
	FileSize   int64  `json:"file_size"`
	PageCount  int    `json:"page_count"`
	Status     string `json:"status"`
}

type KnowledgeSearchQuery struct {
	Query     string   `form:"query" binding:"required"`
	Category  string   `form:"category"`
	Tags      []string `form:"tags"`
	DateRange JSONMap  `form:"date_range"`
	TopK      int      `form:"top_k"`
}

type KnowledgeSearchResult struct {
	Results    []JSONMap `json:"results"`
	Highlights []string  `json:"highlights"`
	Total      int       `json:"total"`
}

type CompetitorIntelligenceQuery struct {
	Industry  string   `form:"industry"`
	Keywords  []string `form:"keywords"`
	DateRange JSONMap  `form:"date_range"`
}

type CompetitorIntelligenceItem struct {
	IntelligenceID string   `json:"intelligence_id"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	Source         string   `json:"source"`
	PublishedAt    string   `json:"published_at"`
	Tags           []string `json:"tags"`
}

type EmailQuery struct {
	Folder    string  `form:"folder"`
	IsRead    *bool   `form:"is_read"`
	IsStarred *bool   `form:"is_starred"`
	DateRange JSONMap `form:"date_range"`
	Sender    string  `form:"sender"`
	PaginationQuery
}

type EmailItem struct {
	EmailID        string      `json:"email_id"`
	Subject        string      `json:"subject"`
	Preview        string      `json:"preview"`
	Sender         PersonRef   `json:"sender"`
	Recipients     []PersonRef `json:"recipients"`
	IsRead         bool        `json:"is_read"`
	IsStarred      bool        `json:"is_starred"`
	HasAttachments bool        `json:"has_attachments"`
	ReceivedAt     string      `json:"received_at"`
}

type SendEmailRequest struct {
	To           []string  `json:"to" binding:"required"`
	CC           []string  `json:"cc"`
	BCC          []string  `json:"bcc"`
	Subject      string    `json:"subject" binding:"required"`
	Body         string    `json:"body" binding:"required"`
	Attachments  []FileRef `json:"attachments"`
	TemplateID   string    `json:"template_id"`
	TemplateVars JSONMap   `json:"template_vars"`
}

type EmailTemplateQuery struct {
	Category string `form:"category"`
	PaginationQuery
}

type EmailTemplateItem struct {
	TemplateID      string   `json:"template_id"`
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	SubjectTemplate string   `json:"subject_template"`
	BodyTemplate    string   `json:"body_template"`
	Variables       []string `json:"variables"`
}

type MeetingMinutesRequest struct {
	MeetingID    string   `json:"meeting_id"`
	AudioURL     string   `json:"audio_url"`
	Transcript   string   `json:"transcript"`
	Participants []string `json:"participants" binding:"required"`
	MeetingTitle string   `json:"meeting_title"`
	MeetingTime  string   `json:"meeting_time"`
}

type MeetingMinutesResponse struct {
	MinutesID   string    `json:"minutes_id"`
	Summary     string    `json:"summary"`
	KeyPoints   []JSONMap `json:"key_points"`
	ActionItems []JSONMap `json:"action_items"`
	Decisions   []JSONMap `json:"decisions"`
	Confidence  float64   `json:"confidence"`
}

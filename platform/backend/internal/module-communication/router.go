package communication

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wfagent/internal/base/middleware"
	"wfagent/internal/base/response"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	handler := &Handler{service: NewService(db)}
	group := v1.Group("/agents/communication")
	group.GET("/mailboxes", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.ListMailboxes)
	group.POST("/mailboxes", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.SaveMailbox)
	group.PUT("/mailboxes/:id", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.SaveMailbox)
	group.POST("/mailboxes/:id/test", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "OPS_MANAGER", "LP"), handler.TestMailbox)
	group.POST("/mailboxes/:id/sync", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.SyncMailbox)
	group.POST("/mailboxes/:id/oauth/callback", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "OPS_MANAGER", "LP"), handler.SaveOAuthToken)
	group.POST("/mailboxes/:id/outlook/authorize", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.BeginOutlookAuthorization)
	group.DELETE("/mailboxes/:id", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.DeleteMailbox)
	group.GET("/emails", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.ListEmails)
	group.GET("/emails/summary", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.DailySummary)
	group.GET("/emails/:id", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.GetEmail)
	group.POST("/emails", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.SendEmail)
	group.POST("/emails/:id/reply", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.ReplyEmail)
	group.POST("/emails/:id/forward", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.ForwardEmail)
	group.POST("/emails/:id/read", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.MarkRead)
	group.POST("/emails/:id/star", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.ToggleStar)
	group.DELETE("/emails/:id", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.DeleteEmail)
	group.GET("/templates", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.ListTemplates)
	group.POST("/templates/ai/chat", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.DraftTemplateFromChat)
	group.POST("/templates/ai/import", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.DraftTemplateFromImport)
	group.POST("/templates/ai/render-assist", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.AssistTemplateRender)
	group.GET("/recipients/lps", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.ListLPRecipients)
	group.POST("/templates", handler.SaveTemplate)
	group.GET("/templates/:id", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.GetTemplate)
	group.PUT("/templates/:id", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "OPS_MANAGER"), handler.SaveTemplate)
	group.GET("/templates/:id/versions", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "OPS_MANAGER", "LEGAL_MANAGER", "REVIEWER"), handler.ListTemplateVersions)
	group.POST("/templates/:id/submit", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "OPS_MANAGER"), handler.SubmitTemplate)
	group.DELETE("/templates/:id", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "OPS_MANAGER"), handler.DeleteTemplate)
	group.POST("/documents/render", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.RenderDocument)
	group.POST("/documents", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.CreateDocument)
	group.GET("/documents", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.ListDocuments)
	group.GET("/documents/:id", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.GetDocument)
	group.POST("/documents/:id/submit", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.SubmitDocument)
	group.POST("/documents/:id/send", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.SendDocument)
	group.GET("/documents/:id/download", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER", "LP"), handler.DownloadDocument)
}

// RegisterPublicIntegrationRoutes registers OAuth callbacks, which cannot carry
// the application's bearer token after Microsoft redirects the browser back.
func RegisterPublicIntegrationRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	handler := &Handler{service: NewService(db)}
	v1.GET("/integrations/microsoft/callback", handler.CompleteOutlookAuthorization)
}

func (h *Handler) ListMailboxes(c *gin.Context) {
	userID, _ := c.Get("user_id")
	roles, _ := c.Get("roles")
	data, err := h.service.ListMailboxes(c.Request.Context(), toString(userID), toRoles(roles))
	writeMaybeError(c, data, err)
}

func (h *Handler) SaveMailbox(c *gin.Context) {
	var req MailboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	data, err := h.service.SaveMailbox(c.Request.Context(), c.Param("id"), req, toString(userID))
	writeMaybeError(c, data, err)
}

func (h *Handler) DeleteMailbox(c *gin.Context) {
	err := h.service.DeleteMailbox(c.Request.Context(), c.Param("id"))
	writeMaybeError(c, gin.H{"status": "deleted"}, err)
}

func (h *Handler) SaveOAuthToken(c *gin.Context) {
	var req OAuthCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	err := h.service.SaveOAuthToken(c.Request.Context(), c.Param("id"), req)
	writeMaybeError(c, gin.H{"status": "ok"}, err)
}

func (h *Handler) BeginOutlookAuthorization(c *gin.Context) {
	data, err := h.service.BeginOutlookAuthorization(c.Request.Context(), c.Param("id"), toString(c.GetString("user_id")))
	writeMaybeError(c, data, err)
}

func (h *Handler) CompleteOutlookAuthorization(c *gin.Context) {
	redirectURL, err := h.service.CompleteOutlookAuthorization(c.Request.Context(), c.Query("code"), c.Query("state"), c.Query("error"), c.Query("error_description"))
	if err != nil {
		c.String(http.StatusBadRequest, "Outlook authorization failed: %s", err.Error())
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) TestMailbox(c *gin.Context) {
	data, err := h.service.TestMailbox(c.Request.Context(), c.Param("id"))
	writeMaybeError(c, data, err)
}

func (h *Handler) SyncMailbox(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	data, err := h.service.ManualSync(c.Request.Context(), c.Param("id"), toString(userID), toString(username))
	writeMaybeError(c, data, err)
}

func (h *Handler) ListEmails(c *gin.Context) {
	var query EmailListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	roles, _ := c.Get("roles")
	data, err := h.service.ListEmails(c.Request.Context(), query, toString(userID), toRoles(roles))
	writeMaybeError(c, data, err)
}

func (h *Handler) DailySummary(c *gin.Context) {
	mailboxID := parseUintQuery(c.Query("mailbox_id"))
	userID, _ := c.Get("user_id")
	roles, _ := c.Get("roles")
	data, err := h.service.DailySummary(c.Request.Context(), mailboxID, toString(userID), toRoles(roles))
	writeMaybeError(c, data, err)
}

func (h *Handler) GetEmail(c *gin.Context) {
	data, err := h.service.GetEmail(c.Request.Context(), c.Param("id"))
	writeMaybeError(c, data, err)
}

func (h *Handler) SendEmail(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	data, err := h.service.SendEmail(c.Request.Context(), req, toString(userID), toString(username))
	writeMaybeError(c, data, err)
}

func (h *Handler) ReplyEmail(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	data, err := h.service.SendEmail(c.Request.Context(), req, toString(userID), toString(username))
	writeMaybeError(c, data, err)
}

func (h *Handler) ForwardEmail(c *gin.Context) {
	h.ReplyEmail(c)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.UpdateEmailFlag(c.Request.Context(), c.Param("id"), "is_read", true); err != nil {
		writeMaybeError(c, nil, err)
		return
	}
	response.Success(c, gin.H{"status": "read"})
}

func (h *Handler) ToggleStar(c *gin.Context) {
	next, err := h.service.ToggleEmailFlag(c.Request.Context(), c.Param("id"), "is_starred")
	if err != nil {
		writeMaybeError(c, nil, err)
		return
	}
	response.Success(c, gin.H{"status": "ok", "is_starred": next})
}

func (h *Handler) DeleteEmail(c *gin.Context) {
	err := h.service.DeleteEmail(c.Request.Context(), c.Param("id"))
	writeMaybeError(c, gin.H{"status": "deleted"}, err)
}

func (h *Handler) ListTemplates(c *gin.Context) {
	var query TemplateQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	data, err := h.service.ListTemplates(c.Request.Context(), query, toString(userID))
	writeMaybeError(c, data, err)
}

func (h *Handler) ListLPRecipients(c *gin.Context) {
	data, err := h.service.ListLPRecipients(c.Request.Context())
	writeMaybeError(c, data, err)
}

func (h *Handler) DraftTemplateFromChat(c *gin.Context) {
	var req TemplateChatAssistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	data, err := h.service.DraftTemplateFromChat(c.Request.Context(), req)
	writeMaybeError(c, data, err)
}

func (h *Handler) DraftTemplateFromImport(c *gin.Context) {
	var req TemplateImportAssistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	data, err := h.service.DraftTemplateFromImport(c.Request.Context(), req)
	writeMaybeError(c, data, err)
}

func (h *Handler) AssistTemplateRender(c *gin.Context) {
	var req TemplateRenderAssistInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	data, err := h.service.AssistTemplateRender(c.Request.Context(), req)
	writeMaybeError(c, data, err)
}

func (h *Handler) SaveTemplate(c *gin.Context) {
	var req TemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	data, err := h.service.SaveTemplate(c.Request.Context(), c.Param("id"), req, toString(userID), toString(username))
	writeMaybeError(c, data, err)
}

func (h *Handler) GetTemplate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	data, err := h.service.GetTemplate(c.Request.Context(), c.Param("id"), toString(userID))
	writeMaybeError(c, data, err)
}

func (h *Handler) ListTemplateVersions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	data, err := h.service.ListTemplateVersions(c.Request.Context(), c.Param("id"), toString(userID))
	writeMaybeError(c, data, err)
}

func (h *Handler) SubmitTemplate(c *gin.Context) {
	writeMaybeError(c, nil, errors.New("private templates cannot be upgraded to public templates"))
}

func (h *Handler) DeleteTemplate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	err := h.service.DeleteTemplate(c.Request.Context(), c.Param("id"), toString(userID))
	if err != nil {
		writeMaybeError(c, nil, err)
		return
	}
	response.Success(c, gin.H{"status": "deleted"})
}

func (h *Handler) RenderDocument(c *gin.Context) {
	var req DocumentRenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	data, err := h.service.RenderDocument(c.Request.Context(), req)
	writeMaybeError(c, data, err)
}

func (h *Handler) CreateDocument(c *gin.Context) {
	var req DocumentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	data, err := h.service.CreateDocument(c.Request.Context(), req, toString(userID), toString(username))
	writeMaybeError(c, data, err)
}

func (h *Handler) ListDocuments(c *gin.Context) {
	var query DocumentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	roles, _ := c.Get("roles")
	data, err := h.service.ListDocuments(c.Request.Context(), query, toString(userID), toRoles(roles))
	writeMaybeError(c, data, err)
}

func (h *Handler) GetDocument(c *gin.Context) {
	data, err := h.service.GetDocument(c.Request.Context(), c.Param("id"))
	writeMaybeError(c, data, err)
}

func (h *Handler) SubmitDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	data, err := h.service.SubmitDocument(c.Request.Context(), c.Param("id"), toString(userID), toString(username))
	writeMaybeError(c, data, err)
}

func (h *Handler) SendDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	data, err := h.service.SendDocument(c.Request.Context(), c.Param("id"), toString(userID), toString(username))
	writeMaybeError(c, data, err)
}

func (h *Handler) DownloadDocument(c *gin.Context) {
	data, fileName, err := h.service.DownloadDocument(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeMaybeError(c, nil, err)
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	c.Header("Content-Disposition", `attachment; filename="`+fileName+`"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data)
}

func writeMaybeError(c *gin.Context, data any, err error) {
	if err == nil {
		response.Success(c, data)
		return
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, response.ErrNotFound, err.Error())
	default:
		response.Error(c, response.ErrValidation, err.Error())
	}
}

func toString(value any) string {
	text, _ := value.(string)
	return text
}

func toRoles(value any) []string {
	items, _ := value.([]string)
	if items == nil {
		return []string{}
	}
	return items
}

func parseUintQuery(value string) uint64 {
	got, _ := strconv.ParseUint(value, 10, 64)
	return got
}

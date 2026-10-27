package communication

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	baseai "wfagent/internal/base/ai"
	"wfagent/internal/base/database"
	basedto "wfagent/internal/base/dto"
	approval "wfagent/internal/module-approval"
)

type Service struct {
	db              *gorm.DB
	approvalService *approval.Service
	mailSender      mailSender
	aiClient        *baseai.Client
}

const (
	templateSourceSystemBuiltin   = "system_builtin"
	templateSourceUserPrivate     = "user_private"
	templateSourcePublishedPublic = "published_public"
)

func NewService(db *gorm.DB) *Service {
	return &Service{
		db:              db,
		approvalService: approval.NewService(db),
		mailSender:      newSMTPMailSender(),
		aiClient:        baseai.NewFromConfig(),
	}
}

func newServiceWithMailSender(db *gorm.DB, sender mailSender) *Service {
	service := NewService(db)
	service.mailSender = sender
	return service
}

func newServiceWithAIClient(db *gorm.DB, client *baseai.Client) *Service {
	service := NewService(db)
	service.aiClient = client
	return service
}

type MailboxRequest struct {
	Name            string `json:"name" binding:"required"`
	MailboxType     string `json:"mailbox_type" binding:"required"`
	EmailAddress    string `json:"email_address" binding:"required"`
	AuthType        string `json:"auth_type" binding:"required"`
	SMTPHost        string `json:"smtp_host"`
	SMTPPort        int    `json:"smtp_port"`
	IMAPHost        string `json:"imap_host"`
	IMAPPort        int    `json:"imap_port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	OAuthProvider   string `json:"oauth_provider"`
	SignatureHTML   string `json:"signature_html"`
	IsDefaultSender bool   `json:"is_default_sender"`
}

type OAuthCallbackRequest struct {
	AccessToken  string `json:"access_token" binding:"required"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}

type EmailListQuery struct {
	MailboxID  uint64 `form:"mailbox_id"`
	Folder     string `form:"folder"`
	Keyword    string `form:"keyword"`
	IsRead     *bool  `form:"is_read"`
	NeedsReply *bool  `form:"needs_reply"`
	SortBy     string `form:"sort_by"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

type SendEmailRequest struct {
	MailboxID    uint64              `json:"mailbox_id"`
	To           []basedto.PersonRef `json:"to" binding:"required"`
	CC           []basedto.PersonRef `json:"cc"`
	BCC          []basedto.PersonRef `json:"bcc"`
	Subject      string              `json:"subject" binding:"required"`
	BodyHTML     string              `json:"body_html" binding:"required"`
	TemplateID   string              `json:"template_id"`
	TemplateVars map[string]any      `json:"template_vars"`
}

type TemplateVariableInput struct {
	VariableName  string `json:"variable_name" binding:"required"`
	VariableLabel string `json:"variable_label"`
	VariableType  string `json:"variable_type"`
	IsRequired    bool   `json:"is_required"`
	DefaultValue  string `json:"default_value"`
}

type TemplateChatAssistRequest struct {
	Scene             string                  `json:"scene"`
	Goal              string                  `json:"goal" binding:"required"`
	CurrentSummary    string                  `json:"current_summary"`
	LatestUserMessage string                  `json:"latest_user_message" binding:"required"`
	KnownFields       []TemplateVariableInput `json:"known_fields"`
}

type TemplateImportAssistRequest struct {
	FileName           string `json:"file_name" binding:"required"`
	Scene              string `json:"scene"`
	DOCXTemplateBase64 string `json:"docx_template_base64" binding:"required"`
}

type TemplateRenderAssistInput struct {
	TemplateName  string         `json:"template_name"`
	TemplateScene string         `json:"template_scene"`
	RecipientType string         `json:"recipient_type"`
	BusinessGoal  string         `json:"business_goal" binding:"required"`
	KnownValues   map[string]any `json:"known_values"`
}

type TemplateAssistResponse struct {
	Summary           string                  `json:"summary"`
	FollowUpQuestions []string                `json:"follow_up_questions"`
	SuggestedFields   []TemplateVariableInput `json:"suggested_fields"`
	SubjectTemplate   string                  `json:"subject_template"`
	HTMLTemplate      string                  `json:"html_template"`
	AISource          string                  `json:"ai_source"`
	AISourceMessage   string                  `json:"ai_source_message"`
	PlainText         string                  `json:"plain_text,omitempty"`
	Paragraphs        []string                `json:"paragraphs,omitempty"`
}

type TemplateRequest struct {
	TemplateCode       string                  `json:"template_code"`
	Name               string                  `json:"name" binding:"required"`
	Category           string                  `json:"category" binding:"required"`
	Scene              string                  `json:"scene"`
	UsageBoundary      string                  `json:"usage_boundary"`
	AISessionID        string                  `json:"ai_session_id"`
	ImportFileName     string                  `json:"import_file_name"`
	ParseStatus        string                  `json:"parse_status"`
	SubjectTemplate    string                  `json:"subject_template"`
	HTMLContent        string                  `json:"html_content" binding:"required"`
	DOCXTemplateName   string                  `json:"docx_template_name"`
	DOCXTemplateBase64 string                  `json:"docx_template_base64"`
	Variables          []TemplateVariableInput `json:"variables"`
}

type TemplateQuery struct {
	Category string `form:"category"`
	Status   string `form:"status"`
	Source   string `form:"source"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type DocumentRenderRequest struct {
	TemplateID      string              `json:"template_id" binding:"required"`
	RecipientName   string              `json:"recipient_name"`
	RecipientEmail  string              `json:"recipient_email"`
	CCRecipients    []basedto.PersonRef `json:"cc_recipients"`
	SenderMailboxID uint64              `json:"sender_mailbox_id"`
	Variables       map[string]any      `json:"variables"`
	Title           string              `json:"title"`
	HTMLContent     string              `json:"html_content"`
}

type DocumentCreateRequest struct {
	DocumentRenderRequest
	VisibleUserIDs []string `json:"visible_user_ids"`
	DocumentType   string   `json:"document_type"`
}

type DocumentQuery struct {
	Status       string `form:"status"`
	DocumentType string `form:"document_type"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

type EmailView struct {
	EmailID        string              `json:"email_id"`
	MailboxID      uint64              `json:"mailbox_id"`
	Folder         string              `json:"folder"`
	Subject        string              `json:"subject"`
	Preview        string              `json:"preview"`
	BodyHTML       string              `json:"body_html,omitempty"`
	Sender         basedto.PersonRef   `json:"sender"`
	Recipients     []basedto.PersonRef `json:"recipients"`
	CCRecipients   []basedto.PersonRef `json:"cc_recipients"`
	BCCRecipients  []basedto.PersonRef `json:"bcc_recipients"`
	Summary        string              `json:"summary"`
	ReplyDraft     string              `json:"reply_draft"`
	PriorityLabel  string              `json:"priority_label"`
	PriorityScore  int                 `json:"priority_score"`
	NeedsReply     bool                `json:"needs_reply"`
	IsRead         bool                `json:"is_read"`
	IsStarred      bool                `json:"is_starred"`
	HasAttachments bool                `json:"has_attachments"`
	Status         string              `json:"status"`
	ReceivedAt     string              `json:"received_at,omitempty"`
	SentAt         string              `json:"sent_at,omitempty"`
}

type MailboxView struct {
	MailboxID           uint64 `json:"mailbox_id"`
	MailboxCode         string `json:"mailbox_code"`
	Name                string `json:"name"`
	MailboxType         string `json:"mailbox_type"`
	EmailAddress        string `json:"email_address"`
	AuthType            string `json:"auth_type"`
	SMTPHost            string `json:"smtp_host"`
	SMTPPort            int    `json:"smtp_port"`
	IMAPHost            string `json:"imap_host"`
	IMAPPort            int    `json:"imap_port"`
	Username            string `json:"username"`
	OAuthProvider       string `json:"oauth_provider"`
	SignatureHTML       string `json:"signature_html"`
	IsDefaultSender     bool   `json:"is_default_sender"`
	Status              string `json:"status"`
	LastSyncAt          string `json:"last_sync_at,omitempty"`
	LastSyncStatus      string `json:"last_sync_status"`
	LastSyncMessage     string `json:"last_sync_message"`
	LastIMAPUID         uint64 `json:"last_imap_uid"`
	LastIMAPUIDValidity uint64 `json:"last_imap_uid_validity"`
	OAuthConnected      bool   `json:"oauth_connected"`
}

type TemplateVersionView struct {
	VersionID          string                  `json:"version_id"`
	VersionNo          int                     `json:"version_no"`
	SubjectTemplate    string                  `json:"subject_template"`
	HTMLContent        string                  `json:"html_content"`
	DOCXTemplateName   string                  `json:"docx_template_name"`
	DOCXTemplateBase64 string                  `json:"docx_template_base64,omitempty"`
	Status             string                  `json:"status"`
	ApprovalID         string                  `json:"approval_id"`
	SubmittedAt        string                  `json:"submitted_at,omitempty"`
	PublishedAt        string                  `json:"published_at,omitempty"`
	Variables          []TemplateVariableInput `json:"variables"`
}

type TemplateView struct {
	TemplateID         string                  `json:"template_id"`
	TemplateCode       string                  `json:"template_code"`
	Name               string                  `json:"name"`
	Category           string                  `json:"category"`
	Scene              string                  `json:"scene"`
	UsageBoundary      string                  `json:"usage_boundary"`
	TemplateSource     string                  `json:"template_source"`
	OwnerUserID        string                  `json:"owner_user_id,omitempty"`
	SourceRefID        string                  `json:"source_ref_id,omitempty"`
	IsEditable         bool                    `json:"is_editable"`
	AISessionID        string                  `json:"ai_session_id,omitempty"`
	ImportFileName     string                  `json:"import_file_name,omitempty"`
	ParseStatus        string                  `json:"parse_status"`
	Status             string                  `json:"status"`
	PublishedVersionID string                  `json:"published_version_id,omitempty"`
	CurrentVersion     *TemplateVersionView    `json:"current_version,omitempty"`
	Variables          []TemplateVariableInput `json:"variables"`
}

type DocumentView struct {
	DocumentID         string              `json:"document_id"`
	TemplateID         string              `json:"template_id"`
	Title              string              `json:"title"`
	DocumentType       string              `json:"document_type"`
	RecipientName      string              `json:"recipient_name"`
	RecipientEmail     string              `json:"recipient_email"`
	CCRecipients       []basedto.PersonRef `json:"cc_recipients"`
	SenderMailboxID    uint64              `json:"sender_mailbox_id,omitempty"`
	SenderMailboxName  string              `json:"sender_mailbox_name,omitempty"`
	SenderMailboxEmail string              `json:"sender_mailbox_email,omitempty"`
	HTMLContent        string              `json:"html_content"`
	DOCXFileName       string              `json:"docx_file_name"`
	DOCXBase64         string              `json:"docx_base64,omitempty"`
	Status             string              `json:"status"`
	ApprovalID         string              `json:"approval_id"`
	SubmittedByUserID  string              `json:"submitted_by_user_id"`
	Variables          map[string]any      `json:"variables"`
	SendResult         string              `json:"send_result,omitempty"`
	SentAt             string              `json:"sent_at,omitempty"`
}

type LPRecipientView struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type DailySummaryView struct {
	Date        string      `json:"date"`
	Total       int         `json:"total"`
	Unread      int         `json:"unread"`
	NeedsReply  int         `json:"needs_reply"`
	TopPriority []EmailView `json:"top_priority"`
	DailyBrief  string      `json:"daily_brief"`
}

func SeedDemoData(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&MailboxAccount{}).Where("mailbox_code = ?", "shared_hq").Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			mailbox := MailboxAccount{
				MailboxCode:     "shared_hq",
				Name:            "鍩洪噾杩愯惀鍏变韩閭",
				MailboxType:     "shared",
				EmailAddress:    "ops@fintechbase.local",
				AuthType:        "password",
				SMTPHost:        "smtp.example.com",
				SMTPPort:        587,
				IMAPHost:        "imap.example.com",
				IMAPPort:        993,
				Username:        "ops@fintechbase.local",
				Status:          "active",
				IsDefaultSender: true,
			}
			if err := tx.Create(&mailbox).Error; err != nil {
				return err
			}
			now := time.Now()
			score, label, needsReply, summary, draft := classifyEmailPriority("【重要】LP季度沟通材料确认", "<p>请今日内确认发送版本并回复。</p>", []string{"LP", "季度汇报"})
			email := Email{
				EmailCode:     "MAIL-" + uuid.NewString()[:8],
				MailboxID:     mailbox.ID,
				Direction:     "inbound",
				Folder:        "inbox",
				Subject:       "【重要】LP季度沟通材料确认",
				Preview:       "请今日内确认发送版本并回复。",
				BodyHTML:      "<p>璇蜂粖澶╁唴纭鍙戦€佺増鏈苟鍥炲銆?/p>",
				SenderName:    "寮犳晱",
				SenderEmail:   "lp@example.com",
				ToRecipients:  toRecipients([]basedto.PersonRef{{Name: "鍩洪噾杩愯惀鍏变韩閭", Email: mailbox.EmailAddress}}),
				Summary:       summary,
				ReplyDraft:    draft,
				PriorityScore: score,
				PriorityLabel: label,
				NeedsReply:    needsReply,
				IsRead:        false,
				Status:        "ready",
				ReceivedAt:    &now,
			}
			if err := tx.Create(&email).Error; err != nil {
				return err
			}

			template := Template{
				TemplateCode:  "LP_NOTICE_Q",
				Name:          "LP瀛ｅ害娌熼€氶€氱煡",
				Category:      "notice",
				Scene:         "lp_communication",
				Status:        "published",
				CreatorUserID: "1",
				CreatorName:   "admin",
			}
			if err := tx.Create(&template).Error; err != nil {
				return err
			}
			for _, variable := range []TemplateVariable{
				{TemplateID: template.ID, VariableName: "lp_name", VariableLabel: "LP鍚嶇О", VariableType: "text", IsRequired: true},
				{TemplateID: template.ID, VariableName: "fund_name", VariableLabel: "鍩洪噾鍚嶇О", VariableType: "text", IsRequired: true},
				{TemplateID: template.ID, VariableName: "notice_date", VariableLabel: "閫氱煡鏃ユ湡", VariableType: "date", IsRequired: true},
			} {
				if err := tx.Create(&variable).Error; err != nil {
					return err
				}
			}
			docx := sampleDOCXTemplate()
			version := TemplateVersion{
				TemplateID:       template.ID,
				VersionNo:        1,
				SubjectTemplate:  "{{fund_name}}瀛ｅ害娌熼€氶€氱煡",
				HTMLContent:      "<h1>{{fund_name}}瀛ｅ害娌熼€氶€氱煡</h1><p>灏婃暚鐨剓{lp_name}}锛屾偍濂姐€?/p><p>閫氱煡鏃ユ湡锛歿{notice_date}}</p>",
				DOCXTemplateName: "lp-quarterly-notice.docx",
				DOCXTemplateData: docx,
				Status:           "published",
			}
			publishedAt := time.Now()
			version.PublishedAt = &publishedAt
			if err := tx.Create(&version).Error; err != nil {
				return err
			}
			if err := tx.Model(&template).Update("published_version_id", version.ID).Error; err != nil {
				return err
			}
		}
		if err := seedBuiltinTemplates(tx); err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) ListMailboxes(ctx context.Context, userID string, roles []string) ([]MailboxView, error) {
	var boxes []MailboxAccount
	db := s.db.WithContext(ctx).Where("is_deleted = false").Order("is_default_sender desc, create_time desc")
	if !roleIsInternal(roles) {
		db = db.Where("owner_user_id IS NOT NULL AND owner_user_id::text = ?", userID)
	}
	if err := db.Find(&boxes).Error; err != nil {
		return nil, err
	}
	authorized := map[uint64]bool{}
	var mailboxIDs []uint64
	for _, box := range boxes {
		if strings.EqualFold(box.OAuthProvider, "microsoft_graph") {
			mailboxIDs = append(mailboxIDs, box.ID)
		}
	}
	if len(mailboxIDs) > 0 {
		var authorizedIDs []uint64
		if err := s.db.WithContext(ctx).Model(&MailboxOAuthToken{}).Where("is_deleted = false AND mailbox_id IN ?", mailboxIDs).Pluck("mailbox_id", &authorizedIDs).Error; err != nil {
			return nil, err
		}
		for _, id := range authorizedIDs {
			authorized[id] = true
		}
	}
	items := make([]MailboxView, 0, len(boxes))
	for _, item := range boxes {
		view := toMailboxView(item)
		view.OAuthConnected = authorized[item.ID]
		items = append(items, view)
	}
	return items, nil
}

func (s *Service) SaveMailbox(ctx context.Context, mailboxID string, req MailboxRequest, userID string) (MailboxView, error) {
	returnValue := MailboxView{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		box := MailboxAccount{}
		if mailboxID != "" {
			id, err := strconv.ParseUint(mailboxID, 10, 64)
			if err != nil {
				return err
			}
			if err := tx.Where("id = ? AND is_deleted = false", id).First(&box).Error; err != nil {
				return err
			}
		} else {
			box = MailboxAccount{
				MailboxCode: "MBX-" + uuid.NewString()[:8],
				Status:      "active",
			}
		}
		if req.SMTPPort == 0 {
			req.SMTPPort = 587
		}
		if req.IMAPPort == 0 {
			req.IMAPPort = 993
		}
		box.Name = req.Name
		box.MailboxType = req.MailboxType
		box.EmailAddress = req.EmailAddress
		box.AuthType = req.AuthType
		box.SMTPHost = req.SMTPHost
		box.SMTPPort = req.SMTPPort
		box.IMAPHost = req.IMAPHost
		box.IMAPPort = req.IMAPPort
		box.Username = req.Username
		box.PasswordSecret = req.Password
		box.OAuthProvider = req.OAuthProvider
		box.SignatureHTML = req.SignatureHTML
		box.IsDefaultSender = req.IsDefaultSender
		if req.MailboxType == "personal" {
			if parsed, err := strconv.ParseUint(userID, 10, 64); err == nil {
				box.OwnerUserID = &parsed
			}
		} else {
			box.OwnerUserID = nil
		}
		if box.ID == 0 {
			if err := tx.Create(&box).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(map[string]any{
				"name":              box.Name,
				"mailbox_type":      box.MailboxType,
				"email_address":     box.EmailAddress,
				"auth_type":         box.AuthType,
				"smtp_host":         box.SMTPHost,
				"smtp_port":         box.SMTPPort,
				"imap_host":         box.IMAPHost,
				"imap_port":         box.IMAPPort,
				"username":          box.Username,
				"password_secret":   box.PasswordSecret,
				"oauth_provider":    box.OAuthProvider,
				"signature_html":    box.SignatureHTML,
				"is_default_sender": box.IsDefaultSender,
				"owner_user_id":     box.OwnerUserID,
				"is_deleted":        false,
			}).Error; err != nil {
				return err
			}
		}
		returnValue = toMailboxView(box)
		return nil
	})
	return returnValue, err
}

func (s *Service) DeleteMailbox(ctx context.Context, mailboxID string) error {
	id, err := strconv.ParseUint(mailboxID, 10, 64)
	if err != nil || id == 0 {
		return fmt.Errorf("invalid mailbox id")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&MailboxAccount{}).Where("id = ? AND is_deleted = false", id).Update("is_deleted", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Model(&MailboxOAuthToken{}).Where("mailbox_id = ?", id).Update("is_deleted", true).Error; err != nil {
			return err
		}
		return tx.Model(&Email{}).Where("mailbox_id = ? AND is_deleted = false", id).Update("is_deleted", true).Error
	})
}

func (s *Service) SaveOAuthToken(ctx context.Context, mailboxID string, req OAuthCallbackRequest) error {
	id, err := strconv.ParseUint(mailboxID, 10, 64)
	if err != nil {
		return err
	}
	expiry := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token MailboxOAuthToken
		err := tx.Where("mailbox_id = ?", id).First(&token).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			token = MailboxOAuthToken{
				MailboxID:    id,
				AccessToken:  req.AccessToken,
				RefreshToken: req.RefreshToken,
				TokenType:    req.TokenType,
				Scope:        req.Scope,
				ExpiryAt:     &expiry,
			}
			return tx.Create(&token).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&MailboxOAuthToken{}).Where("mailbox_id = ?", id).Updates(map[string]any{
			"access_token":  req.AccessToken,
			"refresh_token": req.RefreshToken,
			"token_type":    req.TokenType,
			"scope":         req.Scope,
			"expiry_at":     &expiry,
		}).Error
	})
}

func (s *Service) TestMailbox(ctx context.Context, mailboxID string) (basedto.JSONMap, error) {
	box, err := s.findMailbox(ctx, mailboxID)
	if err != nil {
		return nil, err
	}
	result := basedto.JSONMap{
		"mailbox_id":     box.ID,
		"imap_reachable": false,
		"smtp_reachable": false,
	}
	if box.IMAPHost != "" && box.IMAPPort > 0 {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(box.IMAPHost, strconv.Itoa(box.IMAPPort)), 3*time.Second)
		if err == nil {
			result["imap_reachable"] = true
			_ = conn.Close()
		} else {
			result["imap_error"] = err.Error()
		}
	}
	if box.SMTPHost != "" && box.SMTPPort > 0 {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(box.SMTPHost, strconv.Itoa(box.SMTPPort)), 3*time.Second)
		if err == nil {
			result["smtp_reachable"] = true
			_ = conn.Close()
		} else {
			result["smtp_error"] = err.Error()
		}
	}
	if token, ok := result["imap_reachable"].(bool); ok && token {
		result["status"] = "ok"
	} else {
		result["status"] = "partial"
	}
	return result, nil
}

func (s *Service) ManualSync(ctx context.Context, mailboxID, actorID, actorName string) (basedto.JSONMap, error) {
	box, err := s.findMailbox(ctx, mailboxID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(box.OAuthProvider, "microsoft_graph") {
		return s.syncMailboxFromMicrosoftGraph(ctx, box, actorID, actorName, "manual")
	}
	return s.syncMailboxFromIMAPClean(ctx, box, actorID, actorName, "manual")
	/*
			now := time.Now()
			logItem := ManualSyncLog{
				MailboxID:     box.ID,
				TriggeredByID: actorID,
				TriggeredBy:   actorName,
				Status:        "success",
				Message:       "已按手动同步策略写入最新邮件样例",
				StartedAt:     now,
			}
			err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				for index, subject := range []string{
					"【紧急】函件审批节点待处理",
					"基金投后管理月报模板修订建议",
				} {
					var count int64
					if err := tx.Model(&Email{}).Where("mailbox_id = ? AND subject = ? AND direction = ?", box.ID, subject, "inbound").Count(&count).Error; err != nil {
						return err
					}
					if count > 0 {
						continue
					}
					body := "<p>这是一封手动同步导入的样例邮件，用于验证通讯管理流程。</p>"
					score, label, needsReply, summary, draft := classifyEmailPriority(subject, body, []string{"手动同步"})
					receivedAt := now.Add(-time.Duration(index) * time.Hour)
					email := Email{
						EmailCode:     "MAIL-" + uuid.NewString()[:8],
						MailboxID:     box.ID,
						Direction:     "inbound",
						Folder:        "inbox",
						Subject:       subject,
						Preview:       summarizeHTML(body),
						BodyHTML:      body,
						SenderName:    "系统同步器",
						SenderEmail:   "sync@fintechbase.local",
						ToRecipients:  toRecipients([]basedto.PersonRef{{Name: box.Name, Email: box.EmailAddress}}),
						Summary:       summary,
						ReplyDraft:    draft,
						PriorityScore: score,
						PriorityLabel: label,
						NeedsReply:    needsReply,
						IsRead:        false,
						Status:        "ready",
						ReceivedAt:    &receivedAt,
					}
					if err := tx.Create(&email).Error; err != nil {
						return err
					}
					logItem.SyncedCount++
				}
				finishedAt := time.Now()
				logItem.FinishedAt = &finishedAt
				if err := tx.Create(&logItem).Error; err != nil {
					return err
				}
				return tx.Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(map[string]any{
					"last_sync_at":      finishedAt,
					"last_sync_status":  logItem.Status,
					"last_sync_message": logItem.Message,
				}).Error
			})
			if err != nil {
				return nil, err
			}
			return basedto.JSONMap{
				"mailbox_id":   box.ID,
				"synced_count": logItem.SyncedCount,
				"status":       logItem.Status,
				"message":      logItem.Message,
			}, nil
		}

	*/
}

/*
func (s *Service) syncMailboxFromIMAP(ctx context.Context, box MailboxAccount, actorID, actorName, trigger string) (basedto.JSONMap, error) {
	now := time.Now()
	logItem := ManualSyncLog{
		MailboxID:     box.ID,
		TriggeredByID: actorID,
		TriggeredBy:   actorName,
		Status:        "success",
		Message:       "IMAP 同步完成",
		StartedAt:     now,
	}
	if trigger == "auto" {
		logItem.TriggeredByID = "system"
		logItem.TriggeredBy = "system"
	}
	items, uidValidity, err := fetchMailboxFromIMAP(ctx, box)
	if err != nil {
		finishedAt := time.Now()
		logItem.FinishedAt = &finishedAt
		logItem.Status = "failed"
		logItem.Message = truncateString(err.Error(), 255)
		_ = s.db.WithContext(ctx).Create(&logItem).Error
		_ = s.db.WithContext(ctx).Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(map[string]any{
			"last_sync_at":      finishedAt,
			"last_sync_status":  logItem.Status,
			"last_sync_message": logItem.Message,
		}).Error
		return nil, err
	}
	var maxUID uint64
	for _, item := range items {
		if item.UID > maxUID {
			maxUID = item.UID
		}
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			created, err := s.createInboundEmailFromIMAP(ctx, tx, box, item, actorID, actorName)
			if err != nil {
				return err
			}
			if created {
				logItem.SyncedCount++
			}
		}
		finishedAt := time.Now()
		logItem.FinishedAt = &finishedAt
		logItem.Message = fmt.Sprintf("IMAP 同步完成，新增 %d 封邮件", logItem.SyncedCount)
		logItem.Message = fmt.Sprintf("IMAP sync completed, %d new email(s)", logItem.SyncedCount)
		if err := tx.Create(&logItem).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"last_sync_at":      finishedAt,
			"last_sync_status":  logItem.Status,
			"last_sync_message": logItem.Message,
		}
		if uidValidity > 0 {
			updates["last_imap_uid_validity"] = uidValidity
		}
		if maxUID > 0 {
			updates["last_imap_uid"] = maxUID
		}
		return tx.Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return basedto.JSONMap{
		"mailbox_id":   box.ID,
		"synced_count": logItem.SyncedCount,
		"status":       logItem.Status,
		"message":      logItem.Message,
	}, nil
}

}
*/

func (s *Service) syncMailboxFromIMAP(ctx context.Context, box MailboxAccount, actorID, actorName, trigger string) (basedto.JSONMap, error) {
	now := time.Now()
	logItem := ManualSyncLog{
		MailboxID:     box.ID,
		TriggeredByID: actorID,
		TriggeredBy:   actorName,
		Status:        "success",
		Message:       "IMAP 同步完成",
		StartedAt:     now,
	}
	if trigger == "auto" {
		logItem.TriggeredByID = "system"
		logItem.TriggeredBy = "system"
	}

	items, uidValidity, err := fetchMailboxFromIMAP(ctx, box)
	if err != nil {
		finishedAt := time.Now()
		logItem.FinishedAt = &finishedAt
		logItem.Status = "failed"
		logItem.Message = truncateString(err.Error(), 255)
		_ = s.db.WithContext(ctx).Create(&logItem).Error
		_ = s.db.WithContext(ctx).Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(map[string]any{
			"last_sync_at":      finishedAt,
			"last_sync_status":  logItem.Status,
			"last_sync_message": logItem.Message,
		}).Error
		return nil, err
	}

	var maxUID uint64
	for _, item := range items {
		if item.UID > maxUID {
			maxUID = item.UID
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			created, err := s.createInboundEmailFromIMAP(ctx, tx, box, item, actorID, actorName)
			if err != nil {
				return err
			}
			if created {
				logItem.SyncedCount++
			}
		}
		finishedAt := time.Now()
		logItem.FinishedAt = &finishedAt
		logItem.Message = fmt.Sprintf("IMAP 同步完成，新增 %d 封邮件", logItem.SyncedCount)
		if err := tx.Create(&logItem).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"last_sync_at":      finishedAt,
			"last_sync_status":  logItem.Status,
			"last_sync_message": logItem.Message,
		}
		if uidValidity > 0 {
			updates["last_imap_uid_validity"] = uidValidity
		}
		if maxUID > 0 {
			updates["last_imap_uid"] = maxUID
		}
		return tx.Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return basedto.JSONMap{
		"mailbox_id":   box.ID,
		"synced_count": logItem.SyncedCount,
		"status":       logItem.Status,
		"message":      logItem.Message,
	}, nil
}

func (s *Service) syncMailboxFromIMAPClean(ctx context.Context, box MailboxAccount, actorID, actorName, trigger string) (basedto.JSONMap, error) {
	now := time.Now()
	logItem := ManualSyncLog{
		MailboxID:     box.ID,
		TriggeredByID: actorID,
		TriggeredBy:   actorName,
		Status:        "success",
		Message:       "IMAP sync completed",
		StartedAt:     now,
	}
	if trigger == "auto" {
		logItem.TriggeredByID = "system"
		logItem.TriggeredBy = "system"
	}

	items, uidValidity, err := fetchMailboxFromIMAP(ctx, box)
	if err != nil {
		finishedAt := time.Now()
		logItem.FinishedAt = &finishedAt
		logItem.Status = "failed"
		logItem.Message = truncateString(err.Error(), 255)
		_ = s.db.WithContext(ctx).Create(&logItem).Error
		_ = s.db.WithContext(ctx).Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(map[string]any{
			"last_sync_at":      finishedAt,
			"last_sync_status":  logItem.Status,
			"last_sync_message": logItem.Message,
		}).Error
		return nil, err
	}

	var maxUID uint64
	for _, item := range items {
		if item.UID > maxUID {
			maxUID = item.UID
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			created, err := s.createInboundEmailFromIMAP(ctx, tx, box, item, actorID, actorName)
			if err != nil {
				return err
			}
			if created {
				logItem.SyncedCount++
			}
		}
		finishedAt := time.Now()
		logItem.FinishedAt = &finishedAt
		logItem.Message = fmt.Sprintf("IMAP sync completed, %d new email(s)", logItem.SyncedCount)
		if err := tx.Create(&logItem).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"last_sync_at":      finishedAt,
			"last_sync_status":  logItem.Status,
			"last_sync_message": logItem.Message,
		}
		if uidValidity > 0 {
			updates["last_imap_uid_validity"] = uidValidity
		}
		if maxUID > 0 {
			updates["last_imap_uid"] = maxUID
		}
		return tx.Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return basedto.JSONMap{
		"mailbox_id":   box.ID,
		"synced_count": logItem.SyncedCount,
		"status":       logItem.Status,
		"message":      logItem.Message,
	}, nil
}

func (s *Service) createInboundEmailFromIMAP(ctx context.Context, tx *gorm.DB, box MailboxAccount, item imapFetchedMessage, actorID, actorName string) (bool, error) {
	parsed, err := parseInboundMessage(item)
	if err != nil {
		return false, err
	}
	var count int64
	if item.UID > 0 && item.UIDValidity > 0 {
		if err := tx.WithContext(ctx).Model(&Email{}).
			Where("mailbox_id = ? AND imap_uid_validity = ? AND imap_uid = ? AND direction = ? AND is_deleted = false", box.ID, item.UIDValidity, item.UID, "inbound").
			Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return false, nil
		}
	}
	if count == 0 && parsed.MessageID != "" {
		if err := tx.WithContext(ctx).Model(&Email{}).
			Where("mailbox_id = ? AND message_id = ? AND direction = ? AND is_deleted = false", box.ID, parsed.MessageID, "inbound").
			Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return false, nil
		}
	}
	score, label, needsReply, summary, draft := classifyEmailPriority(parsed.Subject, parsed.BodyHTML, []string{"imap"})
	email := Email{
		EmailCode:        "MAIL-" + uuid.NewString()[:8],
		MailboxID:        box.ID,
		Direction:        "inbound",
		Folder:           "inbox",
		MessageID:        cleanUTF8(parsed.MessageID),
		IMAPUID:          item.UID,
		IMAPUIDValidity:  item.UIDValidity,
		ThreadID:         cleanUTF8(parsed.ThreadID),
		Subject:          cleanUTF8(parsed.Subject),
		Preview:          summarizeHTML(parsed.BodyHTML),
		BodyHTML:         cleanUTF8(parsed.BodyHTML),
		SenderName:       cleanUTF8(parsed.SenderName),
		SenderEmail:      cleanUTF8(parsed.SenderEmail),
		ToRecipients:     toRecipients(cleanPersonRefs(parsed.To)),
		CcRecipients:     toRecipients(cleanPersonRefs(parsed.CC)),
		Summary:          summary,
		ReplyDraft:       draft,
		PriorityScore:    score,
		PriorityLabel:    label,
		NeedsReply:       needsReply,
		IsRead:           parsed.IsRead,
		HasAttachments:   len(parsed.Attachments) > 0,
		Status:           "ready",
		OperatorUserID:   cleanUTF8(actorID),
		OperatorUsername: cleanUTF8(actorName),
		ReceivedAt:       parsed.ReceivedAt,
	}
	if err := tx.WithContext(ctx).Create(&email).Error; err != nil {
		return false, err
	}
	for _, attachment := range parsed.Attachments {
		if err := tx.WithContext(ctx).Create(&EmailAttachment{
			EmailID:     email.ID,
			FileName:    cleanUTF8(attachment.FileName),
			ContentType: cleanUTF8(attachment.ContentType),
			FileSize:    int64(len(attachment.Data)),
			FileData:    attachment.Data,
		}).Error; err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Service) ListEmails(ctx context.Context, query EmailListQuery, userID string, roles []string) (basedto.PageResult[EmailView], error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
	db := s.db.WithContext(ctx).Model(&Email{}).Where("is_deleted = false")
	if query.MailboxID > 0 {
		db = db.Where("mailbox_id = ?", query.MailboxID)
	}
	if query.Folder != "" {
		db = db.Where("folder = ?", query.Folder)
	}
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		db = db.Where("subject ILIKE ? OR preview ILIKE ? OR sender_email ILIKE ?", keyword, keyword, keyword)
	}
	if query.IsRead != nil {
		db = db.Where("is_read = ?", *query.IsRead)
	}
	if query.NeedsReply != nil {
		db = db.Where("needs_reply = ?", *query.NeedsReply)
	}
	if !roleIsInternal(roles) {
		var boxes []uint64
		if err := s.db.WithContext(ctx).Model(&MailboxAccount{}).Where("owner_user_id::text = ? AND is_deleted = false", userID).Pluck("id", &boxes).Error; err != nil {
			return basedto.PageResult[EmailView]{}, err
		}
		if len(boxes) == 0 {
			return basedto.PageResult[EmailView]{List: []EmailView{}, Pagination: basedto.PaginationInfo{Page: query.Page, PageSize: query.PageSize, Total: 0}}, nil
		}
		db = db.Where("mailbox_id IN ?", boxes)
	}
	sortClause := "priority_score desc, received_at desc"
	if query.SortBy == "sender" {
		sortClause = "sender_email asc, priority_score desc"
	}
	if query.SortBy == "subject" {
		sortClause = "subject asc, priority_score desc"
	}
	if query.SortBy == "received_at" {
		sortClause = "received_at desc nulls last, imap_uid desc"
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return basedto.PageResult[EmailView]{}, err
	}
	var rows []Email
	if err := db.Order(sortClause).Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return basedto.PageResult[EmailView]{}, err
	}
	items := make([]EmailView, 0, len(rows))
	for _, row := range rows {
		items = append(items, toEmailView(row, false))
	}
	return basedto.PageResult[EmailView]{
		List:       items,
		Pagination: basedto.PaginationInfo{Page: query.Page, PageSize: query.PageSize, Total: total},
	}, nil
}

func (s *Service) GetEmail(ctx context.Context, emailID string) (EmailView, error) {
	var email Email
	if err := s.db.WithContext(ctx).Where("email_code = ? AND is_deleted = false", emailID).First(&email).Error; err != nil {
		return EmailView{}, err
	}
	return toEmailView(email, true), nil
}

func (s *Service) DeleteEmail(ctx context.Context, emailID string) error {
	result := s.db.WithContext(ctx).Model(&Email{}).Where("email_code = ? AND is_deleted = false", emailID).Update("is_deleted", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Service) SendEmail(ctx context.Context, req SendEmailRequest, actorID, actorName string) (basedto.JSONMap, error) {
	box, err := s.resolveSendMailbox(ctx, req.MailboxID)
	if err != nil {
		return nil, err
	}
	subject := req.Subject
	body := req.BodyHTML
	if req.TemplateID != "" {
		rendered, err := s.RenderDocument(ctx, DocumentRenderRequest{
			TemplateID:     req.TemplateID,
			RecipientName:  firstPersonName(req.To),
			RecipientEmail: firstPersonEmail(req.To),
			Variables:      req.TemplateVars,
			Title:          req.Subject,
		})
		if err != nil {
			return nil, err
		}
		subject = rendered.Title
		body = rendered.HTMLContent
	}
	if err := s.sendOutbound(ctx, box, outboundMessage{
		FromName:    box.Name,
		FromAddress: box.EmailAddress,
		To:          req.To,
		CC:          req.CC,
		BCC:         req.BCC,
		Subject:     subject,
		HTMLBody:    withMailboxSignature(body, box.SignatureHTML),
	}); err != nil {
		return nil, err
	}
	score, label, needsReply, summary, draft := classifyEmailPriority(subject, body, []string{"outbound"})
	now := time.Now()
	email := Email{
		EmailCode:        "MAIL-" + uuid.NewString()[:8],
		MailboxID:        box.ID,
		Direction:        "outbound",
		Folder:           "sent",
		Subject:          subject,
		Preview:          summarizeHTML(body),
		BodyHTML:         withMailboxSignature(body, box.SignatureHTML),
		SenderName:       box.Name,
		SenderEmail:      box.EmailAddress,
		ToRecipients:     toRecipients(req.To),
		CcRecipients:     toRecipients(req.CC),
		BccRecipients:    toRecipients(req.BCC),
		Summary:          summary,
		ReplyDraft:       draft,
		PriorityScore:    score,
		PriorityLabel:    label,
		NeedsReply:       needsReply,
		IsRead:           true,
		Status:           "sent",
		OperatorUserID:   actorID,
		OperatorUsername: actorName,
		SentAt:           &now,
	}
	if err := s.db.WithContext(ctx).Create(&email).Error; err != nil {
		return nil, err
	}
	return basedto.JSONMap{"status": "sent", "email_id": email.EmailCode}, nil
}

func (s *Service) UpdateEmailFlag(ctx context.Context, emailID string, field string, value bool) error {
	updates := map[string]any{field: value}
	return s.db.WithContext(ctx).Model(&Email{}).Where("email_code = ? AND is_deleted = false", emailID).Updates(updates).Error
}

func (s *Service) ToggleEmailFlag(ctx context.Context, emailID string, field string) (bool, error) {
	row := Email{}
	if err := s.db.WithContext(ctx).Where("email_code = ? AND is_deleted = false", emailID).First(&row).Error; err != nil {
		return false, err
	}

	var next bool
	switch field {
	case "is_starred":
		next = !row.IsStarred
	case "is_read":
		next = !row.IsRead
	default:
		return false, fmt.Errorf("unsupported email flag: %s", field)
	}

	updates := map[string]any{field: next}
	if err := s.db.WithContext(ctx).Model(&Email{}).Where("email_code = ? AND is_deleted = false", emailID).Updates(updates).Error; err != nil {
		return false, err
	}
	return next, nil
}

func (s *Service) DailySummary(ctx context.Context, mailboxID uint64, userID string, roles []string) (DailySummaryView, error) {
	db := s.db.WithContext(ctx).Model(&Email{}).Where("is_deleted = false")
	if mailboxID > 0 {
		db = db.Where("mailbox_id = ?", mailboxID)
	}
	if !roleIsInternal(roles) {
		var boxes []uint64
		if err := s.db.WithContext(ctx).Model(&MailboxAccount{}).Where("owner_user_id::text = ? AND is_deleted = false", userID).Pluck("id", &boxes).Error; err != nil {
			return DailySummaryView{}, err
		}
		if len(boxes) == 0 {
			return DailySummaryView{
				Date:        time.Now().Format("2006-01-02"),
				TopPriority: []EmailView{},
				DailyBrief:  "今日通讯中心暂无邮件。",
			}, nil
		}
		db = db.Where("mailbox_id IN ?", boxes)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return DailySummaryView{}, err
	}

	var unread int64
	if err := db.Session(&gorm.Session{}).Where("is_read = ?", false).Count(&unread).Error; err != nil {
		return DailySummaryView{}, err
	}

	var needsReply int64
	if err := db.Session(&gorm.Session{}).Where("needs_reply = ?", true).Count(&needsReply).Error; err != nil {
		return DailySummaryView{}, err
	}

	var systemRows []Email
	if err := db.Session(&gorm.Session{}).Order("priority_score desc, received_at desc").Limit(5).Find(&systemRows).Error; err != nil {
		return DailySummaryView{}, err
	}

	var starredRows []Email
	if err := db.Session(&gorm.Session{}).Where("is_starred = ?", true).Order("received_at desc").Limit(5).Find(&starredRows).Error; err != nil {
		return DailySummaryView{}, err
	}

	priorityMap := make(map[string]Email)
	for _, row := range systemRows {
		priorityMap[row.EmailCode] = row
	}
	for _, row := range starredRows {
		priorityMap[row.EmailCode] = row
	}

	mergedRows := make([]Email, 0, len(priorityMap))
	for _, row := range priorityMap {
		mergedRows = append(mergedRows, row)
	}
	sort.SliceStable(mergedRows, func(i, j int) bool {
		if mergedRows[i].IsStarred != mergedRows[j].IsStarred {
			return mergedRows[i].IsStarred
		}
		if mergedRows[i].PriorityScore != mergedRows[j].PriorityScore {
			return mergedRows[i].PriorityScore > mergedRows[j].PriorityScore
		}
		left := time.Time{}
		right := time.Time{}
		if mergedRows[i].ReceivedAt != nil {
			left = *mergedRows[i].ReceivedAt
		}
		if mergedRows[j].ReceivedAt != nil {
			right = *mergedRows[j].ReceivedAt
		}
		return left.After(right)
	})
	if len(mergedRows) > 5 {
		mergedRows = mergedRows[:5]
	}

	topPriority := make([]EmailView, 0, len(mergedRows))
	for _, row := range mergedRows {
		topPriority = append(topPriority, toEmailView(row, false))
	}

	summary := DailySummaryView{
		Date:        time.Now().Format("2006-01-02"),
		Total:       int(total),
		Unread:      int(unread),
		NeedsReply:  int(needsReply),
		TopPriority: topPriority,
	}
	if total == 0 {
		summary.DailyBrief = "今日通讯中心暂无邮件。"
	} else {
		summary.DailyBrief = fmt.Sprintf("今日通讯中心共 %d 封邮件，其中 %d 封未读，%d 封需要回复。", summary.Total, summary.Unread, summary.NeedsReply)
	}
	return summary, nil
}
func (s *Service) ListTemplates(ctx context.Context, query TemplateQuery, actorID string) (basedto.PageResult[TemplateView], error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
	if err := s.syncTemplatePublication(ctx); err != nil {
		return basedto.PageResult[TemplateView]{}, err
	}
	db := s.db.WithContext(ctx).Model(&Template{}).Where("is_deleted = false")
	db = db.Where("(template_source = ? OR (template_source = ? AND owner_user_id = ?))", templateSourceSystemBuiltin, templateSourceUserPrivate, actorID)
	if query.Category != "" {
		db = db.Where("category = ?", query.Category)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Source != "" {
		db = db.Where("template_source = ?", query.Source)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return basedto.PageResult[TemplateView]{}, err
	}
	var rows []Template
	if err := db.Order("update_time desc").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return basedto.PageResult[TemplateView]{}, err
	}
	items := make([]TemplateView, 0, len(rows))
	for _, row := range rows {
		view, err := s.getTemplateView(ctx, row)
		if err != nil {
			return basedto.PageResult[TemplateView]{}, err
		}
		items = append(items, view)
	}
	return basedto.PageResult[TemplateView]{
		List:       items,
		Pagination: basedto.PaginationInfo{Page: query.Page, PageSize: query.PageSize, Total: total},
	}, nil
}

func (s *Service) ListLPRecipients(ctx context.Context) ([]LPRecipientView, error) {
	var rows []database.SysUser
	err := s.db.WithContext(ctx).
		Table("sys_user").
		Select("sys_user.*").
		Joins("JOIN sys_user_role sur ON sur.user_id = sys_user.id AND sur.is_deleted = false").
		Joins("JOIN sys_role sr ON sr.id = sur.role_id AND sr.is_deleted = false").
		Where("sys_user.is_deleted = false AND sys_user.status = ? AND sr.role_code = ?", "active", "LP").
		Order("sys_user.create_time desc").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]LPRecipientView, 0, len(rows))
	for _, row := range rows {
		items = append(items, LPRecipientView{
			UserID:   strconv.FormatUint(row.ID, 10),
			Name:     defaultString(row.RealName, row.Username),
			Email:    row.Email,
			Username: row.Username,
		})
	}
	return items, nil
}

func (s *Service) DraftTemplateFromChat(ctx context.Context, req TemplateChatAssistRequest) (TemplateAssistResponse, error) {
	if s.aiClient == nil {
		return fallbackTemplateChatAssist(req), nil
	}
	response, err := s.aiClient.DraftTemplateFromChat(ctx, baseai.TemplateChatRequest{
		Scene:             req.Scene,
		Goal:              req.Goal,
		CurrentSummary:    req.CurrentSummary,
		LatestUserMessage: req.LatestUserMessage,
		KnownFields:       toAIKnownFields(req.KnownFields),
	})
	if err != nil {
		return fallbackTemplateChatAssist(req), nil
	}
	return toTemplateAssistResponse(response, "", nil), nil
}

func (s *Service) DraftTemplateFromImport(ctx context.Context, req TemplateImportAssistRequest) (TemplateAssistResponse, error) {
	if s.aiClient == nil {
		docxData, err := decodeBase64(req.DOCXTemplateBase64)
		if err != nil {
			return TemplateAssistResponse{}, err
		}
		plainText, paragraphs, err := extractDOCXText(docxData)
		if err != nil {
			return TemplateAssistResponse{}, err
		}
		return fallbackTemplateImportAssist(req, plainText, paragraphs), nil
	}
	docxData, err := decodeBase64(req.DOCXTemplateBase64)
	if err != nil {
		return TemplateAssistResponse{}, err
	}
	plainText, paragraphs, err := extractDOCXText(docxData)
	if err != nil {
		return TemplateAssistResponse{}, err
	}
	response, err := s.aiClient.DraftTemplateFromImport(ctx, baseai.TemplateImportRequest{
		FileName:   req.FileName,
		PlainText:  plainText,
		Paragraphs: paragraphs,
		Scene:      req.Scene,
	})
	if err != nil {
		return fallbackTemplateImportAssist(req, plainText, paragraphs), nil
	}
	return toTemplateAssistResponse(response, plainText, paragraphs), nil
}

func (s *Service) AssistTemplateRender(ctx context.Context, req TemplateRenderAssistInput) (TemplateAssistResponse, error) {
	if s.aiClient == nil {
		return fallbackTemplateRenderAssist(req), nil
	}
	response, err := s.aiClient.RenderTemplateAssist(ctx, baseai.TemplateRenderAssistRequest{
		TemplateName:  req.TemplateName,
		TemplateScene: req.TemplateScene,
		RecipientType: req.RecipientType,
		BusinessGoal:  req.BusinessGoal,
		KnownValues:   req.KnownValues,
	})
	if err != nil {
		return fallbackTemplateRenderAssist(req), nil
	}
	return toTemplateAssistResponse(response, "", nil), nil
}

func (s *Service) SaveTemplate(ctx context.Context, templateID string, req TemplateRequest, actorID, actorName string) (TemplateView, error) {
	var output TemplateView
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		docxData, err := decodeBase64(req.DOCXTemplateBase64)
		if err != nil {
			return err
		}
		tpl := Template{}
		if templateID != "" {
			id, err := strconv.ParseUint(templateID, 10, 64)
			if err != nil {
				return err
			}
			if err := tx.Where("id = ? AND is_deleted = false", id).First(&tpl).Error; err != nil {
				return err
			}
			if tpl.TemplateSource == templateSourceSystemBuiltin {
				return fmt.Errorf("system builtin template %s is read-only", tpl.TemplateCode)
			}
			if tpl.TemplateSource == templateSourcePublishedPublic {
				return fmt.Errorf("published public template %s cannot be edited directly; create a private copy first", tpl.TemplateCode)
			}
			if strings.TrimSpace(tpl.OwnerUserID) != strings.TrimSpace(actorID) {
				return fmt.Errorf("template %s does not belong to current user", tpl.TemplateCode)
			}
		} else {
			code := strings.TrimSpace(req.TemplateCode)
			if code == "" {
				code = "TPL-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
			}
			tpl = Template{
				TemplateCode:   code,
				TemplateSource: templateSourceUserPrivate,
				OwnerUserID:    actorID,
				IsEditable:     true,
				ParseStatus:    defaultString(req.ParseStatus, "ready"),
				CreatorUserID:  actorID,
				CreatorName:    actorName,
			}
			if err := tx.Create(&tpl).Error; err != nil {
				return err
			}
		}
		tpl.Name = req.Name
		tpl.Category = req.Category
		tpl.Scene = req.Scene
		tpl.UsageBoundary = req.UsageBoundary
		tpl.AISessionID = req.AISessionID
		tpl.ImportFileName = req.ImportFileName
		tpl.ParseStatus = defaultString(req.ParseStatus, "ready")
		if err := tx.Model(&Template{}).Where("id = ?", tpl.ID).Updates(map[string]any{
			"name":             tpl.Name,
			"category":         tpl.Category,
			"scene":            tpl.Scene,
			"usage_boundary":   tpl.UsageBoundary,
			"status":           "draft",
			"template_source":  tpl.TemplateSource,
			"owner_user_id":    tpl.OwnerUserID,
			"is_editable":      tpl.IsEditable,
			"ai_session_id":    tpl.AISessionID,
			"import_file_name": tpl.ImportFileName,
			"parse_status":     tpl.ParseStatus,
			"creator_name":     actorName,
			"is_deleted":       false,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("template_id = ?", tpl.ID).Delete(&TemplateVariable{}).Error; err != nil {
			return err
		}
		variables := make([]TemplateVariable, 0, len(req.Variables))
		for _, variable := range req.Variables {
			row := TemplateVariable{
				TemplateID:    tpl.ID,
				VariableName:  variable.VariableName,
				VariableLabel: variable.VariableLabel,
				VariableType:  defaultString(variable.VariableType, "text"),
				IsRequired:    variable.IsRequired,
				DefaultValue:  variable.DefaultValue,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			variables = append(variables, row)
		}
		if err := validateVariableCoverage(variables, req.HTMLContent, docxData); err != nil {
			return err
		}
		versionNo := 1
		var lastVersion TemplateVersion
		if err := tx.Where("template_id = ?", tpl.ID).Order("version_no desc").First(&lastVersion).Error; err == nil {
			versionNo = lastVersion.VersionNo + 1
		}
		version := TemplateVersion{
			TemplateID:       tpl.ID,
			VersionNo:        versionNo,
			SubjectTemplate:  req.SubjectTemplate,
			HTMLContent:      req.HTMLContent,
			DOCXTemplateName: req.DOCXTemplateName,
			DOCXTemplateData: docxData,
			Status:           "draft",
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if err := tx.Model(&Template{}).Where("id = ?", tpl.ID).Update("published_version_id", version.ID).Error; err != nil {
			return err
		}
		output = TemplateView{
			TemplateID:     strconv.FormatUint(tpl.ID, 10),
			TemplateCode:   tpl.TemplateCode,
			Name:           tpl.Name,
			Category:       tpl.Category,
			Scene:          tpl.Scene,
			UsageBoundary:  tpl.UsageBoundary,
			TemplateSource: tpl.TemplateSource,
			OwnerUserID:    tpl.OwnerUserID,
			IsEditable:     tpl.IsEditable,
			AISessionID:    tpl.AISessionID,
			ImportFileName: tpl.ImportFileName,
			ParseStatus:    tpl.ParseStatus,
			Status:         "draft",
			Variables:      toTemplateVariableInputs(variables),
			CurrentVersion: &TemplateVersionView{
				VersionID:          strconv.FormatUint(version.ID, 10),
				VersionNo:          version.VersionNo,
				SubjectTemplate:    version.SubjectTemplate,
				HTMLContent:        version.HTMLContent,
				DOCXTemplateName:   version.DOCXTemplateName,
				DOCXTemplateBase64: encodeBase64(version.DOCXTemplateData),
				Status:             version.Status,
			},
		}
		return nil
	})
	return output, err
}

func (s *Service) GetTemplate(ctx context.Context, templateID, actorID string) (TemplateView, error) {
	if err := s.syncTemplatePublication(ctx); err != nil {
		return TemplateView{}, err
	}
	id, err := strconv.ParseUint(templateID, 10, 64)
	if err != nil {
		return TemplateView{}, err
	}
	var tpl Template
	if err := s.db.WithContext(ctx).Where("id = ? AND is_deleted = false", id).First(&tpl).Error; err != nil {
		return TemplateView{}, err
	}
	if !canAccessTemplate(tpl, actorID) {
		return TemplateView{}, gorm.ErrRecordNotFound
	}
	return s.getTemplateView(ctx, tpl)
}

func (s *Service) DeleteTemplate(ctx context.Context, templateID, actorID string) error {
	id, err := strconv.ParseUint(templateID, 10, 64)
	if err != nil {
		return err
	}
	var tpl Template
	if err := s.db.WithContext(ctx).Where("id = ? AND is_deleted = false", id).First(&tpl).Error; err != nil {
		return err
	}
	if tpl.TemplateSource != templateSourceUserPrivate {
		return fmt.Errorf("template %s cannot be deleted", tpl.TemplateCode)
	}
	if strings.TrimSpace(tpl.OwnerUserID) != strings.TrimSpace(actorID) {
		return fmt.Errorf("template %s does not belong to current user", tpl.TemplateCode)
	}
	return s.db.WithContext(ctx).Model(&Template{}).Where("id = ?", id).Updates(map[string]any{"is_deleted": true, "status": "deleted"}).Error
}

func (s *Service) ListTemplateVersions(ctx context.Context, templateID, actorID string) ([]TemplateVersionView, error) {
	id, err := strconv.ParseUint(templateID, 10, 64)
	if err != nil {
		return nil, err
	}
	var tpl Template
	if err := s.db.WithContext(ctx).Where("id = ? AND is_deleted = false", id).First(&tpl).Error; err != nil {
		return nil, err
	}
	if !canAccessTemplate(tpl, actorID) {
		return nil, gorm.ErrRecordNotFound
	}
	var versions []TemplateVersion
	if err := s.db.WithContext(ctx).Where("template_id = ? AND is_deleted = false", id).Order("version_no desc").Find(&versions).Error; err != nil {
		return nil, err
	}
	var variables []TemplateVariable
	_ = s.db.WithContext(ctx).Where("template_id = ? AND is_deleted = false", id).Find(&variables).Error
	items := make([]TemplateVersionView, 0, len(versions))
	for _, version := range versions {
		items = append(items, TemplateVersionView{
			VersionID:          strconv.FormatUint(version.ID, 10),
			VersionNo:          version.VersionNo,
			SubjectTemplate:    version.SubjectTemplate,
			HTMLContent:        version.HTMLContent,
			DOCXTemplateName:   version.DOCXTemplateName,
			DOCXTemplateBase64: encodeBase64(version.DOCXTemplateData),
			Status:             version.Status,
			ApprovalID:         version.ApprovalID,
			SubmittedAt:        formatRFC3339(version.SubmittedAt),
			PublishedAt:        formatRFC3339(version.PublishedAt),
			Variables:          toTemplateVariableInputs(variables),
		})
	}
	return items, nil
}

func (s *Service) SubmitTemplate(ctx context.Context, templateID, actorID, actorName string) (basedto.JSONMap, error) {
	id, err := strconv.ParseUint(templateID, 10, 64)
	if err != nil {
		return nil, err
	}
	var tpl Template
	if err := s.db.WithContext(ctx).Where("id = ? AND is_deleted = false", id).First(&tpl).Error; err != nil {
		return nil, err
	}
	if tpl.TemplateSource != templateSourceUserPrivate {
		return nil, fmt.Errorf("only private templates can be submitted for publication")
	}
	if strings.TrimSpace(tpl.OwnerUserID) != strings.TrimSpace(actorID) {
		return nil, fmt.Errorf("template %s does not belong to current user", tpl.TemplateCode)
	}
	var version TemplateVersion
	if err := s.db.WithContext(ctx).Where("template_id = ? AND is_deleted = false", id).Order("version_no desc").First(&version).Error; err != nil {
		return nil, err
	}
	created, err := s.approvalService.CreateApproval(ctx, approval.CreateApprovalRequest{
		BusinessType: "template_publish",
		BusinessID:   strconv.FormatUint(version.ID, 10),
		Title:        "模板发布审批：" + tpl.Name,
		Payload:      basedto.JSONMap{"template_id": tpl.ID, "template_version_id": version.ID},
	}, actorID, actorName)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&TemplateVersion{}).Where("id = ?", version.ID).Updates(map[string]any{
			"status":       "pending_approval",
			"approval_id":  created.ApprovalID,
			"submitted_at": &now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&Template{}).Where("id = ?", tpl.ID).Updates(map[string]any{
			"status": "pending_approval",
		}).Error
	}); err != nil {
		return nil, err
	}
	return basedto.JSONMap{"approval_id": created.ApprovalID, "status": "pending_approval"}, nil
}

func (s *Service) RenderDocument(ctx context.Context, req DocumentRenderRequest) (DocumentView, error) {
	templateView, version, variables, err := s.resolveTemplateForRender(ctx, req.TemplateID)
	if err != nil {
		return DocumentView{}, err
	}
	values, err := ensureVariableValues(variables, req.Variables)
	if err != nil {
		return DocumentView{}, err
	}
	htmlTemplate := defaultString(version.HTMLContent, "")
	if strings.TrimSpace(req.HTMLContent) != "" {
		htmlTemplate = req.HTMLContent
	}
	htmlContent := renderTemplateString(htmlTemplate, values)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = renderTemplateString(defaultString(version.SubjectTemplate, templateView.Name), values)
	}
	docxFileName := version.DOCXTemplateName
	docxBytes := []byte(nil)
	if hasFormalTemplateDOCX(version) {
		docxBytes, err = buildDOCXFromTemplate(version.DOCXTemplateData, values)
		if err != nil {
			return DocumentView{}, err
		}
	} else {
		docxBytes = buildHTMLDocumentDOCX(title, htmlContent)
		docxFileName = "communication-document.docx"
	}
	return DocumentView{
		TemplateID:      req.TemplateID,
		Title:           title,
		DocumentType:    templateView.Category,
		RecipientName:   req.RecipientName,
		RecipientEmail:  req.RecipientEmail,
		CCRecipients:    req.CCRecipients,
		SenderMailboxID: req.SenderMailboxID,
		HTMLContent:     htmlContent,
		DOCXFileName:    docxFileName,
		DOCXBase64:      encodeBase64(docxBytes),
		Variables:       req.Variables,
		Status:          "rendered",
	}, nil
}

func (s *Service) CreateDocument(ctx context.Context, req DocumentCreateRequest, actorID, actorName string) (DocumentView, error) {
	rendered, err := s.RenderDocument(ctx, req.DocumentRenderRequest)
	if err != nil {
		return DocumentView{}, err
	}
	templateID, _ := strconv.ParseUint(req.TemplateID, 10, 64)
	templateView, version, _, err := s.resolveTemplateForRender(ctx, req.TemplateID)
	if err != nil {
		return DocumentView{}, err
	}
	docxBytes, err := decodeBase64(rendered.DOCXBase64)
	if err != nil {
		return DocumentView{}, err
	}
	sendMailbox, err := s.resolveSendMailbox(ctx, req.SenderMailboxID)
	if err != nil {
		return DocumentView{}, err
	}
	visibleUserIDs, err := s.resolveDocumentVisibleUserIDs(ctx, req.RecipientEmail, req.VisibleUserIDs)
	if err != nil {
		return DocumentView{}, err
	}
	document := DocumentInstance{
		DocumentCode:       "DOC-" + uuid.NewString()[:8],
		TemplateID:         templateID,
		TemplateVersionID:  version.ID,
		Title:              rendered.Title,
		DocumentType:       defaultString(req.DocumentType, templateView.Category),
		RecipientName:      req.RecipientName,
		RecipientEmail:     req.RecipientEmail,
		CCRecipients:       toRecipients(req.CCRecipients),
		SenderMailboxID:    &sendMailbox.ID,
		SenderMailboxName:  sendMailbox.Name,
		SenderMailboxEmail: sendMailbox.EmailAddress,
		VisibleUserIDs:     mustJSON(visibleUserIDs, "[]"),
		VariableSnapshot:   mustJSON(req.Variables, "{}"),
		HTMLContent:        rendered.HTMLContent,
		DOCXFileName:       rendered.DOCXFileName,
		DOCXData:           docxBytes,
		Status:             "draft",
		SubmittedByUserID:  actorID,
		SubmittedByName:    actorName,
	}
	if err := s.db.WithContext(ctx).Create(&document).Error; err != nil {
		return DocumentView{}, err
	}
	rendered.DocumentID = document.DocumentCode
	rendered.Status = document.Status
	rendered.CCRecipients = req.CCRecipients
	rendered.SenderMailboxID = sendMailbox.ID
	rendered.SenderMailboxName = sendMailbox.Name
	rendered.SenderMailboxEmail = sendMailbox.EmailAddress
	return rendered, nil
}

func (s *Service) ListDocuments(ctx context.Context, query DocumentQuery, userID string, roles []string) (basedto.PageResult[DocumentView], error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
	if err := s.syncDocumentApproval(ctx); err != nil {
		return basedto.PageResult[DocumentView]{}, err
	}
	db := s.db.WithContext(ctx).Model(&DocumentInstance{}).Where("is_deleted = false")
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.DocumentType != "" {
		db = db.Where("document_type = ?", query.DocumentType)
	}
	var user database.SysUser
	if parsedUserID, err := strconv.ParseUint(strings.TrimSpace(userID), 10, 64); err == nil {
		_ = s.db.WithContext(ctx).Where("id = ?", parsedUserID).First(&user).Error
	}
	if !roleIsInternal(roles) {
		db = db.Where("recipient_email = ? OR visible_user_ids::text LIKE ?", user.Email, "%"+userID+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return basedto.PageResult[DocumentView]{}, err
	}
	var rows []DocumentInstance
	if err := db.Order("create_time desc").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return basedto.PageResult[DocumentView]{}, err
	}
	items := make([]DocumentView, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDocumentView(row, false))
	}
	return basedto.PageResult[DocumentView]{
		List:       items,
		Pagination: basedto.PaginationInfo{Page: query.Page, PageSize: query.PageSize, Total: total},
	}, nil
}

func (s *Service) GetDocument(ctx context.Context, documentID string) (DocumentView, error) {
	if err := s.syncDocumentApproval(ctx); err != nil {
		return DocumentView{}, err
	}
	var row DocumentInstance
	if err := s.db.WithContext(ctx).Where("document_code = ? AND is_deleted = false", documentID).First(&row).Error; err != nil {
		return DocumentView{}, err
	}
	return toDocumentView(row, true), nil
}

func (s *Service) SubmitDocument(ctx context.Context, documentID, actorID, actorName string) (basedto.JSONMap, error) {
	var row DocumentInstance
	if err := s.db.WithContext(ctx).Where("document_code = ? AND is_deleted = false", documentID).First(&row).Error; err != nil {
		return nil, err
	}
	created, err := s.approvalService.CreateApproval(ctx, approval.CreateApprovalRequest{
		BusinessType: "document_send",
		BusinessID:   strconv.FormatUint(row.ID, 10),
		Title:        "函件发送审批：" + row.Title,
		Payload:      basedto.JSONMap{"document_id": row.ID},
	}, actorID, actorName)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&DocumentInstance{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":      "pending_level_1",
		"approval_id": created.ApprovalID,
	}).Error; err != nil {
		return nil, err
	}
	return basedto.JSONMap{"approval_id": created.ApprovalID, "status": "pending_level_1"}, nil
}

func (s *Service) SendDocument(ctx context.Context, documentID, actorID, actorName string) (basedto.JSONMap, error) {
	if err := s.syncDocumentApproval(ctx); err != nil {
		return nil, err
	}
	var row DocumentInstance
	if err := s.db.WithContext(ctx).Where("document_code = ? AND is_deleted = false", documentID).First(&row).Error; err != nil {
		return nil, err
	}
	if row.Status == "sent" {
		return basedto.JSONMap{
			"status":               "sent",
			"sent_at":              formatRFC3339(row.SentAt),
			"sender_mailbox_id":    derefUint64(row.SenderMailboxID),
			"sender_mailbox_name":  row.SenderMailboxName,
			"sender_mailbox_email": row.SenderMailboxEmail,
			"send_result":          row.SendResult,
		}, nil
	}
	if row.Status != "approved" && row.Status != "ready_to_send" {
		return nil, fmt.Errorf("document %s cannot be sent while status is %s", row.DocumentCode, row.Status)
	}
	if row.SubmittedByUserID == "" || row.SubmittedByUserID != actorID {
		return nil, fmt.Errorf("only the document initiator can send an approved document")
	}
	if !hasFormalDocumentDOCX(row) {
		return nil, fmt.Errorf("document %s has no formal DOCX template configured yet", row.DocumentCode)
	}
	sendMailbox, err := s.resolveSendMailbox(ctx, derefUint64(row.SenderMailboxID))
	if err != nil {
		return nil, err
	}
	if shouldUseManualDelivery(sendMailbox) {
		return s.markDocumentDelivered(ctx, row, sendMailbox, actorID, actorName, "manual_external_delivery", "manual_delivery", "函件已生成，请下载后通过人工渠道外发")
	}
	resultMessage := "smtp_sent"
	channelMessage := "已通过 SMTP 发送正式函件"
	if strings.EqualFold(sendMailbox.OAuthProvider, "microsoft_graph") {
		resultMessage = "microsoft_graph_sent"
		channelMessage = "已通过 Microsoft Graph 发送正式函件"
	}
	recipients := append([]basedto.PersonRef{{Name: row.RecipientName, Email: row.RecipientEmail}}, fromRecipients(row.CCRecipients)...)
	bodyHTML := withMailboxSignature(row.HTMLContent, sendMailbox.SignatureHTML)
	subject := sanitizeMailSubject(row.Title)
	if err := validateExternalDocumentContent(subject, bodyHTML); err != nil {
		return nil, err
	}
	if err := s.sendOutbound(ctx, sendMailbox, outboundMessage{
		FromName:    sendMailbox.Name,
		FromAddress: sendMailbox.EmailAddress,
		To:          recipients,
		Subject:     subject,
		HTMLBody:    bodyHTML,
		Attachments: []outboundAttachment{{
			FileName:    row.DOCXFileName,
			ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			Data:        row.DOCXData,
		}},
	}); err != nil {
		_ = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			delivery := DocumentDelivery{
				DocumentID:     row.ID,
				Channel:        "email",
				Status:         "failed",
				MailboxID:      &sendMailbox.ID,
				RecipientEmail: row.RecipientEmail,
				Message:        truncateString(err.Error(), 255),
			}
			if createErr := tx.Create(&delivery).Error; createErr != nil {
				return createErr
			}
			return tx.Model(&DocumentInstance{}).Where("id = ?", row.ID).Update("send_result", "smtp_failed:"+truncateString(err.Error(), 200)).Error
		})
		return nil, err
	}
	now := time.Now()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&DocumentInstance{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":               "sent",
			"sent_at":              &now,
			"sender_mailbox_id":    sendMailbox.ID,
			"sender_mailbox_name":  sendMailbox.Name,
			"sender_mailbox_email": sendMailbox.EmailAddress,
			"send_result":          resultMessage,
		}).Error; err != nil {
			return err
		}
		delivery := DocumentDelivery{
			DocumentID:     row.ID,
			Channel:        "email",
			Status:         "sent",
			MailboxID:      &sendMailbox.ID,
			RecipientEmail: row.RecipientEmail,
			Message:        channelMessage,
			SentAt:         &now,
		}
		if err := tx.Create(&delivery).Error; err != nil {
			return err
		}
		email := Email{
			EmailCode:        "MAIL-" + uuid.NewString()[:8],
			MailboxID:        sendMailbox.ID,
			Direction:        "outbound",
			Folder:           "sent",
			Subject:          cleanUTF8(subject),
			Preview:          summarizeHTML(bodyHTML),
			BodyHTML:         cleanUTF8(bodyHTML),
			SenderName:       cleanUTF8(sendMailbox.Name),
			SenderEmail:      cleanUTF8(sendMailbox.EmailAddress),
			ToRecipients:     toRecipients(cleanPersonRefs(recipients)),
			Summary:          summarizeHTML(bodyHTML),
			PriorityScore:    60,
			PriorityLabel:    "high",
			Status:           "sent",
			OperatorUserID:   cleanUTF8(actorID),
			OperatorUsername: cleanUTF8(actorName),
			SentAt:           &now,
		}
		if err := tx.Create(&email).Error; err != nil {
			return err
		}
		if len(row.DOCXData) == 0 {
			return nil
		}
		return tx.Create(&EmailAttachment{
			EmailID:     email.ID,
			FileName:    cleanUTF8(row.DOCXFileName),
			ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			FileSize:    int64(len(row.DOCXData)),
			FileData:    row.DOCXData,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return basedto.JSONMap{
		"status":               "sent",
		"sent_at":              now.Format(time.RFC3339),
		"sender_mailbox_id":    sendMailbox.ID,
		"sender_mailbox_name":  sendMailbox.Name,
		"sender_mailbox_email": sendMailbox.EmailAddress,
		"send_result":          resultMessage,
	}, nil
}

func shouldUseManualDelivery(mailbox MailboxAccount) bool {
	if strings.EqualFold(mailbox.OAuthProvider, "microsoft_graph") {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(mailbox.SMTPHost))
	email := strings.ToLower(strings.TrimSpace(mailbox.EmailAddress))
	if host == "" || mailbox.SMTPPort <= 0 {
		return true
	}
	if strings.Contains(host, "example.com") || strings.HasSuffix(host, ".local") {
		return true
	}
	if strings.HasSuffix(email, ".local") {
		return true
	}
	if strings.EqualFold(mailbox.AuthType, "password") && strings.TrimSpace(mailbox.PasswordSecret) == "" {
		return true
	}
	return false
}

func (s *Service) markDocumentDelivered(ctx context.Context, row DocumentInstance, sendMailbox MailboxAccount, actorID, actorName, resultMessage, deliveryStatus, deliveryMessage string) (basedto.JSONMap, error) {
	now := time.Now()
	ccRecipients := fromRecipients(row.CCRecipients)
	bodyHTML := withMailboxSignature(row.HTMLContent, sendMailbox.SignatureHTML)
	subject := sanitizeMailSubject(row.Title)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&DocumentInstance{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":               "sent",
			"sent_at":              &now,
			"sender_mailbox_id":    sendMailbox.ID,
			"sender_mailbox_name":  sendMailbox.Name,
			"sender_mailbox_email": sendMailbox.EmailAddress,
			"send_result":          resultMessage,
		}).Error; err != nil {
			return err
		}
		delivery := DocumentDelivery{
			DocumentID:     row.ID,
			Channel:        "email",
			Status:         deliveryStatus,
			MailboxID:      &sendMailbox.ID,
			RecipientEmail: row.RecipientEmail,
			Message:        deliveryMessage,
			SentAt:         &now,
		}
		if err := tx.Create(&delivery).Error; err != nil {
			return err
		}
		email := Email{
			EmailCode:        "MAIL-" + uuid.NewString()[:8],
			MailboxID:        sendMailbox.ID,
			Direction:        "outbound",
			Folder:           "sent",
			Subject:          cleanUTF8(subject),
			Preview:          summarizeHTML(bodyHTML),
			BodyHTML:         cleanUTF8(bodyHTML),
			SenderName:       cleanUTF8(sendMailbox.Name),
			SenderEmail:      cleanUTF8(sendMailbox.EmailAddress),
			ToRecipients:     toRecipients(cleanPersonRefs([]basedto.PersonRef{{Name: row.RecipientName, Email: row.RecipientEmail}})),
			CcRecipients:     toRecipients(cleanPersonRefs(ccRecipients)),
			Summary:          summarizeHTML(bodyHTML),
			PriorityScore:    60,
			PriorityLabel:    "high",
			Status:           deliveryStatus,
			OperatorUserID:   cleanUTF8(actorID),
			OperatorUsername: cleanUTF8(actorName),
			SentAt:           &now,
		}
		if err := tx.Create(&email).Error; err != nil {
			return err
		}
		if len(row.DOCXData) == 0 {
			return nil
		}
		return tx.Create(&EmailAttachment{
			EmailID:     email.ID,
			FileName:    cleanUTF8(row.DOCXFileName),
			ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			FileSize:    int64(len(row.DOCXData)),
			FileData:    row.DOCXData,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return basedto.JSONMap{
		"status":               "sent",
		"sent_at":              now.Format(time.RFC3339),
		"sender_mailbox_id":    sendMailbox.ID,
		"sender_mailbox_name":  sendMailbox.Name,
		"sender_mailbox_email": sendMailbox.EmailAddress,
		"send_result":          resultMessage,
	}, nil
}

func (s *Service) DownloadDocument(ctx context.Context, documentID string) ([]byte, string, error) {
	var row DocumentInstance
	if err := s.db.WithContext(ctx).Where("document_code = ? AND is_deleted = false", documentID).First(&row).Error; err != nil {
		return nil, "", err
	}
	if !hasFormalDocumentDOCX(row) {
		return nil, "", fmt.Errorf("document %s has no formal DOCX template configured yet", row.DocumentCode)
	}
	return row.DOCXData, row.DOCXFileName, nil
}

func hasFormalTemplateDOCX(version TemplateVersion) bool {
	return strings.TrimSpace(version.DOCXTemplateName) != "" && len(version.DOCXTemplateData) > 0
}

func hasFormalDocumentDOCX(row DocumentInstance) bool {
	return strings.TrimSpace(row.DOCXFileName) != "" && len(row.DOCXData) > 0
}

func (s *Service) findMailbox(ctx context.Context, mailboxID string) (MailboxAccount, error) {
	id, err := strconv.ParseUint(mailboxID, 10, 64)
	if err != nil {
		return MailboxAccount{}, err
	}
	var box MailboxAccount
	err = s.db.WithContext(ctx).Where("id = ? AND is_deleted = false", id).First(&box).Error
	return box, err
}

func (s *Service) resolveSendMailbox(ctx context.Context, mailboxID uint64) (MailboxAccount, error) {
	var box MailboxAccount
	db := s.db.WithContext(ctx).Where("is_deleted = false AND status = ?", "active")
	if mailboxID > 0 {
		db = db.Where("id = ?", mailboxID)
	} else {
		db = db.Where("is_default_sender = true")
	}
	if err := db.Order("id asc").First(&box).Error; err != nil {
		return MailboxAccount{}, err
	}
	return box, nil
}

func (s *Service) resolveDocumentVisibleUserIDs(ctx context.Context, recipientEmail string, explicit []string) ([]string, error) {
	seen := map[string]struct{}{}
	items := make([]string, 0, len(explicit)+1)
	for _, item := range explicit {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}
	if strings.TrimSpace(recipientEmail) == "" {
		return items, nil
	}
	var users []database.SysUser
	err := s.db.WithContext(ctx).
		Table("sys_user").
		Select("sys_user.*").
		Joins("JOIN sys_user_role sur ON sur.user_id = sys_user.id AND sur.is_deleted = false").
		Joins("JOIN sys_role sr ON sr.id = sur.role_id AND sr.is_deleted = false").
		Where("sys_user.is_deleted = false AND sys_user.status = ? AND lower(sys_user.email) = lower(?) AND sr.role_code = ?", "active", recipientEmail, "LP").
		Find(&users).Error
	if err != nil {
		if isMissingSystemUserTableError(err) {
			return items, nil
		}
		return nil, err
	}
	for _, user := range users {
		id := strconv.FormatUint(user.ID, 10)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		items = append(items, id)
	}
	return items, nil
}

func isMissingSystemUserTableError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table: sys_user") ||
		strings.Contains(message, "no such table: sys_user_role") ||
		strings.Contains(message, "no such table: sys_role") ||
		strings.Contains(message, `relation "sys_user" does not exist`) ||
		strings.Contains(message, `relation "sys_user_role" does not exist`) ||
		strings.Contains(message, `relation "sys_role" does not exist`)
}

func (s *Service) getTemplateView(ctx context.Context, tpl Template) (TemplateView, error) {
	var variables []TemplateVariable
	if err := s.db.WithContext(ctx).Where("template_id = ? AND is_deleted = false", tpl.ID).Find(&variables).Error; err != nil {
		return TemplateView{}, err
	}
	view := TemplateView{
		TemplateID:         strconv.FormatUint(tpl.ID, 10),
		TemplateCode:       tpl.TemplateCode,
		Name:               tpl.Name,
		Category:           tpl.Category,
		Scene:              tpl.Scene,
		UsageBoundary:      tpl.UsageBoundary,
		TemplateSource:     tpl.TemplateSource,
		OwnerUserID:        tpl.OwnerUserID,
		SourceRefID:        uint64ToString(tpl.SourceRefID),
		IsEditable:         tpl.IsEditable,
		AISessionID:        tpl.AISessionID,
		ImportFileName:     tpl.ImportFileName,
		ParseStatus:        tpl.ParseStatus,
		Status:             tpl.Status,
		PublishedVersionID: uint64ToString(tpl.PublishedVersionID),
		Variables:          toTemplateVariableInputs(variables),
	}
	var version TemplateVersion
	if tpl.PublishedVersionID != nil {
		if err := s.db.WithContext(ctx).Where("id = ?", *tpl.PublishedVersionID).First(&version).Error; err == nil {
			view.CurrentVersion = &TemplateVersionView{
				VersionID:        strconv.FormatUint(version.ID, 10),
				VersionNo:        version.VersionNo,
				SubjectTemplate:  version.SubjectTemplate,
				HTMLContent:      version.HTMLContent,
				DOCXTemplateName: version.DOCXTemplateName,
				Status:           version.Status,
				ApprovalID:       version.ApprovalID,
				SubmittedAt:      formatRFC3339(version.SubmittedAt),
				PublishedAt:      formatRFC3339(version.PublishedAt),
				Variables:        toTemplateVariableInputs(variables),
			}
		}
	}
	return view, nil
}

func (s *Service) resolveTemplateForRender(ctx context.Context, templateID string) (Template, TemplateVersion, []TemplateVariable, error) {
	id, err := strconv.ParseUint(templateID, 10, 64)
	if err != nil {
		return Template{}, TemplateVersion{}, nil, err
	}
	var tpl Template
	if err := s.db.WithContext(ctx).Where("id = ? AND is_deleted = false", id).First(&tpl).Error; err != nil {
		return Template{}, TemplateVersion{}, nil, err
	}
	if tpl.TemplateSource == templateSourceUserPrivate {
		var version TemplateVersion
		if err := s.db.WithContext(ctx).Where("template_id = ? AND is_deleted = false", tpl.ID).Order("version_no desc").First(&version).Error; err != nil {
			return Template{}, TemplateVersion{}, nil, err
		}
		var variables []TemplateVariable
		if err := s.db.WithContext(ctx).Where("template_id = ? AND is_deleted = false", tpl.ID).Find(&variables).Error; err != nil {
			return Template{}, TemplateVersion{}, nil, err
		}
		return tpl, version, variables, nil
	}
	if tpl.PublishedVersionID == nil {
		return Template{}, TemplateVersion{}, nil, fmt.Errorf("template %s has no published version", tpl.TemplateCode)
	}
	var version TemplateVersion
	if err := s.db.WithContext(ctx).Where("id = ?", *tpl.PublishedVersionID).First(&version).Error; err != nil {
		return Template{}, TemplateVersion{}, nil, err
	}
	var variables []TemplateVariable
	if err := s.db.WithContext(ctx).Where("template_id = ? AND is_deleted = false", tpl.ID).Find(&variables).Error; err != nil {
		return Template{}, TemplateVersion{}, nil, err
	}
	return tpl, version, variables, nil
}

func (s *Service) syncTemplatePublication(ctx context.Context) error {
	var versions []TemplateVersion
	if err := s.db.WithContext(ctx).Where("approval_id <> '' AND status = ?", "pending_approval").Find(&versions).Error; err != nil {
		return err
	}
	for _, version := range versions {
		approvalView, err := s.approvalService.GetApproval(ctx, version.ApprovalID)
		if err != nil {
			continue
		}
		switch approvalView.Status {
		case "approved":
			_ = s.publishApprovedTemplate(ctx, version)
		case "rejected", "returned":
			_ = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&TemplateVersion{}).Where("id = ?", version.ID).Update("status", approvalView.Status).Error; err != nil {
					return err
				}
				return tx.Model(&Template{}).Where("id = ?", version.TemplateID).Update("status", approvalView.Status).Error
			})
		}
	}
	return nil
}

func (s *Service) syncDocumentApproval(ctx context.Context) error {
	var docs []DocumentInstance
	if err := s.db.WithContext(ctx).Where("approval_id <> '' AND status IN ?", []string{"pending_level_1", "pending_level_2", "pending_approval"}).Find(&docs).Error; err != nil {
		return err
	}
	for _, row := range docs {
		approvalView, err := s.approvalService.GetApproval(ctx, row.ApprovalID)
		if err != nil {
			continue
		}
		nextStatus := row.Status
		switch approvalView.Status {
		case "pending":
			if approvalView.CurrentStepNo == 2 {
				nextStatus = "pending_level_2"
			} else {
				nextStatus = "pending_level_1"
			}
		case "approved":
			nextStatus = "approved"
		case "rejected", "returned":
			nextStatus = approvalView.Status
		}
		if nextStatus != row.Status {
			if err := s.db.WithContext(ctx).Model(&DocumentInstance{}).Where("id = ?", row.ID).Update("status", nextStatus).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func canAccessTemplate(tpl Template, actorID string) bool {
	switch tpl.TemplateSource {
	case templateSourceSystemBuiltin, templateSourcePublishedPublic:
		return true
	case templateSourceUserPrivate:
		return strings.TrimSpace(tpl.OwnerUserID) != "" && strings.TrimSpace(tpl.OwnerUserID) == strings.TrimSpace(actorID)
	default:
		return false
	}
}

func (s *Service) publishApprovedTemplate(ctx context.Context, version TemplateVersion) error {
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tpl Template
		if err := tx.Where("id = ? AND is_deleted = false", version.TemplateID).First(&tpl).Error; err != nil {
			return err
		}
		if err := tx.Model(&TemplateVersion{}).Where("id = ?", version.ID).Updates(map[string]any{
			"status":       "published",
			"published_at": &now,
		}).Error; err != nil {
			return err
		}
		if tpl.TemplateSource != templateSourceUserPrivate {
			return tx.Model(&Template{}).Where("id = ?", tpl.ID).Updates(map[string]any{
				"status":               "published",
				"published_version_id": version.ID,
			}).Error
		}
		if err := tx.Model(&Template{}).Where("id = ?", tpl.ID).Updates(map[string]any{
			"status":               "approved",
			"published_version_id": version.ID,
		}).Error; err != nil {
			return err
		}
		return publishTemplateCopy(tx, tpl, version, now)
	})
}

func publishTemplateCopy(tx *gorm.DB, source Template, sourceVersion TemplateVersion, publishedAt time.Time) error {
	var publicTpl Template
	err := tx.Where("source_ref_id = ? AND template_source = ? AND is_deleted = false", source.ID, templateSourcePublishedPublic).First(&publicTpl).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		publicTpl = Template{
			TemplateCode:   "PUB-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", "")),
			Name:           source.Name,
			Category:       source.Category,
			Scene:          source.Scene,
			TemplateSource: templateSourcePublishedPublic,
			OwnerUserID:    source.OwnerUserID,
			SourceRefID:    &source.ID,
			IsEditable:     false,
			Status:         "published",
			CreatorUserID:  source.CreatorUserID,
			CreatorName:    source.CreatorName,
		}
		if err := tx.Create(&publicTpl).Error; err != nil {
			return err
		}
		if err := tx.Model(&Template{}).Where("id = ?", publicTpl.ID).Update("is_editable", false).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&Template{}).Where("id = ?", publicTpl.ID).Updates(map[string]any{
		"name":        source.Name,
		"category":    source.Category,
		"scene":       source.Scene,
		"status":      "published",
		"is_editable": false,
	}).Error; err != nil {
		return err
	}
	if err := tx.Where("template_id = ?", publicTpl.ID).Delete(&TemplateVariable{}).Error; err != nil {
		return err
	}
	var sourceVariables []TemplateVariable
	if err := tx.Where("template_id = ? AND is_deleted = false", source.ID).Find(&sourceVariables).Error; err != nil {
		return err
	}
	for _, variable := range sourceVariables {
		copyRow := TemplateVariable{
			TemplateID:    publicTpl.ID,
			VariableName:  variable.VariableName,
			VariableLabel: variable.VariableLabel,
			VariableType:  variable.VariableType,
			IsRequired:    variable.IsRequired,
			DefaultValue:  variable.DefaultValue,
		}
		if err := tx.Create(&copyRow).Error; err != nil {
			return err
		}
	}
	publicVersion := TemplateVersion{
		TemplateID:       publicTpl.ID,
		VersionNo:        nextTemplateVersionNo(tx, publicTpl.ID),
		SubjectTemplate:  sourceVersion.SubjectTemplate,
		HTMLContent:      sourceVersion.HTMLContent,
		DOCXTemplateName: sourceVersion.DOCXTemplateName,
		DOCXTemplateData: sourceVersion.DOCXTemplateData,
		Status:           "published",
		PublishedAt:      &publishedAt,
	}
	if err := tx.Create(&publicVersion).Error; err != nil {
		return err
	}
	return tx.Model(&Template{}).Where("id = ?", publicTpl.ID).Update("published_version_id", publicVersion.ID).Error
}

func nextTemplateVersionNo(tx *gorm.DB, templateID uint64) int {
	var lastVersion TemplateVersion
	if err := tx.Where("template_id = ?", templateID).Order("version_no desc").First(&lastVersion).Error; err == nil {
		return lastVersion.VersionNo + 1
	}
	return 1
}

func seedBuiltinTemplates(tx *gorm.DB) error {
	items := []struct {
		code      string
		name      string
		category  string
		scene     string
		subject   string
		html      string
		docxName  string
		variables []TemplateVariableInput
	}{
		{
			code:     "SYS_INVESTOR_COMM_RECORD",
			name:     "投资者关系活动记录模板",
			category: "notice",
			scene:    "investor_relations",
			subject:  "{{activity_topic}}沟通纪要",
			html:     "<h1>{{activity_topic}}沟通纪要</h1><p>沟通对象：{{counterparty_name}}</p><p>活动时间：{{activity_date}}</p><p>交流要点：{{key_points}}</p>",
			docxName: "investor-communication-record.docx",
			variables: []TemplateVariableInput{
				{VariableName: "activity_topic", VariableLabel: "活动主题", VariableType: "text", IsRequired: true},
				{VariableName: "counterparty_name", VariableLabel: "沟通对象名称", VariableType: "text", IsRequired: true},
				{VariableName: "activity_date", VariableLabel: "活动日期", VariableType: "date", IsRequired: true},
				{VariableName: "key_points", VariableLabel: "交流要点", VariableType: "textarea", IsRequired: true},
			},
		},
		{
			code:     "SYS_STRATEGIC_COOP_NOTICE",
			name:     "战略合作沟通通知模板",
			category: "notice",
			scene:    "strategic_cooperation",
			subject:  "{{partner_name}}合作事项通知",
			html:     "<h1>{{partner_name}}合作事项通知</h1><p>合作事项：{{cooperation_subject}}</p><p>合作背景：{{cooperation_background}}</p><p>生效日期：{{effective_date}}</p>",
			docxName: "strategic-cooperation-notice.docx",
			variables: []TemplateVariableInput{
				{VariableName: "partner_name", VariableLabel: "合作方名称", VariableType: "text", IsRequired: true},
				{VariableName: "cooperation_subject", VariableLabel: "合作事项名称", VariableType: "text", IsRequired: true},
				{VariableName: "cooperation_background", VariableLabel: "合作背景", VariableType: "textarea", IsRequired: true},
				{VariableName: "effective_date", VariableLabel: "生效日期", VariableType: "date", IsRequired: true},
			},
		},
	}
	for _, item := range items {
		if err := upsertBuiltinTemplate(tx, item.code, item.name, item.category, item.scene, item.subject, item.html, item.docxName, item.variables); err != nil {
			return err
		}
	}
	return nil
}

func upsertBuiltinTemplate(tx *gorm.DB, code, name, category, scene, subject, html, docxName string, variables []TemplateVariableInput) error {
	var tpl Template
	err := tx.Where("template_code = ?", code).First(&tpl).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		tpl = Template{
			TemplateCode:   code,
			Name:           name,
			Category:       category,
			Scene:          scene,
			TemplateSource: templateSourceSystemBuiltin,
			IsEditable:     false,
			ParseStatus:    "ready",
			Status:         "published",
			CreatorUserID:  "system",
			CreatorName:    "system",
		}
		if err := tx.Create(&tpl).Error; err != nil {
			return err
		}
		if err := tx.Model(&Template{}).Where("id = ?", tpl.ID).Update("is_editable", false).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&Template{}).Where("id = ?", tpl.ID).Updates(map[string]any{
		"name":            name,
		"category":        category,
		"scene":           scene,
		"template_source": templateSourceSystemBuiltin,
		"is_editable":     false,
		"parse_status":    "ready",
		"status":          "published",
	}).Error; err != nil {
		return err
	}
	if err := tx.Where("template_id = ?", tpl.ID).Delete(&TemplateVariable{}).Error; err != nil {
		return err
	}
	for _, variable := range variables {
		row := TemplateVariable{
			TemplateID:    tpl.ID,
			VariableName:  variable.VariableName,
			VariableLabel: variable.VariableLabel,
			VariableType:  defaultString(variable.VariableType, "text"),
			IsRequired:    variable.IsRequired,
			DefaultValue:  variable.DefaultValue,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	docxData := sampleDOCXTemplate()
	var versions []TemplateVersion
	if err := tx.Where("template_id = ? AND is_deleted = false", tpl.ID).Order("version_no asc").Find(&versions).Error; err != nil {
		return err
	}
	for _, version := range versions {
		if version.SubjectTemplate != subject || version.HTMLContent != html || version.DOCXTemplateName != docxName {
			continue
		}
		duplicateIDs := make([]uint64, 0)
		for _, candidate := range versions {
			if candidate.ID != version.ID && candidate.SubjectTemplate == subject && candidate.HTMLContent == html && candidate.DOCXTemplateName == docxName {
				duplicateIDs = append(duplicateIDs, candidate.ID)
			}
		}
		if len(duplicateIDs) > 0 {
			if err := tx.Model(&TemplateVersion{}).Where("id IN ?", duplicateIDs).Update("is_deleted", true).Error; err != nil {
				return err
			}
		}
		return tx.Model(&Template{}).Where("id = ?", tpl.ID).Update("published_version_id", version.ID).Error
	}
	publishedAt := time.Now()
	version := TemplateVersion{
		TemplateID:       tpl.ID,
		VersionNo:        nextTemplateVersionNo(tx, tpl.ID),
		SubjectTemplate:  subject,
		HTMLContent:      html,
		DOCXTemplateName: docxName,
		DOCXTemplateData: docxData,
		Status:           "published",
		PublishedAt:      &publishedAt,
	}
	if err := tx.Create(&version).Error; err != nil {
		return err
	}
	return tx.Model(&Template{}).Where("id = ?", tpl.ID).Update("published_version_id", version.ID).Error
}

func toEmailView(email Email, withBody bool) EmailView {
	view := EmailView{
		EmailID:        email.EmailCode,
		MailboxID:      email.MailboxID,
		Folder:         email.Folder,
		Subject:        email.Subject,
		Preview:        email.Preview,
		Sender:         basedto.PersonRef{Name: email.SenderName, Email: email.SenderEmail},
		Recipients:     fromRecipients(email.ToRecipients),
		CCRecipients:   fromRecipients(email.CcRecipients),
		BCCRecipients:  fromRecipients(email.BccRecipients),
		Summary:        email.Summary,
		ReplyDraft:     email.ReplyDraft,
		PriorityLabel:  email.PriorityLabel,
		PriorityScore:  email.PriorityScore,
		NeedsReply:     email.NeedsReply,
		IsRead:         email.IsRead,
		IsStarred:      email.IsStarred,
		HasAttachments: email.HasAttachments,
		Status:         email.Status,
		ReceivedAt:     formatRFC3339(email.ReceivedAt),
		SentAt:         formatRFC3339(email.SentAt),
	}
	if withBody {
		view.BodyHTML = email.BodyHTML
	}
	return view
}

func toMailboxView(item MailboxAccount) MailboxView {
	return MailboxView{
		MailboxID:           item.ID,
		MailboxCode:         item.MailboxCode,
		Name:                item.Name,
		MailboxType:         item.MailboxType,
		EmailAddress:        item.EmailAddress,
		AuthType:            item.AuthType,
		SMTPHost:            item.SMTPHost,
		SMTPPort:            item.SMTPPort,
		IMAPHost:            item.IMAPHost,
		IMAPPort:            item.IMAPPort,
		Username:            item.Username,
		OAuthProvider:       item.OAuthProvider,
		SignatureHTML:       item.SignatureHTML,
		IsDefaultSender:     item.IsDefaultSender,
		Status:              item.Status,
		LastSyncAt:          formatRFC3339(item.LastSyncAt),
		LastSyncStatus:      item.LastSyncStatus,
		LastSyncMessage:     item.LastSyncMessage,
		LastIMAPUID:         item.LastIMAPUID,
		LastIMAPUIDValidity: item.LastIMAPUIDValidity,
	}
}

func toDocumentView(row DocumentInstance, withContent bool) DocumentView {
	var vars map[string]any
	_ = decodeJSON(row.VariableSnapshot, &vars)
	view := DocumentView{
		DocumentID:         row.DocumentCode,
		TemplateID:         strconv.FormatUint(row.TemplateID, 10),
		Title:              row.Title,
		DocumentType:       row.DocumentType,
		RecipientName:      row.RecipientName,
		RecipientEmail:     row.RecipientEmail,
		CCRecipients:       fromRecipients(row.CCRecipients),
		SenderMailboxID:    derefUint64(row.SenderMailboxID),
		SenderMailboxName:  row.SenderMailboxName,
		SenderMailboxEmail: row.SenderMailboxEmail,
		DOCXFileName:       row.DOCXFileName,
		Status:             row.Status,
		ApprovalID:         row.ApprovalID,
		SubmittedByUserID:  row.SubmittedByUserID,
		Variables:          vars,
		SendResult:         row.SendResult,
		SentAt:             formatRFC3339(row.SentAt),
	}
	if withContent {
		view.HTMLContent = row.HTMLContent
		view.DOCXBase64 = encodeBase64(row.DOCXData)
	}
	return view
}

func toTemplateVariableInputs(values []TemplateVariable) []TemplateVariableInput {
	items := make([]TemplateVariableInput, 0, len(values))
	for _, value := range values {
		items = append(items, TemplateVariableInput{
			VariableName:  value.VariableName,
			VariableLabel: value.VariableLabel,
			VariableType:  value.VariableType,
			IsRequired:    value.IsRequired,
			DefaultValue:  value.DefaultValue,
		})
	}
	return items
}

func toAIKnownFields(values []TemplateVariableInput) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, map[string]interface{}{
			"variable_name":  value.VariableName,
			"variable_label": value.VariableLabel,
			"variable_type":  value.VariableType,
			"is_required":    value.IsRequired,
			"default_value":  value.DefaultValue,
		})
	}
	return items
}

func fallbackTemplateChatAssist(req TemplateChatAssistRequest) TemplateAssistResponse {
	fields := mergeSuggestedFields(
		[]TemplateVariableInput{
			{VariableName: "fund_name", VariableLabel: "基金名称", VariableType: "text", IsRequired: true},
			{VariableName: "recipient_name", VariableLabel: "收件对象名称", VariableType: "text", IsRequired: true},
			{VariableName: "notice_date", VariableLabel: "通知日期", VariableType: "date", IsRequired: true},
		},
		req.KnownFields,
	)
	subject := defaultString(strings.TrimSpace(req.Goal), "正式沟通通知")
	return TemplateAssistResponse{
		Summary:           summarizeChatGoal(req.Goal, req.LatestUserMessage),
		FollowUpQuestions: nil,
		SuggestedFields:   fields,
		SubjectTemplate:   subject + "｜{{fund_name}}｜{{notice_date}}",
		HTMLTemplate: strings.Join([]string{
			"<p>尊敬的 {{recipient_name}}：</p>",
			"<p>您好。</p>",
			"<p>现就 {{fund_name}} 相关事项向您发送正式通知，请查收。</p>",
			"<p>本次沟通主题：" + escapeHTML(defaultString(strings.TrimSpace(req.LatestUserMessage), req.Goal)) + "</p>",
			"<p>如需进一步沟通，请及时与我们联系。</p>",
			"<p>{{notice_date}}</p>",
		}, ""),
		AISource:        "local_fallback",
		AISourceMessage: "Python AI 服务不可用，当前为后端本地兜底结果，请检查 9100 服务。",
	}
}

func fallbackTemplateImportAssist(req TemplateImportAssistRequest, plainText string, paragraphs []string) TemplateAssistResponse {
	fields := []TemplateVariableInput{
		{VariableName: "fund_name", VariableLabel: "基金名称", VariableType: "text", IsRequired: true},
		{VariableName: "recipient_name", VariableLabel: "收件对象名称", VariableType: "text", IsRequired: true},
		{VariableName: "notice_date", VariableLabel: "通知日期", VariableType: "date", IsRequired: true},
	}
	summary := "已根据导入文档提取出一版模板草稿，可继续补充字段并直接保存。"
	if strings.TrimSpace(req.FileName) != "" {
		summary = "已根据导入文件《" + req.FileName + "》提取出一版模板草稿，可继续补充字段并直接保存。"
	}
	return TemplateAssistResponse{
		Summary:           summary,
		FollowUpQuestions: nil,
		SuggestedFields:   fields,
		SubjectTemplate:   defaultString(strings.TrimSuffix(strings.TrimSpace(req.FileName), ".docx"), "导入模板") + "｜{{fund_name}}",
		HTMLTemplate:      fallbackHTMLFromParagraphs(paragraphs),
		AISource:          "local_fallback",
		AISourceMessage:   "Python AI 服务不可用，当前为后端本地兜底结果，请检查 9100 服务。",
		PlainText:         plainText,
		Paragraphs:        paragraphs,
	}
}

func fallbackTemplateRenderAssist(req TemplateRenderAssistInput) TemplateAssistResponse {
	recipientName := firstNonEmptyKnownValue(req.KnownValues, "recipient_name", "lp_name", "partner_name", "company_name")
	if recipientName == "" {
		recipientName = "收件方"
	}
	fundName := firstNonEmptyKnownValue(req.KnownValues, "fund_name")
	if fundName == "" {
		fundName = "基金管理人"
	}
	noticeDate := firstNonEmptyKnownValue(req.KnownValues, "notice_date")
	if noticeDate == "" {
		noticeDate = time.Now().Format("2006-01-02")
	}
	subject := strings.TrimSpace(req.TemplateName)
	if subject == "" {
		subject = defaultString(strings.TrimSpace(req.BusinessGoal), "正式沟通通知")
	}
	body := strings.Join([]string{
		"<p>尊敬的 " + escapeHTML(recipientName) + "：</p>",
		"<p>您好。</p>",
		"<p>现就" + escapeHTML(fundName) + "相关事项向您发送正式通知，请查收。</p>",
		"<p>本次沟通事项：" + escapeHTML(defaultString(strings.TrimSpace(req.BusinessGoal), "请查收正式函件及附件内容。")) + "</p>",
		"<p>如需进一步沟通，请及时与我们联系。</p>",
		"<p>" + escapeHTML(noticeDate) + "</p>",
	}, "")
	return TemplateAssistResponse{
		Summary:           "已生成一版正式邮件正文建议，可继续手动修改后再预览或创建函件。",
		FollowUpQuestions: nil,
		SuggestedFields:   nil,
		SubjectTemplate:   subject,
		HTMLTemplate:      body,
		AISource:          "local_fallback",
		AISourceMessage:   "Python AI 服务不可用，当前为后端本地兜底结果，请检查 9100 服务。",
	}
}

func toTemplateAssistResponse(value baseai.TemplateDraftResponse, plainText string, paragraphs []string) TemplateAssistResponse {
	fields := make([]TemplateVariableInput, 0, len(value.SuggestedFields))
	for _, field := range value.SuggestedFields {
		fields = append(fields, TemplateVariableInput{
			VariableName:  field.VariableName,
			VariableLabel: field.VariableLabel,
			VariableType:  defaultString(field.VariableType, "text"),
			IsRequired:    field.IsRequired,
		})
	}
	return TemplateAssistResponse{
		Summary:           value.Summary,
		FollowUpQuestions: value.FollowUpQuestions,
		SuggestedFields:   fields,
		SubjectTemplate:   value.SubjectTemplate,
		HTMLTemplate:      value.HTMLTemplate,
		AISource:          "ai_service",
		AISourceMessage:   "已连接 Python AI 服务，当前为真实 AI 生成结果。",
		PlainText:         plainText,
		Paragraphs:        paragraphs,
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func mergeSuggestedFields(base []TemplateVariableInput, extra []TemplateVariableInput) []TemplateVariableInput {
	seen := make(map[string]struct{}, len(base)+len(extra))
	result := make([]TemplateVariableInput, 0, len(base)+len(extra))
	for _, item := range append(base, extra...) {
		name := strings.TrimSpace(item.VariableName)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if strings.TrimSpace(item.VariableLabel) == "" {
			item.VariableLabel = name
		}
		if strings.TrimSpace(item.VariableType) == "" {
			item.VariableType = "text"
		}
		result = append(result, item)
	}
	return result
}

func summarizeChatGoal(goal, latest string) string {
	goal = strings.TrimSpace(goal)
	latest = strings.TrimSpace(latest)
	switch {
	case goal != "" && latest != "":
		return "已根据当前沟通目标生成一版模板草稿，核心需求为：" + goal + "。你补充的最新信息已纳入草稿：" + latest
	case goal != "":
		return "已根据当前沟通目标生成一版模板草稿，核心需求为：" + goal
	case latest != "":
		return "已根据你的最新补充生成一版模板草稿：" + latest
	default:
		return "已生成一版模板草稿，可继续补充需求并保存。"
	}
}

func fallbackHTMLFromParagraphs(paragraphs []string) string {
	if len(paragraphs) == 0 {
		return "<p>尊敬的 {{recipient_name}}：</p><p>您好。</p><p>现就 {{fund_name}} 相关事项向您发送正式通知，请查收。</p><p>{{notice_date}}</p>"
	}
	limit := len(paragraphs)
	if limit > 6 {
		limit = 6
	}
	parts := make([]string, 0, limit)
	for _, paragraph := range paragraphs[:limit] {
		text := strings.TrimSpace(paragraph)
		if text == "" {
			continue
		}
		parts = append(parts, "<p>"+escapeHTML(text)+"</p>")
	}
	if len(parts) == 0 {
		return "<p>尊敬的 {{recipient_name}}：</p><p>您好。</p><p>现就 {{fund_name}} 相关事项向您发送正式通知，请查收。</p><p>{{notice_date}}</p>"
	}
	return strings.Join(parts, "")
}

func firstNonEmptyKnownValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok || value == nil {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func escapeHTML(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(value)
}

func firstPersonName(items []basedto.PersonRef) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].Name
}

func firstPersonEmail(items []basedto.PersonRef) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].Email
}

func uint64ToString(value *uint64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatUint(*value, 10)
}

func derefUint64(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}

func sampleDOCXTemplate() []byte {
	buffer := &strings.Builder{}
	buffer.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buffer.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	buffer.WriteString(`<w:p><w:r><w:t>{{fund_name}}瀛ｅ害娌熼€氶€氱煡</w:t></w:r></w:p>`)
	buffer.WriteString(`<w:p><w:r><w:t>灏婃暚鐨剓{lp_name}}锛屾偍濂姐€?/w:t></w:r></w:p>`)
	buffer.WriteString(`<w:p><w:r><w:t>閫氱煡鏃ユ湡锛歿{notice_date}}</w:t></w:r></w:p>`)
	buffer.WriteString(`</w:body></w:document>`)
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`
	buf := &strings.Builder{}
	_ = buf
	var bytesBuf strings.Builder
	_ = bytesBuf
	out := &strings.Builder{}
	_ = out
	zipBuffer := &strings.Builder{}
	_ = zipBuffer
	mem := &strings.Builder{}
	_ = mem
	// Use archive/zip through helper package kept local in this file.
	byteBuffer := &strings.Builder{}
	_ = byteBuffer
	bufferZip := new(strings.Builder)
	_ = bufferZip
	// Small inline zip builder.
	return buildStaticDOCX(contentTypes, rels, docRels, buffer.String())
}

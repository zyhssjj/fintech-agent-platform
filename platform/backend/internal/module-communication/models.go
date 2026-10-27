package communication

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"wfagent/internal/base/database"
)

type MailboxAccount struct {
	database.BaseModel
	MailboxCode         string     `gorm:"column:mailbox_code;type:varchar(64);not null;uniqueIndex" json:"mailbox_code"`
	Name                string     `gorm:"column:name;type:varchar(128);not null" json:"name"`
	MailboxType         string     `gorm:"column:mailbox_type;type:varchar(32);not null;index" json:"mailbox_type"`
	OwnerUserID         *uint64    `gorm:"column:owner_user_id;index" json:"owner_user_id"`
	OwnerEmail          string     `gorm:"column:owner_email;type:varchar(128);not null;default:''" json:"owner_email"`
	EmailAddress        string     `gorm:"column:email_address;type:varchar(128);not null;index" json:"email_address"`
	AuthType            string     `gorm:"column:auth_type;type:varchar(32);not null" json:"auth_type"`
	SMTPHost            string     `gorm:"column:smtp_host;type:varchar(128);not null;default:''" json:"smtp_host"`
	SMTPPort            int        `gorm:"column:smtp_port;not null;default:587" json:"smtp_port"`
	IMAPHost            string     `gorm:"column:imap_host;type:varchar(128);not null;default:''" json:"imap_host"`
	IMAPPort            int        `gorm:"column:imap_port;not null;default:993" json:"imap_port"`
	Username            string     `gorm:"column:username;type:varchar(128);not null;default:''" json:"username"`
	PasswordSecret      string     `gorm:"column:password_secret;type:text;not null;default:''" json:"-"`
	OAuthProvider       string     `gorm:"column:oauth_provider;type:varchar(64);not null;default:''" json:"oauth_provider"`
	SignatureHTML       string     `gorm:"column:signature_html;type:text;not null;default:''" json:"signature_html"`
	IsDefaultSender     bool       `gorm:"column:is_default_sender;not null;default:false" json:"is_default_sender"`
	Status              string     `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
	LastSyncAt          *time.Time `gorm:"column:last_sync_at" json:"last_sync_at"`
	LastSyncStatus      string     `gorm:"column:last_sync_status;type:varchar(32);not null;default:''" json:"last_sync_status"`
	LastSyncMessage     string     `gorm:"column:last_sync_message;type:varchar(255);not null;default:''" json:"last_sync_message"`
	LastIMAPUID         uint64     `gorm:"column:last_imap_uid;not null;default:0" json:"last_imap_uid"`
	LastIMAPUIDValidity uint64     `gorm:"column:last_imap_uid_validity;not null;default:0" json:"last_imap_uid_validity"`
}

func (MailboxAccount) TableName() string { return "cm_mailbox_account" }

type MailboxOAuthToken struct {
	database.BaseModel
	MailboxID    uint64     `gorm:"column:mailbox_id;not null;uniqueIndex" json:"mailbox_id"`
	AccessToken  string     `gorm:"column:access_token;type:text;not null;default:''" json:"-"`
	RefreshToken string     `gorm:"column:refresh_token;type:text;not null;default:''" json:"-"`
	TokenType    string     `gorm:"column:token_type;type:varchar(32);not null;default:''" json:"token_type"`
	ExpiryAt     *time.Time `gorm:"column:expiry_at" json:"expiry_at"`
	Scope        string     `gorm:"column:scope;type:text;not null;default:''" json:"scope"`
}

func (MailboxOAuthToken) TableName() string { return "cm_mailbox_oauth_token" }

type Email struct {
	database.BaseModel
	EmailCode        string         `gorm:"column:email_code;type:varchar(64);not null;uniqueIndex" json:"email_code"`
	MailboxID        uint64         `gorm:"column:mailbox_id;not null;index" json:"mailbox_id"`
	Direction        string         `gorm:"column:direction;type:varchar(16);not null;index" json:"direction"`
	Folder           string         `gorm:"column:folder;type:varchar(32);not null;default:'inbox';index" json:"folder"`
	MessageID        string         `gorm:"column:message_id;type:varchar(255);not null;default:'';index" json:"message_id"`
	IMAPUID          uint64         `gorm:"column:imap_uid;not null;default:0;index" json:"imap_uid"`
	IMAPUIDValidity  uint64         `gorm:"column:imap_uid_validity;not null;default:0;index" json:"imap_uid_validity"`
	ThreadID         string         `gorm:"column:thread_id;type:varchar(128);not null;default:'';index" json:"thread_id"`
	Subject          string         `gorm:"column:subject;type:varchar(255);not null;default:''" json:"subject"`
	Preview          string         `gorm:"column:preview;type:text;not null;default:''" json:"preview"`
	BodyHTML         string         `gorm:"column:body_html;type:text;not null;default:''" json:"body_html"`
	SenderName       string         `gorm:"column:sender_name;type:varchar(128);not null;default:''" json:"sender_name"`
	SenderEmail      string         `gorm:"column:sender_email;type:varchar(128);not null;default:'';index" json:"sender_email"`
	ToRecipients     datatypes.JSON `gorm:"column:to_recipients;type:jsonb;not null;default:'[]'" json:"to_recipients"`
	CcRecipients     datatypes.JSON `gorm:"column:cc_recipients;type:jsonb;not null;default:'[]'" json:"cc_recipients"`
	BccRecipients    datatypes.JSON `gorm:"column:bcc_recipients;type:jsonb;not null;default:'[]'" json:"bcc_recipients"`
	Tags             datatypes.JSON `gorm:"column:tags;type:jsonb;not null;default:'[]'" json:"tags"`
	Summary          string         `gorm:"column:summary;type:text;not null;default:''" json:"summary"`
	ReplyDraft       string         `gorm:"column:reply_draft;type:text;not null;default:''" json:"reply_draft"`
	PriorityScore    int            `gorm:"column:priority_score;not null;default:0;index" json:"priority_score"`
	PriorityLabel    string         `gorm:"column:priority_label;type:varchar(32);not null;default:'normal';index" json:"priority_label"`
	NeedsReply       bool           `gorm:"column:needs_reply;not null;default:false;index" json:"needs_reply"`
	IsRead           bool           `gorm:"column:is_read;not null;default:false;index" json:"is_read"`
	IsStarred        bool           `gorm:"column:is_starred;not null;default:false;index" json:"is_starred"`
	HasAttachments   bool           `gorm:"column:has_attachments;not null;default:false" json:"has_attachments"`
	Status           string         `gorm:"column:status;type:varchar(32);not null;default:'ready';index" json:"status"`
	OperatorUserID   string         `gorm:"column:operator_user_id;type:varchar(64);not null;default:''" json:"operator_user_id"`
	OperatorUsername string         `gorm:"column:operator_username;type:varchar(128);not null;default:''" json:"operator_username"`
	ReceivedAt       *time.Time     `gorm:"column:received_at;index" json:"received_at"`
	SentAt           *time.Time     `gorm:"column:sent_at;index" json:"sent_at"`
}

func (Email) TableName() string { return "cm_email" }

type EmailAttachment struct {
	database.BaseModel
	EmailID     uint64 `gorm:"column:email_id;not null;index" json:"email_id"`
	FileName    string `gorm:"column:file_name;type:varchar(255);not null" json:"file_name"`
	ContentType string `gorm:"column:content_type;type:varchar(128);not null;default:''" json:"content_type"`
	FileSize    int64  `gorm:"column:file_size;not null;default:0" json:"file_size"`
	FileData    []byte `gorm:"column:file_data;type:bytea" json:"-"`
}

func (EmailAttachment) TableName() string { return "cm_email_attachment" }

type Template struct {
	database.BaseModel
	TemplateCode       string  `gorm:"column:template_code;type:varchar(64);not null;uniqueIndex" json:"template_code"`
	Name               string  `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Category           string  `gorm:"column:category;type:varchar(32);not null;index" json:"category"`
	Scene              string  `gorm:"column:scene;type:varchar(64);not null;default:'';index" json:"scene"`
	UsageBoundary      string  `gorm:"column:usage_boundary;type:text;not null;default:''" json:"usage_boundary"`
	TemplateSource     string  `gorm:"column:template_source;type:varchar(32);not null;default:'user_private';index" json:"template_source"`
	OwnerUserID        string  `gorm:"column:owner_user_id;type:varchar(64);not null;default:'';index" json:"owner_user_id"`
	SourceRefID        *uint64 `gorm:"column:source_ref_id;index" json:"source_ref_id"`
	IsEditable         bool    `gorm:"column:is_editable;not null;default:true" json:"is_editable"`
	AISessionID        string  `gorm:"column:ai_session_id;type:varchar(128);not null;default:''" json:"ai_session_id"`
	ImportFileName     string  `gorm:"column:import_file_name;type:varchar(255);not null;default:''" json:"import_file_name"`
	ParseStatus        string  `gorm:"column:parse_status;type:varchar(32);not null;default:'ready'" json:"parse_status"`
	Status             string  `gorm:"column:status;type:varchar(32);not null;default:'draft';index" json:"status"`
	PublishedVersionID *uint64 `gorm:"column:published_version_id;index" json:"published_version_id"`
	CreatorUserID      string  `gorm:"column:creator_user_id;type:varchar(64);not null;default:''" json:"creator_user_id"`
	CreatorName        string  `gorm:"column:creator_name;type:varchar(128);not null;default:''" json:"creator_name"`
}

func (Template) TableName() string { return "cm_template" }

type TemplateVersion struct {
	database.BaseModel
	TemplateID       uint64     `gorm:"column:template_id;not null;index" json:"template_id"`
	VersionNo        int        `gorm:"column:version_no;not null" json:"version_no"`
	SubjectTemplate  string     `gorm:"column:subject_template;type:varchar(255);not null;default:''" json:"subject_template"`
	HTMLContent      string     `gorm:"column:html_content;type:text;not null;default:''" json:"html_content"`
	DOCXTemplateName string     `gorm:"column:docx_template_name;type:varchar(255);not null;default:''" json:"docx_template_name"`
	DOCXTemplateData []byte     `gorm:"column:docx_template_data;type:bytea" json:"-"`
	Status           string     `gorm:"column:status;type:varchar(32);not null;default:'draft';index" json:"status"`
	ApprovalID       string     `gorm:"column:approval_id;type:varchar(64);not null;default:'';index" json:"approval_id"`
	SubmittedAt      *time.Time `gorm:"column:submitted_at" json:"submitted_at"`
	PublishedAt      *time.Time `gorm:"column:published_at" json:"published_at"`
}

func (TemplateVersion) TableName() string { return "cm_template_version" }

type TemplateVariable struct {
	database.BaseModel
	TemplateID    uint64 `gorm:"column:template_id;not null;index" json:"template_id"`
	VariableName  string `gorm:"column:variable_name;type:varchar(64);not null" json:"variable_name"`
	VariableLabel string `gorm:"column:variable_label;type:varchar(128);not null;default:''" json:"variable_label"`
	VariableType  string `gorm:"column:variable_type;type:varchar(32);not null;default:'text'" json:"variable_type"`
	IsRequired    bool   `gorm:"column:is_required;not null;default:false" json:"is_required"`
	DefaultValue  string `gorm:"column:default_value;type:text;not null;default:''" json:"default_value"`
}

func (TemplateVariable) TableName() string { return "cm_template_variable" }

type DocumentInstance struct {
	database.BaseModel
	DocumentCode       string         `gorm:"column:document_code;type:varchar(64);not null;uniqueIndex" json:"document_code"`
	TemplateID         uint64         `gorm:"column:template_id;not null;index" json:"template_id"`
	TemplateVersionID  uint64         `gorm:"column:template_version_id;not null;index" json:"template_version_id"`
	Title              string         `gorm:"column:title;type:varchar(255);not null" json:"title"`
	DocumentType       string         `gorm:"column:document_type;type:varchar(32);not null;index" json:"document_type"`
	RecipientName      string         `gorm:"column:recipient_name;type:varchar(128);not null;default:''" json:"recipient_name"`
	RecipientEmail     string         `gorm:"column:recipient_email;type:varchar(128);not null;default:'';index" json:"recipient_email"`
	CCRecipients       datatypes.JSON `gorm:"column:cc_recipients;type:jsonb;not null;default:'[]'" json:"cc_recipients"`
	SenderMailboxID    *uint64        `gorm:"column:sender_mailbox_id;index" json:"sender_mailbox_id"`
	SenderMailboxName  string         `gorm:"column:sender_mailbox_name;type:varchar(128);not null;default:''" json:"sender_mailbox_name"`
	SenderMailboxEmail string         `gorm:"column:sender_mailbox_email;type:varchar(128);not null;default:''" json:"sender_mailbox_email"`
	VisibleUserIDs     datatypes.JSON `gorm:"column:visible_user_ids;type:jsonb;not null;default:'[]'" json:"visible_user_ids"`
	VariableSnapshot   datatypes.JSON `gorm:"column:variable_snapshot;type:jsonb;not null;default:'{}'" json:"variable_snapshot"`
	HTMLContent        string         `gorm:"column:html_content;type:text;not null;default:''" json:"html_content"`
	DOCXFileName       string         `gorm:"column:docx_file_name;type:varchar(255);not null;default:''" json:"docx_file_name"`
	DOCXData           []byte         `gorm:"column:docx_data;type:bytea" json:"-"`
	Status             string         `gorm:"column:status;type:varchar(32);not null;default:'draft';index" json:"status"`
	ApprovalID         string         `gorm:"column:approval_id;type:varchar(64);not null;default:'';index" json:"approval_id"`
	SubmittedByUserID  string         `gorm:"column:submitted_by_user_id;type:varchar(64);not null;default:''" json:"submitted_by_user_id"`
	SubmittedByName    string         `gorm:"column:submitted_by_name;type:varchar(128);not null;default:''" json:"submitted_by_name"`
	SendResult         string         `gorm:"column:send_result;type:varchar(255);not null;default:''" json:"send_result"`
	SentAt             *time.Time     `gorm:"column:sent_at" json:"sent_at"`
}

func (DocumentInstance) TableName() string { return "cm_document_instance" }

type DocumentDelivery struct {
	database.BaseModel
	DocumentID     uint64     `gorm:"column:document_id;not null;index" json:"document_id"`
	Channel        string     `gorm:"column:channel;type:varchar(32);not null;default:'email'" json:"channel"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;default:'pending';index" json:"status"`
	MailboxID      *uint64    `gorm:"column:mailbox_id;index" json:"mailbox_id"`
	RecipientEmail string     `gorm:"column:recipient_email;type:varchar(128);not null;default:''" json:"recipient_email"`
	Message        string     `gorm:"column:message;type:varchar(255);not null;default:''" json:"message"`
	SentAt         *time.Time `gorm:"column:sent_at" json:"sent_at"`
}

func (DocumentDelivery) TableName() string { return "cm_document_delivery" }

type ManualSyncLog struct {
	database.BaseModel
	MailboxID     uint64     `gorm:"column:mailbox_id;not null;index" json:"mailbox_id"`
	TriggeredByID string     `gorm:"column:triggered_by_id;type:varchar(64);not null;default:''" json:"triggered_by_id"`
	TriggeredBy   string     `gorm:"column:triggered_by;type:varchar(128);not null;default:''" json:"triggered_by"`
	Status        string     `gorm:"column:status;type:varchar(32);not null;default:'success';index" json:"status"`
	Message       string     `gorm:"column:message;type:varchar(255);not null;default:''" json:"message"`
	SyncedCount   int        `gorm:"column:synced_count;not null;default:0" json:"synced_count"`
	StartedAt     time.Time  `gorm:"column:started_at;not null" json:"started_at"`
	FinishedAt    *time.Time `gorm:"column:finished_at" json:"finished_at"`
}

func (ManualSyncLog) TableName() string { return "cm_manual_sync_log" }

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(
		&MailboxAccount{},
		&MailboxOAuthToken{},
		&Email{},
		&EmailAttachment{},
		&Template{},
		&TemplateVersion{},
		&TemplateVariable{},
		&DocumentInstance{},
		&DocumentDelivery{},
		&ManualSyncLog{},
	)
}

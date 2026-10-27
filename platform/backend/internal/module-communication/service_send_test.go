package communication

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	basedto "wfagent/internal/base/dto"
	approval "wfagent/internal/module-approval"
)

type mockMailSender struct {
	err      error
	messages []outboundMessage
}

func (m *mockMailSender) Send(_ context.Context, _ MailboxAccount, message outboundMessage) error {
	m.messages = append(m.messages, message)
	return m.err
}

func TestSendEmailSMTPSuccess(t *testing.T) {
	db := openCommunicationTestDB(t)
	mailbox := createTestMailbox(t, db)
	sender := &mockMailSender{}
	service := newServiceWithMailSender(db, sender)

	result, err := service.SendEmail(context.Background(), SendEmailRequest{
		MailboxID: mailbox.ID,
		To: []basedto.PersonRef{
			{Name: "合作方", Email: "partner@example.com"},
		},
		Subject:  "外部合作沟通",
		BodyHTML: "<p>这是正式邮件</p>",
	}, "1", "ops_manager")
	if err != nil {
		t.Fatalf("send email: %v", err)
	}
	if got := result["status"]; got != "sent" {
		t.Fatalf("unexpected send email status: %v", got)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 outbound message, got %d", len(sender.messages))
	}
	if !strings.Contains(sender.messages[0].HTMLBody, mailbox.SignatureHTML) {
		t.Fatalf("expected signature to be appended")
	}

	var stored Email
	if err := db.Where("subject = ?", "外部合作沟通").First(&stored).Error; err != nil {
		t.Fatalf("query stored email: %v", err)
	}
	if stored.Status != "sent" {
		t.Fatalf("expected stored email status sent, got %s", stored.Status)
	}
}

func TestSendDocumentSMTPAndAttachmentSuccess(t *testing.T) {
	db := openCommunicationTestDB(t)
	mailbox := createTestMailbox(t, db)
	doc := createApprovedDocument(t, db, mailbox.ID)
	sender := &mockMailSender{}
	service := newServiceWithMailSender(db, sender)

	result, err := service.SendDocument(context.Background(), doc.DocumentCode, "1", "ops_manager")
	if err != nil {
		t.Fatalf("send document: %v", err)
	}
	if got := result["status"]; got != "sent" {
		t.Fatalf("unexpected send document status: %v", got)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 outbound document email, got %d", len(sender.messages))
	}
	if len(sender.messages[0].Attachments) != 1 {
		t.Fatalf("expected docx attachment to be sent")
	}

	var updated DocumentInstance
	if err := db.Where("document_code = ?", doc.DocumentCode).First(&updated).Error; err != nil {
		t.Fatalf("query updated document: %v", err)
	}
	if updated.Status != "sent" {
		t.Fatalf("expected sent document, got %s", updated.Status)
	}
	if updated.SendResult != "smtp_sent" {
		t.Fatalf("expected smtp_sent, got %s", updated.SendResult)
	}

	var attachments int64
	if err := db.Model(&EmailAttachment{}).Count(&attachments).Error; err != nil {
		t.Fatalf("count attachments: %v", err)
	}
	if attachments != 1 {
		t.Fatalf("expected 1 email attachment, got %d", attachments)
	}
}

func TestSummarizeHTMLDoesNotSplitUTF8(t *testing.T) {
	summary := summarizeHTML("<p>" + strings.Repeat("胜利基金与LP投资者战略合作通知", 30) + "</p>")
	if !utf8.ValidString(summary) {
		t.Fatalf("summary should remain valid UTF-8: %q", summary)
	}
	if !strings.HasSuffix(summary, "...") {
		t.Fatalf("expected truncated summary suffix, got %q", summary)
	}
}

func TestSendDocumentSMTPFailureKeepsApprovalState(t *testing.T) {
	db := openCommunicationTestDB(t)
	mailbox := createTestMailbox(t, db)
	doc := createApprovedDocument(t, db, mailbox.ID)
	service := newServiceWithMailSender(db, &mockMailSender{err: errors.New("smtp timeout")})

	if _, err := service.SendDocument(context.Background(), doc.DocumentCode, "1", "ops_manager"); err == nil {
		t.Fatalf("expected smtp failure")
	}

	var updated DocumentInstance
	if err := db.Where("document_code = ?", doc.DocumentCode).First(&updated).Error; err != nil {
		t.Fatalf("query updated document: %v", err)
	}
	if updated.Status != "approved" {
		t.Fatalf("expected document to stay approved after send failure, got %s", updated.Status)
	}
	if !strings.HasPrefix(updated.SendResult, "smtp_failed:") {
		t.Fatalf("expected smtp_failed marker, got %s", updated.SendResult)
	}

	var failedCount int64
	if err := db.Model(&DocumentDelivery{}).Where("document_id = ? AND status = ?", updated.ID, "failed").Count(&failedCount).Error; err != nil {
		t.Fatalf("count failed delivery: %v", err)
	}
	if failedCount != 1 {
		t.Fatalf("expected 1 failed delivery, got %d", failedCount)
	}
}

func TestSendDocumentRejectsNonInitiator(t *testing.T) {
	db := openCommunicationTestDB(t)
	mailbox := createTestMailbox(t, db)
	doc := createApprovedDocument(t, db, mailbox.ID)
	service := newServiceWithMailSender(db, &mockMailSender{})

	if _, err := service.SendDocument(context.Background(), doc.DocumentCode, "2", "legal_manager"); err == nil {
		t.Fatal("expected a non-initiator send to be rejected")
	}

	var updated DocumentInstance
	if err := db.Where("document_code = ?", doc.DocumentCode).First(&updated).Error; err != nil {
		t.Fatalf("query updated document: %v", err)
	}
	if updated.Status != "approved" {
		t.Fatalf("expected document to stay approved, got %s", updated.Status)
	}
}

func TestDocumentApprovalStatusTracksBothLevels(t *testing.T) {
	db := openCommunicationTestDB(t)
	if err := approval.SeedDefaults(db); err != nil {
		t.Fatalf("seed approval flows: %v", err)
	}
	mailbox := createTestMailbox(t, db)
	doc := createApprovedDocument(t, db, mailbox.ID)
	if err := db.Model(&DocumentInstance{}).Where("id = ?", doc.ID).Update("status", "draft").Error; err != nil {
		t.Fatalf("reset document status: %v", err)
	}
	service := newServiceWithMailSender(db, &mockMailSender{})

	submitted, err := service.SubmitDocument(context.Background(), doc.DocumentCode, "1", "initiator")
	if err != nil {
		t.Fatalf("submit document: %v", err)
	}
	if got := submitted["status"]; got != "pending_level_1" {
		t.Fatalf("expected pending_level_1 on submit, got %v", got)
	}
	approvalID := submitted["approval_id"].(string)
	if _, err := service.approvalService.ActionApproval(context.Background(), approvalID, approval.ApprovalActionRequest{Action: "approve"}, "2", "ops", []string{"OPS_MANAGER"}); err != nil {
		t.Fatalf("approve level 1: %v", err)
	}
	if err := service.syncDocumentApproval(context.Background()); err != nil {
		t.Fatalf("sync level 2 status: %v", err)
	}

	var afterLevel1 DocumentInstance
	if err := db.Where("id = ?", doc.ID).First(&afterLevel1).Error; err != nil {
		t.Fatalf("load document after level 1: %v", err)
	}
	if afterLevel1.Status != "pending_level_2" {
		t.Fatalf("expected pending_level_2 after level 1, got %s", afterLevel1.Status)
	}
	if _, err := service.approvalService.ActionApproval(context.Background(), approvalID, approval.ApprovalActionRequest{Action: "approve"}, "3", "legal", []string{"LEGAL_MANAGER"}); err != nil {
		t.Fatalf("approve level 2: %v", err)
	}
	if err := service.syncDocumentApproval(context.Background()); err != nil {
		t.Fatalf("sync approved status: %v", err)
	}

	var afterLevel2 DocumentInstance
	if err := db.Where("id = ?", doc.ID).First(&afterLevel2).Error; err != nil {
		t.Fatalf("load document after level 2: %v", err)
	}
	if afterLevel2.Status != "approved" {
		t.Fatalf("expected approved after level 2, got %s", afterLevel2.Status)
	}
}

func TestSendDocumentFallsBackToManualDeliveryWhenSMTPIsPlaceholder(t *testing.T) {
	db := openCommunicationTestDB(t)
	mailbox := createTestMailbox(t, db)
	mailbox.SMTPHost = "smtp.example.com"
	mailbox.EmailAddress = "ops@fintechbase.local"
	mailbox.PasswordSecret = ""
	if err := db.Model(&MailboxAccount{}).Where("id = ?", mailbox.ID).Updates(map[string]any{
		"smtp_host":       mailbox.SMTPHost,
		"email_address":   mailbox.EmailAddress,
		"password_secret": mailbox.PasswordSecret,
	}).Error; err != nil {
		t.Fatalf("update mailbox: %v", err)
	}
	doc := createApprovedDocument(t, db, mailbox.ID)
	sender := &mockMailSender{}
	service := newServiceWithMailSender(db, sender)

	result, err := service.SendDocument(context.Background(), doc.DocumentCode, "1", "ops_manager")
	if err != nil {
		t.Fatalf("send document manual fallback: %v", err)
	}
	if got := result["send_result"]; got != "manual_external_delivery" {
		t.Fatalf("unexpected manual send result: %v", got)
	}
	if len(sender.messages) != 0 {
		t.Fatalf("manual fallback should not call smtp sender")
	}

	var updated DocumentInstance
	if err := db.Where("document_code = ?", doc.DocumentCode).First(&updated).Error; err != nil {
		t.Fatalf("query updated document: %v", err)
	}
	if updated.Status != "sent" {
		t.Fatalf("expected sent status after manual fallback, got %s", updated.Status)
	}
	if updated.SendResult != "manual_external_delivery" {
		t.Fatalf("expected manual_external_delivery, got %s", updated.SendResult)
	}

	var delivery DocumentDelivery
	if err := db.Where("document_id = ?", updated.ID).Order("id desc").First(&delivery).Error; err != nil {
		t.Fatalf("query delivery: %v", err)
	}
	if delivery.Status != "manual_delivery" {
		t.Fatalf("expected manual_delivery status, got %s", delivery.Status)
	}
}

func TestMicrosoftGraphMailboxDoesNotUseManualDelivery(t *testing.T) {
	mailbox := MailboxAccount{
		EmailAddress:  "outlook@example.com",
		AuthType:      "oauth",
		OAuthProvider: "microsoft_graph",
	}
	if shouldUseManualDelivery(mailbox) {
		t.Fatal("Microsoft Graph mailbox must not fall back to manual delivery")
	}
}

func TestSendDocumentRejectsMissingFormalDocx(t *testing.T) {
	db := openCommunicationTestDB(t)
	mailbox := createTestMailbox(t, db)
	doc := createApprovedDocument(t, db, mailbox.ID)
	if err := db.Model(&DocumentInstance{}).Where("id = ?", doc.ID).Updates(map[string]any{
		"docx_file_name": "",
		"docx_data":      []byte{},
	}).Error; err != nil {
		t.Fatalf("clear document docx: %v", err)
	}

	service := newServiceWithMailSender(db, &mockMailSender{})
	if _, err := service.SendDocument(context.Background(), doc.DocumentCode, "1", "ops_manager"); err == nil {
		t.Fatalf("expected missing docx error")
	}
}

func TestDownloadDocumentRejectsMissingFormalDocx(t *testing.T) {
	db := openCommunicationTestDB(t)
	mailbox := createTestMailbox(t, db)
	doc := createApprovedDocument(t, db, mailbox.ID)
	if err := db.Model(&DocumentInstance{}).Where("id = ?", doc.ID).Updates(map[string]any{
		"docx_file_name": "",
		"docx_data":      []byte{},
	}).Error; err != nil {
		t.Fatalf("clear document docx: %v", err)
	}

	service := newServiceWithMailSender(db, &mockMailSender{})
	if _, _, err := service.DownloadDocument(context.Background(), doc.DocumentCode); err == nil {
		t.Fatalf("expected missing docx error")
	}
}

func openCommunicationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := approval.AutoMigrate(db); err != nil {
		t.Fatalf("approval migrate: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("communication migrate: %v", err)
	}
	return db
}

func createTestMailbox(t *testing.T, db *gorm.DB) MailboxAccount {
	t.Helper()
	mailbox := MailboxAccount{
		MailboxCode:     "MBX-" + uuid.NewString()[:8],
		Name:            "基金运营共享邮箱",
		MailboxType:     "shared",
		EmailAddress:    "ops@example.com",
		AuthType:        "password",
		SMTPHost:        "smtp.partnermail.test",
		SMTPPort:        465,
		Username:        "ops@example.com",
		PasswordSecret:  "auth-code",
		SignatureHTML:   "<p>基金运营共享邮箱</p>",
		Status:          "active",
		IsDefaultSender: true,
	}
	if err := db.Create(&mailbox).Error; err != nil {
		t.Fatalf("create mailbox: %v", err)
	}
	return mailbox
}

func createApprovedDocument(t *testing.T, db *gorm.DB, mailboxID uint64) DocumentInstance {
	t.Helper()
	document := DocumentInstance{
		DocumentCode:       "DOC-" + uuid.NewString()[:8],
		TemplateID:         1,
		TemplateVersionID:  1,
		Title:              "外部合作方通知函",
		DocumentType:       "notice",
		RecipientName:      "合作方",
		RecipientEmail:     "partner@example.com",
		CCRecipients:       toRecipients([]basedto.PersonRef{{Name: "抄送人", Email: "cc@example.com"}}),
		SenderMailboxID:    &mailboxID,
		SenderMailboxName:  "基金运营共享邮箱",
		SenderMailboxEmail: "ops@example.com",
		HTMLContent:        "<p>合作通知正文</p>",
		DOCXFileName:       "notice.docx",
		DOCXData:           sampleDOCXTemplate(),
		Status:             "approved",
		SubmittedByUserID:  "1",
		SubmittedByName:    "ops_manager",
	}
	if err := db.Create(&document).Error; err != nil {
		t.Fatalf("create approved document: %v", err)
	}
	return document
}

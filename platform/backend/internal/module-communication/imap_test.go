package communication

import (
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestParseInboundMessagePrefersHTMLAndKeepsUTF8(t *testing.T) {
	raw := strings.Join([]string{
		"Message-ID: <imap-html-1@example.com>",
		"Subject: =?UTF-8?B?5Lit5paH6YCa55+l?=",
		"From: =?UTF-8?B?5ZCI5L2c5pa5?= <partner@example.com>",
		"To: ops@example.com",
		"Date: Tue, 28 Jul 2026 10:00:00 +0800",
		"Content-Type: multipart/alternative; boundary=alt",
		"",
		"--alt",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"纯文本正文",
		"--alt",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<p>HTML 正文</p>",
		"--alt--",
		"",
	}, "\r\n")
	parsed, err := parseInboundMessage(imapFetchedMessage{
		UID:         11,
		UIDValidity: 22,
		Flags:       []string{"\\Seen"},
		Raw:         []byte(raw),
	})
	if err != nil {
		t.Fatalf("parse inbound message: %v", err)
	}
	if parsed.Subject != "中文通知" {
		t.Fatalf("subject mismatch: %q", parsed.Subject)
	}
	if parsed.BodyHTML != "<p>HTML 正文</p>" {
		t.Fatalf("html body mismatch: %q", parsed.BodyHTML)
	}
	if !parsed.IsRead {
		t.Fatalf("expected remote Seen flag to map to local read state")
	}
}

func TestParseInboundMessagePlainTextToHTMLAndAttachment(t *testing.T) {
	raw := strings.Join([]string{
		"Message-ID: <imap-attach-1@example.com>",
		"Subject: Attachment",
		"From: partner@example.com",
		"To: ops@example.com",
		"Content-Type: multipart/mixed; boundary=mix",
		"",
		"--mix",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"第一行",
		"第二行",
		"--mix",
		"Content-Type: text/plain; name=\"note.txt\"",
		"Content-Disposition: attachment; filename=\"note.txt\"",
		"Content-Transfer-Encoding: base64",
		"",
		"5rWL6K+V6ZmE5Lu2",
		"--mix--",
		"",
	}, "\r\n")
	parsed, err := parseInboundMessage(imapFetchedMessage{UID: 12, UIDValidity: 22, Raw: []byte(raw)})
	if err != nil {
		t.Fatalf("parse inbound message: %v", err)
	}
	if parsed.BodyHTML != "<p>第一行<br/>第二行</p>" {
		t.Fatalf("plain text html mismatch: %q", parsed.BodyHTML)
	}
	if len(parsed.Attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(parsed.Attachments))
	}
	if string(parsed.Attachments[0].Data) != "测试附件" {
		t.Fatalf("attachment data mismatch: %q", string(parsed.Attachments[0].Data))
	}
}

func TestCreateInboundEmailFromIMAPDedupesByUIDAndMessageID(t *testing.T) {
	db := openCommunicationTestDB(t)
	service := NewService(db)
	box := createIMAPTestMailbox(t, db)
	now := time.Now()
	item := imapFetchedMessage{
		UID:          88,
		UIDValidity:  99,
		InternalDate: &now,
		Raw: []byte(strings.Join([]string{
			"Message-ID: <dedupe@example.com>",
			"Subject: Dedupe",
			"From: partner@example.com",
			"To: ops@example.com",
			"Content-Type: text/plain; charset=utf-8",
			"",
			"hello",
			"",
		}, "\r\n")),
	}
	created, err := service.createInboundEmailFromIMAP(context.Background(), db, box, item, "1", "tester")
	if err != nil || !created {
		t.Fatalf("first create = %v, %v", created, err)
	}
	created, err = service.createInboundEmailFromIMAP(context.Background(), db, box, item, "1", "tester")
	if err != nil {
		t.Fatalf("second create error: %v", err)
	}
	if created {
		t.Fatalf("expected duplicate UID to be skipped")
	}
	item.UID = 0
	item.UIDValidity = 0
	created, err = service.createInboundEmailFromIMAP(context.Background(), db, box, item, "1", "tester")
	if err != nil {
		t.Fatalf("message id dedupe error: %v", err)
	}
	if created {
		t.Fatalf("expected duplicate Message-ID to be skipped")
	}
	var count int64
	if err := db.Model(&Email{}).Where("mailbox_id = ?", box.ID).Count(&count).Error; err != nil {
		t.Fatalf("count emails: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one stored email, got %d", count)
	}
}

func createIMAPTestMailbox(t *testing.T, db *gorm.DB) MailboxAccount {
	t.Helper()
	box := MailboxAccount{
		MailboxCode:    "IMAP-TEST",
		Name:           "IMAP 测试邮箱",
		MailboxType:    "shared",
		EmailAddress:   "ops@example.com",
		AuthType:       "password",
		IMAPHost:       "imap.example.com",
		IMAPPort:       993,
		Username:       "ops@example.com",
		PasswordSecret: "auth-code",
		Status:         "active",
	}
	if err := db.Create(&box).Error; err != nil {
		t.Fatalf("create imap mailbox: %v", err)
	}
	return box
}

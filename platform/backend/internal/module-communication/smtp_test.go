package communication

import (
	"strings"
	"testing"
)

func TestBuildSMTPMessageNormalizesPlainTextAndUsesSenderDomain(t *testing.T) {
	message, err := buildSMTPMessage(outboundMessage{
		FromAddress: "sender@example.com",
		To:          nil,
		Subject:     "测试邮件",
		HTMLBody:    "第一段\n\n第二段",
	})
	if err != nil {
		t.Fatalf("build SMTP message: %v", err)
	}
	content := string(message)
	if !strings.Contains(content, "Message-ID: <") || !strings.Contains(content, "@example.com>") {
		t.Fatalf("expected Message-ID to use sender domain: %s", content)
	}
	if !strings.Contains(content, "<p>第一段</p>") || !strings.Contains(content, "<p>第二段</p>") {
		t.Fatalf("expected plain text HTML body to be normalized: %s", content)
	}
}

func TestValidateExternalDocumentContentRejectsUnresolvedSenderPlaceholders(t *testing.T) {
	err := validateExternalDocumentContent("正式函件", "<p>[发件人姓名]</p>")
	if err == nil || !strings.Contains(err.Error(), "未填写") {
		t.Fatalf("expected unresolved sender placeholder error, got %v", err)
	}
}

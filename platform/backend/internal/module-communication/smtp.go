package communication

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"sort"
	"strings"
	"time"

	basedto "wfagent/internal/base/dto"
)

type outboundAttachment struct {
	FileName    string
	ContentType string
	Data        []byte
}

type outboundMessage struct {
	FromName    string
	FromAddress string
	To          []basedto.PersonRef
	CC          []basedto.PersonRef
	BCC         []basedto.PersonRef
	Subject     string
	HTMLBody    string
	Attachments []outboundAttachment
}

type mailSender interface {
	Send(ctx context.Context, mailbox MailboxAccount, message outboundMessage) error
}

type smtpMailSender struct {
	timeout time.Duration
}

func newSMTPMailSender() mailSender {
	return &smtpMailSender{timeout: 30 * time.Second}
}

func (s *smtpMailSender) Send(ctx context.Context, mailbox MailboxAccount, message outboundMessage) error {
	if err := validateOutboundMessage(mailbox, message); err != nil {
		return err
	}
	address := net.JoinHostPort(mailbox.SMTPHost, fmt.Sprintf("%d", mailbox.SMTPPort))
	dialer := &net.Dialer{Timeout: s.timeout}

	var (
		client *smtp.Client
		err    error
	)
	if mailbox.SMTPPort == 465 {
		tlsConn, tlsErr := dialSMTPTLS(ctx, dialer, mailbox.SMTPHost, mailbox.SMTPPort)
		if tlsErr != nil {
			return fmt.Errorf("connect smtp tls: %w", tlsErr)
		}
		client, err = smtp.NewClient(tlsConn, mailbox.SMTPHost)
		if err != nil {
			_ = tlsConn.Close()
			return fmt.Errorf("create smtp client: %w", err)
		}
	} else {
		conn, dialErr := dialer.DialContext(ctx, "tcp", address)
		if dialErr != nil {
			return fmt.Errorf("connect smtp: %w", dialErr)
		}
		client, err = smtp.NewClient(conn, mailbox.SMTPHost)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("create smtp client: %w", err)
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{
				ServerName: mailbox.SMTPHost,
				MinVersion: tls.VersionTLS12,
			}); err != nil {
				_ = client.Close()
				return fmt.Errorf("starttls: %w", err)
			}
		}
	}
	defer client.Close()

	username := strings.TrimSpace(mailbox.Username)
	if username == "" {
		username = strings.TrimSpace(mailbox.EmailAddress)
	}
	if strings.EqualFold(mailbox.AuthType, "password") {
		if err := client.Auth(smtp.PlainAuth("", username, mailbox.PasswordSecret, mailbox.SMTPHost)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(message.FromAddress); err != nil {
		return fmt.Errorf("smtp from: %w", err)
	}
	for _, recipient := range collectRecipients(message) {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp recipient %s: %w", recipient, err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	content, err := buildSMTPMessage(message)
	if err != nil {
		_ = writer.Close()
		return err
	}
	if _, err := writer.Write(content); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp finalize: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func dialSMTPTLS(ctx context.Context, dialer *net.Dialer, host string, port int) (*tls.Conn, error) {
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 {
		return tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	}
	// Some consumer SMTP providers advertise IPv6 endpoints that accept TCP but
	// never complete TLS. Prefer IPv4, while retaining hostname certificate checks.
	sort.SliceStable(addresses, func(i, j int) bool {
		return addresses[i].IP.To4() != nil && addresses[j].IP.To4() == nil
	})
	var lastErr error
	for _, address := range addresses {
		conn, dialErr := dialer.DialContext(ctx, "tcp", net.JoinHostPort(address.IP.String(), fmt.Sprintf("%d", port)))
		if dialErr != nil {
			lastErr = dialErr
			continue
		}
		tlsConn := tls.Client(conn, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		handshakeCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		handshakeErr := tlsConn.HandshakeContext(handshakeCtx)
		cancel()
		if handshakeErr == nil {
			return tlsConn, nil
		}
		_ = tlsConn.Close()
		lastErr = handshakeErr
	}
	return nil, fmt.Errorf("smtp TLS handshake: %w", lastErr)
}

func validateOutboundMessage(mailbox MailboxAccount, message outboundMessage) error {
	if strings.TrimSpace(mailbox.SMTPHost) == "" || mailbox.SMTPPort <= 0 {
		return fmt.Errorf("mailbox %s smtp config is incomplete", mailbox.EmailAddress)
	}
	if strings.EqualFold(mailbox.AuthType, "password") && strings.TrimSpace(mailbox.PasswordSecret) == "" {
		return fmt.Errorf("mailbox %s password or auth code is missing", mailbox.EmailAddress)
	}
	if len(collectRecipients(message)) == 0 {
		return fmt.Errorf("no outbound recipients")
	}
	return nil
}

func collectRecipients(message outboundMessage) []string {
	seen := map[string]struct{}{}
	recipients := make([]string, 0, len(message.To)+len(message.CC)+len(message.BCC))
	for _, item := range append(append(message.To, message.CC...), message.BCC...) {
		email := strings.TrimSpace(item.Email)
		if email == "" {
			continue
		}
		key := strings.ToLower(email)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		recipients = append(recipients, email)
	}
	return recipients
}

func buildSMTPMessage(message outboundMessage) ([]byte, error) {
	message.HTMLBody = normalizeOutboundHTML(message.HTMLBody)
	buffer := &bytes.Buffer{}
	writeSMTPHeader(buffer, "MIME-Version", "1.0")
	writeSMTPHeader(buffer, "Date", time.Now().Format(time.RFC1123Z))
	writeSMTPHeader(buffer, "Message-ID", fmt.Sprintf("<%d.fintechbase@%s>", time.Now().UnixNano(), messageIDDomain(message.FromAddress)))
	writeSMTPHeader(buffer, "From", formatAddress(message.FromName, message.FromAddress))
	writeSMTPHeader(buffer, "Reply-To", message.FromAddress)
	writeSMTPHeader(buffer, "To", formatAddressList(message.To))
	if len(message.CC) > 0 {
		writeSMTPHeader(buffer, "Cc", formatAddressList(message.CC))
	}
	writeSMTPHeader(buffer, "Subject", mime.QEncoding.Encode("UTF-8", message.Subject))

	if len(message.Attachments) == 0 {
		writeSMTPHeader(buffer, "Content-Type", `text/html; charset="UTF-8"`)
		writeSMTPHeader(buffer, "Content-Transfer-Encoding", "8bit")
		buffer.WriteString("\r\n")
		buffer.WriteString(message.HTMLBody)
		return buffer.Bytes(), nil
	}

	mixedWriter := multipart.NewWriter(buffer)
	writeSMTPHeader(buffer, "Content-Type", `multipart/mixed; boundary="`+mixedWriter.Boundary()+`"`)
	buffer.WriteString("\r\n")

	htmlHeader := textprotoMIMEHeader(map[string]string{
		"Content-Type":              `text/html; charset="UTF-8"`,
		"Content-Transfer-Encoding": "8bit",
	})
	htmlPart, err := mixedWriter.CreatePart(htmlHeader)
	if err != nil {
		return nil, fmt.Errorf("create html part: %w", err)
	}
	if _, err := htmlPart.Write([]byte(message.HTMLBody)); err != nil {
		return nil, fmt.Errorf("write html part: %w", err)
	}

	for _, attachment := range message.Attachments {
		contentType := strings.TrimSpace(attachment.ContentType)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		partHeader := textprotoMIMEHeader(map[string]string{
			"Content-Type":              contentType + `; name="` + attachment.FileName + `"`,
			"Content-Disposition":       `attachment; filename="` + attachment.FileName + `"`,
			"Content-Transfer-Encoding": "base64",
		})
		part, err := mixedWriter.CreatePart(partHeader)
		if err != nil {
			return nil, fmt.Errorf("create attachment part: %w", err)
		}
		writeBase64Lines(part, attachment.Data)
	}
	if err := mixedWriter.Close(); err != nil {
		return nil, fmt.Errorf("close multipart message: %w", err)
	}
	return buffer.Bytes(), nil
}

func messageIDDomain(address string) string {
	parts := strings.Split(strings.TrimSpace(address), "@")
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[1])
	}
	return "fintechbase.local"
}

func normalizeOutboundHTML(value string) string {
	value = cleanUTF8(strings.TrimSpace(value))
	if value == "" || (strings.Contains(value, "<") && strings.Contains(value, ">")) {
		return value
	}
	return plainTextToHTML(value)
}

func writeSMTPHeader(buffer *bytes.Buffer, key, value string) {
	buffer.WriteString(key)
	buffer.WriteString(": ")
	buffer.WriteString(value)
	buffer.WriteString("\r\n")
}

func formatAddressList(items []basedto.PersonRef) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		email := strings.TrimSpace(item.Email)
		if email == "" {
			continue
		}
		parts = append(parts, formatAddress(item.Name, email))
	}
	return strings.Join(parts, ", ")
}

func formatAddress(name, email string) string {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)
	if name == "" {
		return email
	}
	return mime.QEncoding.Encode("UTF-8", name) + " <" + email + ">"
}

func textprotoMIMEHeader(values map[string]string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader, len(values))
	for key, value := range values {
		header[key] = []string{value}
	}
	return header
}

func writeBase64Lines(writer io.Writer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		_, _ = io.WriteString(writer, encoded[:76])
		_, _ = io.WriteString(writer, "\r\n")
		encoded = encoded[76:]
	}
	if encoded != "" {
		_, _ = io.WriteString(writer, encoded)
		_, _ = io.WriteString(writer, "\r\n")
	}
}

package communication

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	basedto "wfagent/internal/base/dto"
)

const (
	imapInitialFetchLimit     = 50
	imapAttachmentTotalLimitB = 20 * 1024 * 1024
	imapNetworkTimeout        = 30 * time.Second
)

var (
	imapLiteralPattern      = regexp.MustCompile(`\{(\d+)\}\s*$`)
	imapUIDPattern          = regexp.MustCompile(`(?i)\bUID\s+(\d+)`)
	imapUIDValidityPattern  = regexp.MustCompile(`(?i)UIDVALIDITY\s+(\d+)`)
	imapInternalDatePattern = regexp.MustCompile(`(?i)INTERNALDATE\s+"([^"]+)"`)
)

type imapFetchedMessage struct {
	UID          uint64
	UIDValidity  uint64
	Flags        []string
	InternalDate *time.Time
	Raw          []byte
}

type parsedInboundMessage struct {
	MessageID   string
	ThreadID    string
	Subject     string
	BodyHTML    string
	SenderName  string
	SenderEmail string
	To          []basedto.PersonRef
	CC          []basedto.PersonRef
	Attachments []inboundAttachment
	ReceivedAt  *time.Time
	IsRead      bool
}

type inboundAttachment struct {
	FileName    string
	ContentType string
	Data        []byte
}

type imapClient struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	tagSeq int
}

func fetchMailboxFromIMAP(ctx context.Context, mailbox MailboxAccount) ([]imapFetchedMessage, uint64, error) {
	client, err := dialIMAP(ctx, mailbox)
	if err != nil {
		return nil, 0, err
	}
	defer client.close()

	username := strings.TrimSpace(mailbox.Username)
	if username == "" {
		username = strings.TrimSpace(mailbox.EmailAddress)
	}
	if err := client.login(username, mailbox.PasswordSecret); err != nil {
		return nil, 0, err
	}
	uidValidity, err := client.selectInbox()
	if err != nil {
		return nil, 0, err
	}
	uids, err := client.searchUIDs(mailbox.LastIMAPUID, mailbox.LastIMAPUIDValidity, uidValidity)
	if err != nil {
		return nil, uidValidity, err
	}
	if len(uids) == 0 {
		return []imapFetchedMessage{}, uidValidity, nil
	}
	if mailbox.LastIMAPUID == 0 || mailbox.LastIMAPUIDValidity != uidValidity {
		sort.Slice(uids, func(i, j int) bool { return uids[i] < uids[j] })
		if len(uids) > imapInitialFetchLimit {
			uids = uids[len(uids)-imapInitialFetchLimit:]
		}
	}
	items, err := client.fetchMessages(uids, uidValidity)
	if err != nil {
		return nil, uidValidity, err
	}
	return items, uidValidity, nil
}

func dialIMAP(ctx context.Context, mailbox MailboxAccount) (*imapClient, error) {
	host := strings.TrimSpace(mailbox.IMAPHost)
	if host == "" || mailbox.IMAPPort <= 0 {
		return nil, fmt.Errorf("mailbox %s imap config is incomplete", mailbox.EmailAddress)
	}
	dialer := &net.Dialer{Timeout: imapNetworkTimeout}
	address := net.JoinHostPort(host, strconv.Itoa(mailbox.IMAPPort))
	var (
		conn net.Conn
		err  error
	)
	if mailbox.IMAPPort == 993 {
		conn, err = tls.DialWithDialer(dialer, "tcp4", address, &tls.Config{ServerName: host})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp4", address)
	}
	if err != nil {
		return nil, fmt.Errorf("connect imap: %w", err)
	}
	client := &imapClient{conn: conn, reader: bufio.NewReader(conn), writer: bufio.NewWriter(conn)}
	if err := client.readGreeting(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if mailbox.IMAPPort != 993 {
		supportsStartTLS, err := client.supportsStartTLS()
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		if supportsStartTLS {
			if err := client.startTLS(host); err != nil {
				_ = conn.Close()
				return nil, err
			}
		}
	}
	return client, nil
}

func (c *imapClient) readGreeting() error {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read imap greeting: %w", err)
	}
	if !strings.Contains(strings.ToUpper(line), "OK") {
		return fmt.Errorf("imap greeting failed: %s", strings.TrimSpace(line))
	}
	return nil
}

func (c *imapClient) startTLS(host string) error {
	if _, err := c.command("STARTTLS"); err != nil {
		return err
	}
	tlsConn := tls.Client(c.conn, &tls.Config{ServerName: host})
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("imap starttls handshake: %w", err)
	}
	c.conn = tlsConn
	c.reader = bufio.NewReader(tlsConn)
	c.writer = bufio.NewWriter(tlsConn)
	return nil
}

func (c *imapClient) supportsStartTLS() (bool, error) {
	lines, err := c.command("CAPABILITY")
	if err != nil {
		return false, err
	}
	for _, line := range lines {
		if strings.HasPrefix(strings.ToUpper(line), "* CAPABILITY") && strings.Contains(strings.ToUpper(line), "STARTTLS") {
			return true, nil
		}
	}
	return false, nil
}

func (c *imapClient) login(username, password string) error {
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("imap password or auth code is missing")
	}
	_, err := c.command("LOGIN %s %s", imapQuote(username), imapQuote(password))
	return err
}

func (c *imapClient) selectInbox() (uint64, error) {
	lines, err := c.command("SELECT INBOX")
	if err != nil {
		return 0, err
	}
	for _, line := range lines {
		match := imapUIDValidityPattern.FindStringSubmatch(line)
		if len(match) == 2 {
			value, _ := strconv.ParseUint(match[1], 10, 64)
			return value, nil
		}
	}
	return 0, fmt.Errorf("imap uidvalidity is missing")
}

func (c *imapClient) searchUIDs(lastUID, lastUIDValidity, uidValidity uint64) ([]uint64, error) {
	query := "UID SEARCH ALL"
	if lastUID > 0 && lastUIDValidity == uidValidity {
		query = fmt.Sprintf("UID SEARCH UID %d:*", lastUID+1)
	}
	lines, err := c.commandRaw(query)
	if err != nil {
		return nil, err
	}
	seen := map[uint64]struct{}{}
	var uids []uint64
	for _, line := range lines {
		if !strings.HasPrefix(line, "* SEARCH") {
			continue
		}
		for _, token := range strings.Fields(strings.TrimPrefix(line, "* SEARCH")) {
			value, err := strconv.ParseUint(token, 10, 64)
			if err != nil || value == 0 {
				continue
			}
			if lastUID > 0 && lastUIDValidity == uidValidity && value <= lastUID {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			uids = append(uids, value)
		}
	}
	sort.Slice(uids, func(i, j int) bool { return uids[i] < uids[j] })
	return uids, nil
}

func (c *imapClient) fetchMessages(uids []uint64, uidValidity uint64) ([]imapFetchedMessage, error) {
	if len(uids) == 0 {
		return nil, nil
	}
	parts := make([]string, 0, len(uids))
	for _, uid := range uids {
		parts = append(parts, strconv.FormatUint(uid, 10))
	}
	lines, literals, err := c.commandWithLiterals("UID FETCH " + strings.Join(parts, ",") + " (UID FLAGS INTERNALDATE BODY.PEEK[])")
	if err != nil {
		return nil, err
	}
	items := make([]imapFetchedMessage, 0, len(literals))
	literalIndex := 0
	for _, line := range lines {
		if !strings.Contains(strings.ToUpper(line), "FETCH") || !imapLiteralPattern.MatchString(line) {
			continue
		}
		if literalIndex >= len(literals) {
			break
		}
		uid := parseIMAPUint(line, imapUIDPattern)
		if uid == 0 {
			uid = uids[min(literalIndex, len(uids)-1)]
		}
		items = append(items, imapFetchedMessage{
			UID:          uid,
			UIDValidity:  uidValidity,
			Flags:        parseIMAPFlags(line),
			InternalDate: parseIMAPInternalDate(line),
			Raw:          literals[literalIndex],
		})
		literalIndex++
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UID < items[j].UID })
	return items, nil
}

func (c *imapClient) command(format string, args ...any) ([]string, error) {
	lines, _, err := c.commandWithLiterals(fmt.Sprintf(format, args...))
	return lines, err
}

func (c *imapClient) commandRaw(command string) ([]string, error) {
	lines, _, err := c.commandWithLiterals(command)
	return lines, err
}

func (c *imapClient) commandWithLiterals(command string) ([]string, [][]byte, error) {
	c.tagSeq++
	tag := fmt.Sprintf("A%04d", c.tagSeq)
	if _, err := c.writer.WriteString(tag + " " + command + "\r\n"); err != nil {
		return nil, nil, err
	}
	if err := c.writer.Flush(); err != nil {
		return nil, nil, err
	}
	var lines []string
	var literals [][]byte
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return lines, literals, err
		}
		line = strings.TrimRight(line, "\r\n")
		lines = append(lines, line)
		if match := imapLiteralPattern.FindStringSubmatch(line); len(match) == 2 {
			size, _ := strconv.Atoi(match[1])
			if size > 0 {
				buf := make([]byte, size)
				if _, err := io.ReadFull(c.reader, buf); err != nil {
					return lines, literals, err
				}
				literals = append(literals, buf)
			}
		}
		if strings.HasPrefix(line, tag+" ") {
			upper := strings.ToUpper(line)
			if strings.Contains(upper, " OK") {
				return lines, literals, nil
			}
			return lines, literals, fmt.Errorf("imap command failed: %s", line)
		}
	}
}

func (c *imapClient) close() {
	_, _ = c.command("LOGOUT")
	_ = c.conn.Close()
}

func parseInboundMessage(item imapFetchedMessage) (parsedInboundMessage, error) {
	message, err := mail.ReadMessage(bytes.NewReader(item.Raw))
	if err != nil {
		return parsedInboundMessage{}, fmt.Errorf("parse mail message: %w", err)
	}
	header := message.Header
	subject, _ := (&mime.WordDecoder{}).DecodeHeader(header.Get("Subject"))
	subject = cleanUTF8(subject)
	from := parseAddressHeader(header.Get("From"))
	to := parseAddressListHeader(header.Get("To"))
	cc := parseAddressListHeader(header.Get("Cc"))
	receivedAt := item.InternalDate
	if dateHeader := header.Get("Date"); dateHeader != "" {
		if parsed, err := mail.ParseDate(dateHeader); err == nil {
			receivedAt = &parsed
		}
	}
	bodyHTML, attachments, err := parseMIMEBody(textproto.MIMEHeader(header), message.Body)
	if err != nil {
		return parsedInboundMessage{}, err
	}
	if strings.TrimSpace(bodyHTML) == "" {
		bodyHTML = "<p></p>"
	}
	messageID := cleanUTF8(strings.Trim(header.Get("Message-Id"), "<>"))
	if messageID == "" {
		messageID = cleanUTF8(strings.Trim(header.Get("Message-ID"), "<>"))
	}
	return parsedInboundMessage{
		MessageID:   messageID,
		ThreadID:    messageID,
		Subject:     subject,
		BodyHTML:    bodyHTML,
		SenderName:  from.Name,
		SenderEmail: from.Email,
		To:          to,
		CC:          cc,
		Attachments: attachments,
		ReceivedAt:  receivedAt,
		IsRead:      hasIMAPFlag(item.Flags, "\\Seen"),
	}, nil
}

func parseMIMEBody(header textproto.MIMEHeader, body io.Reader) (string, []inboundAttachment, error) {
	contentType := header.Get("Content-Type")
	mediaType, params, _ := mime.ParseMediaType(contentType)
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		reader := multipart.NewReader(body, params["boundary"])
		var htmlPart string
		var textPart string
		var attachments []inboundAttachment
		totalAttachmentSize := 0
		for {
			part, err := reader.NextPart()
			if errorsIsEOF(err) {
				break
			}
			if err != nil {
				return "", nil, err
			}
			partHTML, partAttachments, err := parseMIMEPart(part)
			if err != nil {
				return "", nil, err
			}
			if htmlPart == "" && strings.Contains(strings.ToLower(part.Header.Get("Content-Type")), "text/html") {
				htmlPart = partHTML
			}
			if textPart == "" && strings.Contains(strings.ToLower(part.Header.Get("Content-Type")), "text/plain") {
				textPart = partHTML
			}
			for _, attachment := range partAttachments {
				totalAttachmentSize += len(attachment.Data)
				if totalAttachmentSize <= imapAttachmentTotalLimitB {
					attachments = append(attachments, attachment)
				}
			}
		}
		if htmlPart != "" {
			return htmlPart, attachments, nil
		}
		return textPart, attachments, nil
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return "", nil, err
	}
	if strings.Contains(strings.ToLower(mediaType), "text/html") {
		return cleanUTF8(string(data)), nil, nil
	}
	return plainTextToHTML(string(data)), nil, nil
}

func parseMIMEPart(part *multipart.Part) (string, []inboundAttachment, error) {
	mediaType, params, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
	disposition, dispositionParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	fileName := dispositionParams["filename"]
	if fileName == "" {
		fileName = params["name"]
	}
	if decoded, err := (&mime.WordDecoder{}).DecodeHeader(fileName); err == nil {
		fileName = decoded
	}
	if strings.EqualFold(disposition, "attachment") || fileName != "" {
		data, err := readDecodedPart(part)
		if err != nil {
			return "", nil, err
		}
		return "", []inboundAttachment{{
			FileName:    cleanUTF8(defaultString(fileName, "attachment")),
			ContentType: cleanUTF8(defaultString(mediaType, "application/octet-stream")),
			Data:        data,
		}}, nil
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		return parseMIMEBody(part.Header, part)
	}
	data, err := readDecodedPart(part)
	if err != nil {
		return "", nil, err
	}
	if strings.Contains(strings.ToLower(mediaType), "text/html") {
		return cleanUTF8(string(data)), nil, nil
	}
	if strings.Contains(strings.ToLower(mediaType), "text/plain") {
		return plainTextToHTML(string(data)), nil, nil
	}
	return "", nil, nil
}

func readDecodedPart(part *multipart.Part) ([]byte, error) {
	data, err := io.ReadAll(part)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(part.Header.Get("Content-Transfer-Encoding"))) {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(data)), ""))
		if err == nil {
			return decoded, nil
		}
	case "quoted-printable":
		reader := quotedprintable.NewReader(bytes.NewReader(data))
		decoded, err := io.ReadAll(reader)
		if err == nil {
			return decoded, nil
		}
	}
	return data, nil
}

func parseAddressHeader(value string) basedto.PersonRef {
	items := parseAddressListHeader(value)
	if len(items) == 0 {
		return basedto.PersonRef{}
	}
	return items[0]
}

func parseAddressListHeader(value string) []basedto.PersonRef {
	decoded, _ := (&mime.WordDecoder{}).DecodeHeader(value)
	addresses, err := mail.ParseAddressList(decoded)
	if err != nil {
		return []basedto.PersonRef{}
	}
	items := make([]basedto.PersonRef, 0, len(addresses))
	for _, address := range addresses {
		items = append(items, cleanPersonRef(basedto.PersonRef{Name: address.Name, Email: address.Address}))
	}
	return items
}

func imapQuote(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(value) + `"`
}

func parseIMAPUint(line string, pattern *regexp.Regexp) uint64 {
	match := pattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return 0
	}
	value, _ := strconv.ParseUint(match[1], 10, 64)
	return value
}

func parseIMAPFlags(line string) []string {
	upper := strings.ToUpper(line)
	index := strings.Index(upper, "FLAGS (")
	if index < 0 {
		return nil
	}
	start := index + len("FLAGS (")
	end := strings.Index(line[start:], ")")
	if end < 0 {
		return nil
	}
	return strings.Fields(line[start : start+end])
}

func parseIMAPInternalDate(line string) *time.Time {
	match := imapInternalDatePattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return nil
	}
	parsed, err := time.Parse("02-Jan-2006 15:04:05 -0700", match[1])
	if err != nil {
		return nil
	}
	return &parsed
}

func hasIMAPFlag(flags []string, flag string) bool {
	for _, item := range flags {
		if strings.EqualFold(item, flag) {
			return true
		}
	}
	return false
}

func plainTextToHTML(value string) string {
	value = cleanUTF8(value)
	blocks := strings.Split(strings.TrimSpace(value), "\n\n")
	var parts []string
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		for i, line := range lines {
			lines[i] = html.EscapeString(strings.TrimSpace(line))
		}
		parts = append(parts, "<p>"+strings.Join(lines, "<br/>")+"</p>")
	}
	return strings.Join(parts, "")
}

func errorsIsEOF(err error) bool {
	return err == io.EOF
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

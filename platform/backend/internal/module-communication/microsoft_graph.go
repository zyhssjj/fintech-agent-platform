package communication

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wfagent/internal/base/config"
	basedto "wfagent/internal/base/dto"
	"wfagent/internal/base/utils"
)

const microsoftGraphBaseURL = "https://graph.microsoft.com/v1.0"

type microsoftOAuthState struct {
	MailboxID uint64
	UserID    string
	ExpiresAt time.Time
}

var microsoftOAuthStates = struct {
	sync.Mutex
	items map[string]microsoftOAuthState
}{items: map[string]microsoftOAuthState{}}

type microsoftTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`
}

type graphRecipient struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

type graphMessage struct {
	ID                string `json:"id"`
	InternetMessageID string `json:"internetMessageId"`
	Subject           string `json:"subject"`
	Body              struct {
		ContentType string `json:"contentType"`
		Content     string `json:"content"`
	} `json:"body"`
	From             graphRecipient   `json:"from"`
	ToRecipients     []graphRecipient `json:"toRecipients"`
	CCRecipients     []graphRecipient `json:"ccRecipients"`
	IsRead           bool             `json:"isRead"`
	HasAttachments   bool             `json:"hasAttachments"`
	ReceivedDateTime time.Time        `json:"receivedDateTime"`
	Attachments      []inboundAttachment
}

type graphMessagePage struct {
	Value []graphMessage `json:"value"`
}

type graphAttachment struct {
	ID           string `json:"id"`
	ODataType    string `json:"@odata.type"`
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	ContentBytes string `json:"contentBytes"`
}

type graphAttachmentPage struct {
	Value []graphAttachment `json:"value"`
}

func (s *Service) BeginOutlookAuthorization(ctx context.Context, mailboxID, userID string) (basedto.JSONMap, error) {
	box, err := s.findMailbox(ctx, mailboxID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(box.OAuthProvider, "microsoft_graph") {
		return nil, fmt.Errorf("mailbox %s is not configured for Microsoft Graph", box.EmailAddress)
	}
	if err := validateMicrosoftConfig(); err != nil {
		return nil, err
	}
	state, err := newMicrosoftState(box.ID, userID)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", url.PathEscape(config.C.Microsoft.TenantID))
	query := url.Values{
		"client_id":     {config.C.Microsoft.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {config.C.Microsoft.RedirectURI},
		"response_mode": {"query"},
		"scope":         {strings.Join(config.C.Microsoft.GraphScopes, " ")},
		"state":         {state},
		"prompt":        {"select_account"},
	}
	return basedto.JSONMap{"authorization_url": endpoint + "?" + query.Encode()}, nil
}

func (s *Service) CompleteOutlookAuthorization(ctx context.Context, code, state, oauthErr, description string) (string, error) {
	frontendURL := strings.TrimRight(config.C.Microsoft.FrontendURL, "/") + "/agents/communication"
	if oauthErr != "" {
		return "", fmt.Errorf("Microsoft returned %s: %s", oauthErr, description)
	}
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("Microsoft did not return an authorization code")
	}
	oauthState, ok := consumeMicrosoftState(state)
	if !ok {
		return "", fmt.Errorf("authorization state is invalid or expired")
	}
	token, err := exchangeMicrosoftCode(ctx, code)
	if err != nil {
		return "", err
	}
	if err := s.saveMicrosoftToken(ctx, oauthState.MailboxID, token); err != nil {
		return "", err
	}
	return frontendURL + "?outlook=connected", nil
}

func (s *Service) sendOutbound(ctx context.Context, mailbox MailboxAccount, message outboundMessage) error {
	if strings.EqualFold(mailbox.OAuthProvider, "microsoft_graph") {
		return s.sendMicrosoftGraphMail(ctx, mailbox, message)
	}
	return s.mailSender.Send(ctx, mailbox, message)
}

func (s *Service) sendMicrosoftGraphMail(ctx context.Context, mailbox MailboxAccount, message outboundMessage) error {
	if len(collectRecipients(message)) == 0 {
		return fmt.Errorf("no outbound recipients")
	}
	accessToken, err := s.microsoftAccessToken(ctx, mailbox.ID)
	if err != nil {
		return err
	}
	payload := map[string]any{"message": map[string]any{
		"subject":       message.Subject,
		"body":          map[string]string{"contentType": "HTML", "content": message.HTMLBody},
		"toRecipients":  graphRecipients(message.To),
		"ccRecipients":  graphRecipients(message.CC),
		"bccRecipients": graphRecipients(message.BCC),
	}}
	if len(message.Attachments) > 0 {
		attachments := make([]map[string]any, 0, len(message.Attachments))
		for _, attachment := range message.Attachments {
			attachments = append(attachments, map[string]any{
				"@odata.type": "#microsoft.graph.fileAttachment",
				"name":        attachment.FileName, "contentType": attachment.ContentType,
				"contentBytes": base64.StdEncoding.EncodeToString(attachment.Data),
			})
		}
		payload["message"].(map[string]any)["attachments"] = attachments
	}
	return graphJSONRequest(ctx, http.MethodPost, microsoftGraphBaseURL+"/me/sendMail", accessToken, payload, nil)
}

func (s *Service) syncMailboxFromMicrosoftGraph(ctx context.Context, box MailboxAccount, actorID, actorName, trigger string) (basedto.JSONMap, error) {
	now := time.Now()
	logItem := ManualSyncLog{MailboxID: box.ID, TriggeredByID: actorID, TriggeredBy: truncateString(cleanUTF8(actorName), 128), Status: "success", Message: "Microsoft Graph sync completed", StartedAt: now}
	if trigger == "auto" {
		logItem.TriggeredByID, logItem.TriggeredBy = "system", "system"
	}
	accessToken, err := s.microsoftAccessToken(ctx, box.ID)
	if err == nil {
		var page graphMessagePage
		endpoint := microsoftGraphBaseURL + "/me/mailFolders/inbox/messages?$top=50&$orderby=receivedDateTime%20desc&$select=id,internetMessageId,subject,body,from,toRecipients,ccRecipients,isRead,hasAttachments,receivedDateTime"
		err = graphJSONRequest(ctx, http.MethodGet, endpoint, accessToken, nil, &page)
		if err == nil {
			for index := range page.Value {
				if !page.Value[index].HasAttachments {
					continue
				}
				page.Value[index].Attachments, err = fetchMicrosoftGraphAttachments(ctx, accessToken, page.Value[index].ID)
				if err != nil {
					break
				}
			}
		}
		if err == nil {
			err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				for _, item := range page.Value {
					created, createErr := s.createInboundEmailFromMicrosoftGraph(ctx, tx, box, item, actorID, actorName)
					if createErr != nil {
						return createErr
					}
					if created {
						logItem.SyncedCount++
					}
				}
				finishedAt := time.Now()
				logItem.FinishedAt = &finishedAt
				logItem.Message = fmt.Sprintf("Microsoft Graph sync completed, %d new email(s)", logItem.SyncedCount)
				if err := tx.Create(&logItem).Error; err != nil {
					return err
				}
				return tx.Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(map[string]any{"last_sync_at": finishedAt, "last_sync_status": "success", "last_sync_message": logItem.Message}).Error
			})
		}
	}
	if err != nil {
		finishedAt := time.Now()
		logItem.FinishedAt = &finishedAt
		logItem.Status = "failed"
		logItem.Message = truncateString(err.Error(), 255)
		_ = s.db.WithContext(ctx).Create(&logItem).Error
		_ = s.db.WithContext(ctx).Model(&MailboxAccount{}).Where("id = ?", box.ID).Updates(map[string]any{"last_sync_at": finishedAt, "last_sync_status": "failed", "last_sync_message": logItem.Message}).Error
		return nil, err
	}
	return basedto.JSONMap{"mailbox_id": box.ID, "synced_count": logItem.SyncedCount, "status": logItem.Status, "message": logItem.Message}, nil
}

func (s *Service) createInboundEmailFromMicrosoftGraph(ctx context.Context, tx *gorm.DB, box MailboxAccount, item graphMessage, actorID, actorName string) (bool, error) {
	messageID := strings.TrimSpace(item.InternetMessageID)
	if messageID == "" {
		messageID = "graph:" + item.ID
	}
	var count int64
	if err := tx.WithContext(ctx).Model(&Email{}).Where("mailbox_id = ? AND message_id = ? AND direction = ? AND is_deleted = false", box.ID, messageID, "inbound").Count(&count).Error; err != nil || count > 0 {
		return false, err
	}
	body := item.Body.Content
	if !strings.EqualFold(item.Body.ContentType, "html") {
		body = "<p>" + escapeHTML(body) + "</p>"
	}
	from := personRefFromGraph(item.From)
	to, cc := personRefsFromGraph(item.ToRecipients), personRefsFromGraph(item.CCRecipients)
	score, label, needsReply, summary, draft := classifyEmailPriority(item.Subject, body, []string{"microsoft_graph"})
	email := Email{EmailCode: "MAIL-" + uuid.NewString()[:8], MailboxID: box.ID, Direction: "inbound", Folder: "inbox", MessageID: truncateString(cleanUTF8(messageID), 255), ThreadID: truncateString("graph:"+item.ID, 128), Subject: truncateString(cleanUTF8(item.Subject), 255), Preview: summarizeHTML(body), BodyHTML: cleanUTF8(body), SenderName: truncateString(cleanUTF8(from.Name), 128), SenderEmail: truncateString(cleanUTF8(from.Email), 128), ToRecipients: toRecipients(cleanPersonRefs(to)), CcRecipients: toRecipients(cleanPersonRefs(cc)), Summary: summary, ReplyDraft: draft, PriorityScore: score, PriorityLabel: label, NeedsReply: needsReply, IsRead: item.IsRead, HasAttachments: len(item.Attachments) > 0, Status: "ready", OperatorUserID: truncateString(cleanUTF8(actorID), 64), OperatorUsername: truncateString(cleanUTF8(actorName), 128), ReceivedAt: &item.ReceivedDateTime}
	if err := tx.Create(&email).Error; err != nil {
		return false, err
	}
	for _, attachment := range item.Attachments {
		if err := tx.Create(&EmailAttachment{EmailID: email.ID, FileName: truncateString(cleanUTF8(attachment.FileName), 255), ContentType: truncateString(cleanUTF8(attachment.ContentType), 128), FileSize: int64(len(attachment.Data)), FileData: attachment.Data}).Error; err != nil {
			return false, err
		}
	}
	return true, nil
}

func fetchMicrosoftGraphAttachments(ctx context.Context, accessToken, messageID string) ([]inboundAttachment, error) {
	var page graphAttachmentPage
	endpoint := microsoftGraphBaseURL + "/me/messages/" + url.PathEscape(messageID) + "/attachments"
	if err := graphJSONRequest(ctx, http.MethodGet, endpoint, accessToken, nil, &page); err != nil {
		return nil, err
	}
	attachments := make([]inboundAttachment, 0, len(page.Value))
	var total int64
	for _, item := range page.Value {
		if !strings.EqualFold(item.ODataType, "#microsoft.graph.fileAttachment") {
			continue
		}
		if item.ContentBytes == "" && item.ID != "" {
			detailEndpoint := microsoftGraphBaseURL + "/me/messages/" + url.PathEscape(messageID) + "/attachments/" + url.PathEscape(item.ID)
			if err := graphJSONRequest(ctx, http.MethodGet, detailEndpoint, accessToken, nil, &item); err != nil {
				return nil, err
			}
		}
		if item.ContentBytes == "" {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(item.ContentBytes)
		if err != nil {
			return nil, fmt.Errorf("decode Outlook attachment %s: %w", item.Name, err)
		}
		if total+int64(len(data)) > imapAttachmentTotalLimitB {
			continue
		}
		total += int64(len(data))
		attachments = append(attachments, inboundAttachment{FileName: item.Name, ContentType: item.ContentType, Data: data})
	}
	return attachments, nil
}

func (s *Service) microsoftAccessToken(ctx context.Context, mailboxID uint64) (string, error) {
	if err := validateMicrosoftConfig(); err != nil {
		return "", err
	}
	var token MailboxOAuthToken
	if err := s.db.WithContext(ctx).Where("mailbox_id = ? AND is_deleted = false", mailboxID).First(&token).Error; err != nil {
		return "", fmt.Errorf("Outlook mailbox is not authorized: %w", err)
	}
	accessToken, err := decryptMicrosoftToken(token.AccessToken)
	if err != nil {
		return "", err
	}
	if token.ExpiryAt == nil || time.Now().Before(token.ExpiryAt.Add(-time.Minute)) {
		return accessToken, nil
	}
	refreshToken, err := decryptMicrosoftToken(token.RefreshToken)
	if err != nil {
		return "", err
	}
	fresh, err := refreshMicrosoftToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}
	if fresh.RefreshToken == "" {
		fresh.RefreshToken = refreshToken
	}
	if err := s.saveMicrosoftToken(ctx, mailboxID, fresh); err != nil {
		return "", err
	}
	return fresh.AccessToken, nil
}

func (s *Service) saveMicrosoftToken(ctx context.Context, mailboxID uint64, token microsoftTokenResponse) error {
	access, err := encryptMicrosoftToken(token.AccessToken)
	if err != nil {
		return err
	}
	refresh, err := encryptMicrosoftToken(token.RefreshToken)
	if err != nil {
		return err
	}
	expiry := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := MailboxOAuthToken{}
		err := tx.Where("mailbox_id = ?", mailboxID).First(&row).Error
		values := map[string]any{"access_token": access, "refresh_token": refresh, "token_type": token.TokenType, "scope": token.Scope, "expiry_at": &expiry}
		if err == nil {
			return tx.Model(&MailboxOAuthToken{}).Where("id = ?", row.ID).Updates(values).Error
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		return tx.Create(&MailboxOAuthToken{MailboxID: mailboxID, AccessToken: access, RefreshToken: refresh, TokenType: token.TokenType, Scope: token.Scope, ExpiryAt: &expiry}).Error
	})
}

func exchangeMicrosoftCode(ctx context.Context, code string) (microsoftTokenResponse, error) {
	return microsoftTokenRequest(ctx, url.Values{"client_id": {config.C.Microsoft.ClientID}, "client_secret": {config.C.Microsoft.ClientSecret}, "grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {config.C.Microsoft.RedirectURI}})
}
func refreshMicrosoftToken(ctx context.Context, refreshToken string) (microsoftTokenResponse, error) {
	return microsoftTokenRequest(ctx, url.Values{"client_id": {config.C.Microsoft.ClientID}, "client_secret": {config.C.Microsoft.ClientSecret}, "grant_type": {"refresh_token"}, "refresh_token": {refreshToken}, "scope": {strings.Join(config.C.Microsoft.GraphScopes, " ")}})
}
func microsoftTokenRequest(ctx context.Context, form url.Values) (microsoftTokenResponse, error) {
	endpoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", url.PathEscape(config.C.Microsoft.TenantID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return microsoftTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return microsoftTokenResponse{}, fmt.Errorf("request Microsoft token: %w", err)
	}
	defer resp.Body.Close()
	var token microsoftTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return token, err
	}
	if resp.StatusCode >= 300 {
		return token, fmt.Errorf("Microsoft token request failed: %s %s", token.Error, token.Description)
	}
	return token, nil
}
func graphJSONRequest(ctx context.Context, method, endpoint, token string, input any, output any) error {
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = strings.NewReader(string(raw))
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("request Microsoft Graph: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("Microsoft Graph request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if output != nil {
		return json.NewDecoder(resp.Body).Decode(output)
	}
	return nil
}
func graphRecipients(items []basedto.PersonRef) []map[string]map[string]string {
	out := make([]map[string]map[string]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Email) != "" {
			out = append(out, map[string]map[string]string{"emailAddress": {"name": item.Name, "address": item.Email}})
		}
	}
	return out
}
func personRefFromGraph(item graphRecipient) basedto.PersonRef {
	return basedto.PersonRef{Name: item.EmailAddress.Name, Email: item.EmailAddress.Address}
}
func personRefsFromGraph(items []graphRecipient) []basedto.PersonRef {
	out := make([]basedto.PersonRef, 0, len(items))
	for _, item := range items {
		out = append(out, personRefFromGraph(item))
	}
	return out
}
func validateMicrosoftConfig() error {
	if config.C == nil || strings.TrimSpace(config.C.Microsoft.ClientID) == "" || strings.TrimSpace(config.C.Microsoft.ClientSecret) == "" || strings.TrimSpace(config.C.Microsoft.TokenEncryptionKey) == "" {
		return fmt.Errorf("Microsoft Graph is not configured; set MICROSOFT_CLIENT_ID, MICROSOFT_CLIENT_SECRET and MICROSOFT_TOKEN_ENCRYPTION_KEY")
	}
	return nil
}
func encryptionKey() []byte {
	decoded, err := base64.StdEncoding.DecodeString(config.C.Microsoft.TokenEncryptionKey)
	if err == nil && len(decoded) > 0 {
		return decoded
	}
	return []byte(config.C.Microsoft.TokenEncryptionKey)
}
func encryptMicrosoftToken(value string) (string, error) {
	return utils.AESEncrypt([]byte(value), encryptionKey())
}
func decryptMicrosoftToken(value string) (string, error) {
	plain, err := utils.AESDecrypt(value, encryptionKey())
	if err != nil {
		return "", fmt.Errorf("decrypt Microsoft token: %w", err)
	}
	return string(plain), nil
}
func newMicrosoftState(mailboxID uint64, userID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(raw)
	microsoftOAuthStates.Lock()
	defer microsoftOAuthStates.Unlock()
	now := time.Now()
	for key, value := range microsoftOAuthStates.items {
		if now.After(value.ExpiresAt) {
			delete(microsoftOAuthStates.items, key)
		}
	}
	microsoftOAuthStates.items[state] = microsoftOAuthState{MailboxID: mailboxID, UserID: userID, ExpiresAt: now.Add(10 * time.Minute)}
	return state, nil
}
func consumeMicrosoftState(state string) (microsoftOAuthState, bool) {
	microsoftOAuthStates.Lock()
	defer microsoftOAuthStates.Unlock()
	value, ok := microsoftOAuthStates.items[state]
	delete(microsoftOAuthStates.items, state)
	return value, ok && time.Now().Before(value.ExpiresAt)
}

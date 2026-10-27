package communication

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

const mailboxAutoSyncInterval = 5 * time.Minute

func StartMailboxAutoSync(ctx context.Context, db *gorm.DB) {
	if db == nil {
		return
	}
	service := NewService(db)
	go func() {
		ticker := time.NewTicker(mailboxAutoSyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				service.syncActiveMailboxes(ctx)
			}
		}
	}()
}

func (s *Service) syncActiveMailboxes(ctx context.Context) {
	var boxes []MailboxAccount
	if err := s.db.WithContext(ctx).
		Where("is_deleted = false AND status = ? AND ((imap_host <> '' AND imap_host NOT ILIKE ?) OR oauth_provider = ?)", "active", "%example.com%", "microsoft_graph").
		Find(&boxes).Error; err != nil {
		slog.Warn("load communication mailboxes for auto sync failed", "err", err)
		return
	}
	for _, box := range boxes {
		var err error
		if box.OAuthProvider == "microsoft_graph" {
			_, err = s.syncMailboxFromMicrosoftGraph(ctx, box, "system", "system", "auto")
		} else {
			_, err = s.syncMailboxFromIMAPClean(ctx, box, "system", "system", "auto")
		}
		if err != nil {
			slog.Warn("communication mailbox auto sync failed", "mailbox_id", box.ID, "email", box.EmailAddress, "err", err)
		}
	}
}

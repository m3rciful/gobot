package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/m3rciful/gobot/core/logger"
	tghelpers "github.com/m3rciful/gobot/core/telegram/helpers"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// recentUpdates keeps a short-lived set of processed update IDs to avoid double logging.
var (
	recentMu     sync.Mutex
	recentUpdate = make(map[int]time.Time)
	keepFor      = 10 * time.Second
)

func alreadyLogged(updateID int) bool {
	now := time.Now()
	recentMu.Lock()
	defer recentMu.Unlock()
	// GC old entries
	for id, ts := range recentUpdate {
		if now.Sub(ts) > keepFor {
			delete(recentUpdate, id)
		}
	}
	if _, ok := recentUpdate[updateID]; ok {
		return true
	}
	recentUpdate[updateID] = now
	return false
}

// LoggerMiddleware logs a single receipt line per update and sets rid.
// It deduplicates by update_id to prevent double logging when middleware is applied on multiple branches.
func LoggerMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		upd := c.Update()
		user := c.Sender()
		chat := c.Chat()

		// Build rid and expose to downstream handlers
		chatID, userID := int64(0), int64(0)
		if chat != nil {
			chatID = chat.ID
		}
		if user != nil {
			userID = user.ID
		}
		rid := logger.BuildRID(upd.ID, chatID, userID)
		c.Set("rid", rid)
		c.Set("update_start", time.Now())

		ctx := logger.WithRID(logger.Background(), rid)
		ctx = logger.WithUpdateMeta(ctx, upd.ID, userID, chatID)
		ctx = logger.WithLogger(ctx, logger.Component("tg"))
		tghelpers.StoreContext(c, ctx)

		// Deduplicate update receipt logs
		if logger.ShouldSampleDebug() && !alreadyLogged(upd.ID) {
			attrs := []slog.Attr{
				slog.String("status", "ok"),
				slog.String("rid", rid),
				slog.Int("update_id", upd.ID),
			}
			if chatID != 0 {
				attrs = append(attrs, slog.Int64("chat_id", chatID))
				attrs = append(attrs, slog.String("chat_type", string(chat.Type)))
			}
			if userID != 0 {
				attrs = append(attrs, slog.Int64("user_id", userID))
				if user != nil && user.Username != "" {
					attrs = append(attrs, slog.String("username", logger.SanitizeLimit(user.Username, 64)))
				}
				if user != nil && user.LanguageCode != "" {
					attrs = append(attrs, slog.String("lang", user.LanguageCode))
				}
			}

			// Enrich by kind
			switch {
			case upd.Callback != nil:
				key, payload := parseCallback(upd.Callback)
				if key != "" {
					attrs = append(attrs, slog.String("cb_key", logger.SanitizeLimit(key, 128)))
				}
				if payload != "" {
					attrs = append(attrs, slog.String("payload", logger.SanitizeLimit(payload, 256)))
				}
			case upd.Message != nil:
				if t := c.Text(); t != "" {
					attrs = append(attrs, slog.String("payload", logger.SanitizeLimit(t, 256)))
				}
			}
			logger.LogEvent(ctx, logger.Component("tg"), slog.LevelDebug, "update.received", attrs...)
		}

		return next(c)
	}
}

func parseCallback(cb *tele.Callback) (string, string) {
	if cb == nil {
		return "", ""
	}
	if cb.Unique != "" {
		return cb.Unique, cb.Data
	}
	raw := strings.TrimPrefix(cb.Data, "\\f")
	parts := strings.SplitN(raw, "|", 2)
	key := strings.TrimSpace(parts[0])
	payload := ""
	if len(parts) == 2 {
		payload = parts[1]
	}
	return key, payload
}

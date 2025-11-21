package helpers

import (
	"errors"
	"log/slog"
	"sync/atomic"

	"github.com/m3rciful/gobot/core/logger"
	"github.com/m3rciful/gobot/core/telegram/sender"

	tele "gopkg.in/telebot.v4"
)

var globalDispatcher atomic.Pointer[sender.Dispatcher]

// SetDispatcher wires the asynchronous sender used by helper functions.
func SetDispatcher(d *sender.Dispatcher) {
	globalDispatcher.Store(d)
}

func currentDispatcher() *sender.Dispatcher {
	return globalDispatcher.Load()
}

func sendAsync(c tele.Context, action, endpoint string, run func() error) error {
	disp := currentDispatcher()
	if disp == nil {
		return run()
	}

	ctx := BuildContext(c)
	if err := disp.Enqueue(ctx, action, endpoint, run); err != nil {
		if errors.Is(err, sender.ErrQueueFull) || errors.Is(err, sender.ErrQueueClosed) {
			logger.Warn(ctx, "tg.sender", "queue.fallback",
				slog.String("action", action),
				slog.String("endpoint", endpoint),
				slog.String("err", err.Error()),
			)
			return run()
		}
		return err
	}
	return nil
}

// SendText sends raw text (no parse mode) to the current recipient.
func SendText(c tele.Context, text string, opts ...*tele.SendOptions) error {
	var sendOpts *tele.SendOptions
	if len(opts) > 0 {
		sendOpts = opts[0]
	}
	return sendAsync(c, "send.text", "sendMessage", func() error {
		if sendOpts != nil {
			return c.Send(text, sendOpts)
		}
		return c.Send(text)
	})
}

// SendMD sends a message with Markdown parse mode and optional reply markup.
func SendMD(c tele.Context, text string, markup ...*tele.ReplyMarkup) error {
	var rm *tele.ReplyMarkup
	if len(markup) > 0 {
		rm = markup[0]
	}
	opts := &tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: rm}
	return SendText(c, text, opts)
}

// SendMDV2 sends a message with MarkdownV2 parse mode and optional reply markup.
func SendMDV2(c tele.Context, text string, markup ...*tele.ReplyMarkup) error {
	var rm *tele.ReplyMarkup
	if len(markup) > 0 {
		rm = markup[0]
	}
	opts := &tele.SendOptions{ParseMode: tele.ModeMarkdownV2, ReplyMarkup: rm}
	return SendText(c, text, opts)
}

// EditMD edits a message with Markdown parse mode and optional reply markup.
func EditMD(c tele.Context, text string, markup ...*tele.ReplyMarkup) error {
	var rm *tele.ReplyMarkup
	if len(markup) > 0 {
		rm = markup[0]
	}
	return c.Edit(text, &tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: rm})
}

// EditOrSendMD tries to edit the message (Markdown) or sends a new one if edit fails.
func EditOrSendMD(c tele.Context, text string, markup ...*tele.ReplyMarkup) error {
	var rm *tele.ReplyMarkup
	if len(markup) > 0 {
		rm = markup[0]
	}
	return c.EditOrSend(text, &tele.SendOptions{ParseMode: tele.ModeMarkdown, ReplyMarkup: rm})
}

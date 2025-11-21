package keyboard

import tele "gopkg.in/telebot.v4"

// InlineBtn describes a convenience wrapper for inline button properties.
type InlineBtn struct {
	Text   string
	Unique string
	Data   string
}

const defaultCancelButtonText = "❌ Cancel"

// ForceReply returns a markup that forces the user to reply.
func ForceReply() *tele.ReplyMarkup {
	return &tele.ReplyMarkup{ForceReply: true}
}

// RemoveKeyboard returns a markup that hides the keyboard.
func RemoveKeyboard() *tele.ReplyMarkup {
	return &tele.ReplyMarkup{RemoveKeyboard: true}
}

// ReplyButtons builds a reply keyboard from rows of text.
func ReplyButtons(rows ...[]string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}
	var keyboard []tele.Row
	for _, row := range rows {
		var buttons []tele.Btn
		for _, label := range row {
			buttons = append(buttons, markup.Text(label))
		}
		keyboard = append(keyboard, markup.Row(buttons...))
	}
	markup.Reply(keyboard...)
	return markup
}

// InlineButtons builds an inline keyboard where each provided button is placed on its own row.
func InlineButtons(buttons []InlineBtn) *tele.ReplyMarkup {
	rows := make([][]InlineBtn, 0, len(buttons))
	for _, b := range buttons {
		rows = append(rows, []InlineBtn{b})
	}
	return InlineButtonsRows(rows...)
}

// InlineButtonsRows builds an inline keyboard from rows of InlineBtn.
func InlineButtonsRows(rows ...[]InlineBtn) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	inline := make([][]tele.InlineButton, len(rows))
	for i, row := range rows {
		r := make([]tele.InlineButton, len(row))
		for j, btn := range row {
			r[j] = *markup.Data(btn.Text, btn.Unique, btn.Data).Inline()
		}
		inline[i] = r
	}
	markup.InlineKeyboard = inline
	return markup
}

// InlineButtonsNPerRow splits a flat list of buttons into rows with up to n buttons per row.
// If n <= 1, it behaves like InlineButtons (one per row).
func InlineButtonsNPerRow(buttons []InlineBtn, n int) *tele.ReplyMarkup {
	if n <= 1 {
		return InlineButtons(buttons)
	}
	var rows [][]InlineBtn
	for i := 0; i < len(buttons); i += n {
		end := i + n
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, buttons[i:end])
	}
	return InlineButtonsRows(rows...)
}

// ToInlineKeyboard converts [][]tele.Btn to [][]tele.InlineButton.
func ToInlineKeyboard(buttons [][]tele.Btn) [][]tele.InlineButton {
	inline := make([][]tele.InlineButton, 0, len(buttons))
	for _, row := range buttons {
		r := make([]tele.InlineButton, 0, len(row))
		for _, b := range row {
			r = append(r, *b.Inline())
		}
		inline = append(inline, r)
	}
	return inline
}

// ChunkButtons splits a flat list of tele.Btn into rows with up to n buttons per row.
func ChunkButtons(buttons []tele.Btn, n int) [][]tele.Btn {
	if n <= 1 {
		out := make([][]tele.Btn, 0, len(buttons))
		for _, b := range buttons {
			out = append(out, []tele.Btn{b})
		}
		return out
	}
	var rows [][]tele.Btn
	for i := 0; i < len(buttons); i += n {
		end := i + n
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, buttons[i:end])
	}
	return rows
}

// CancelButton returns a reusable cancel inline button for the provided markup and action.
// Optional arguments allow overriding payload (first value) and button label (second value).
func CancelButton(markup *tele.ReplyMarkup, action string, options ...string) tele.Btn {
	payload := "cancel"
	if len(options) > 0 && options[0] != "" {
		payload = options[0]
	}
	text := defaultCancelButtonText
	if len(options) > 1 && options[1] != "" {
		text = options[1]
	}
	return markup.Data(text, action, payload)
}

// SingleCancelMarkup creates an inline keyboard with a single cancel button.
func SingleCancelMarkup(action string, options ...string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	btn := CancelButton(markup, action, options...)
	markup.InlineKeyboard = [][]tele.InlineButton{{*btn.Inline()}}
	return markup
}

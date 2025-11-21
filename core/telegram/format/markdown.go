package format

import (
	"fmt"
	"regexp"
)

const (
	// MarkdownV1 denotes Telegram markdown version 1.
	MarkdownV1 = 1
	// MarkdownV2 denotes Telegram markdown version 2.
	MarkdownV2 = 2
)

const mdV2Specials = "_*[]()~`>#+-=|{}.!"

// EscapeMarkdown escapes special characters for MarkdownV1 or V2.
func EscapeMarkdown(text string, version int, entityType string) (string, error) {
	switch version {
	case MarkdownV1:
		re := regexp.MustCompile(`([_*\\\[` + "`" + `])`)
		return re.ReplaceAllString(text, `\\$1`), nil
	case MarkdownV2:
		re := regexp.MustCompile("[" + regexp.QuoteMeta(mdV2Specials) + "]")
		return re.ReplaceAllString(text, `\\$1`), nil
	}
	return "", fmt.Errorf("unsupported markdown version: %d", version)
}

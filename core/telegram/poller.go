package telegram

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
)

const (
	RunModeWebhook  = "webhook"
	RunModeLongpoll = "longpoll"
)

// WebhookOptions declares webhook listener settings.
type WebhookOptions struct {
	Listen string
	Port   int
	URL    string
}

// PollerOptions configures BuildPoller.
type PollerOptions struct {
	RunMode                string
	LongPollTimeoutSeconds int
	Webhook                WebhookOptions
}

// BuildPoller returns a Telebot poller based on provided options.
func BuildPoller(opts PollerOptions) tele.Poller {
	runMode := strings.ToLower(strings.TrimSpace(opts.RunMode))
	if runMode == RunModeWebhook {
		return &tele.Webhook{
			Listen:   fmt.Sprintf("%s:%d", opts.Webhook.Listen, opts.Webhook.Port),
			Endpoint: &tele.WebhookEndpoint{PublicURL: opts.Webhook.URL},
		}
	}

	timeoutSec := opts.LongPollTimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 10
	}
	return &tele.LongPoller{Timeout: time.Duration(timeoutSec) * time.Second}
}

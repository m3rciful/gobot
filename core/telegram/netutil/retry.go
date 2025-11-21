package netutil

import (
	"errors"
	"net"
	"net/url"
)

// ShouldRetry reports whether a network error is worth retrying.
// It focuses on transient dial/timeout failures produced by net/http
// while contacting the Telegram API.
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() || netErr.Temporary() {
			return true
		}
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Timeout() || opErr.Op == "dial" {
			return true
		}
		if nested, ok := opErr.Err.(net.Error); ok {
			if nested.Timeout() || nested.Temporary() {
				return true
			}
		}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		if urlErr.Err != nil && !errors.Is(urlErr.Err, err) {
			return ShouldRetry(urlErr.Err)
		}
	}

	return false
}

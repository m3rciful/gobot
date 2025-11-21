package telegram

import (
	"net"
	"net/http"
	"time"

	"gobot/core/telegram/netutil"
)

const (
	defaultDialTimeout       = 5 * time.Second
	defaultTLSHandshake      = 5 * time.Second
	defaultIdleConnTimeout   = 30 * time.Second
	defaultResponseTimeout   = 5 * time.Second
	defaultClientTimeout     = 30 * time.Second
	defaultKeepAliveInterval = 30 * time.Second
	defaultRetryAttempts     = 3
	defaultRetryBackoff      = 2 * time.Second
)

// BuildHTTPClient returns an HTTP client tuned for Telegram API calls.
func BuildHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: defaultDialTimeout, KeepAlive: defaultKeepAliveInterval}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       defaultIdleConnTimeout,
		TLSHandshakeTimeout:   defaultTLSHandshake,
		ResponseHeaderTimeout: defaultResponseTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	retry := &retryTransport{
		base:       transport,
		maxRetries: defaultRetryAttempts,
		backoff:    defaultRetryBackoff,
	}

	return &http.Client{
		Timeout:   defaultClientTimeout,
		Transport: retry,
	}
}

type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
	backoff    time.Duration
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	attempts := t.maxRetries + 1
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		currReq := req
		if attempt > 1 {
			currReq = req.Clone(req.Context())
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				currReq.Body = body
			} else if req.Body != nil {
				return nil, lastErr
			}
		}

		resp, err := base.RoundTrip(currReq)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !netutil.ShouldRetry(err) || attempt == attempts {
			break
		}

		delay := t.backoff * time.Duration(attempt)
		if delay <= 0 {
			continue
		}
		timer := time.NewTimer(delay)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		case <-timer.C:
		}
	}

	return nil, lastErr
}

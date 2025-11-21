package logger

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"log/slog"
)

func TestStructuredHandlerKVOrder(t *testing.T) {
	buf := &bytes.Buffer{}
	aw := newAsyncWriter([]io.Writer{buf}, 1024)
	handler := newStructuredHandler(handlerConfig{
		level:    slog.LevelInfo,
		writer:   aw,
		format:   formatKV,
		keyOrder: append([]string(nil), defaultKeyOrder...),
	})
	ctx := WithRID(Background(), "rid-123")
	ctx = WithUpdateMeta(ctx, 42, 7, 9)

	log := slog.New(handler).With("component", "app")
	LogEvent(ctx, log, slog.LevelInfo, "test.event",
		slog.String("status", "ok"),
		slog.String("cause", "unit"),
	)
	if err := aw.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if err := aw.Close(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected log line")
	}
	tokens := strings.Split(line, " ")
	if len(tokens) < 6 {
		t.Fatalf("unexpected token count: %d (%s)", len(tokens), line)
	}
	expected := []string{"ts=", "level=INFO", "component=app", "event=test.event", "status=ok", "rid=rid-123"}
	for i, prefix := range expected {
		if !strings.HasPrefix(tokens[i], prefix) {
			t.Fatalf("token %d = %s, expected prefix %s", i, tokens[i], prefix)
		}
	}
}

func TestStructuredHandlerJSONOrder(t *testing.T) {
	buf := &bytes.Buffer{}
	aw := newAsyncWriter([]io.Writer{buf}, 1024)
	handler := newStructuredHandler(handlerConfig{
		level:    slog.LevelInfo,
		writer:   aw,
		format:   formatJSON,
		keyOrder: append([]string(nil), defaultKeyOrder...),
	})
	ctx := WithRID(Background(), "rid-json")
	ctx = WithUpdateMeta(ctx, 11, 22, 33)

	log := slog.New(handler).With("component", "service.test")
	LogEvent(ctx, log, slog.LevelError, "service.failed",
		slog.String("status", "fail"),
		slog.String("err", "boom"),
		slog.String("err_code", "TEST_FAIL"),
	)
	if err := aw.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if err := aw.Close(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(line, "{") {
		t.Fatalf("expected JSON, got %s", line)
	}
	prefixes := []string{`{"ts":`, `"level":"ERROR"`, `"component":"service.test"`, `"event":"service.failed"`, `"status":"fail"`, `"rid":"rid-json"`}
	pos := -1
	for _, pref := range prefixes {
		idx := strings.Index(line, pref)
		if idx == -1 || idx < pos {
			t.Fatalf("prefix %s not found in order within %s", pref, line)
		}
		pos = idx
	}
}

func TestStructuredHandlerCompactRID(t *testing.T) {
	buf := &bytes.Buffer{}
	aw := newAsyncWriter([]io.Writer{buf}, 1024)
	handler := newStructuredHandler(handlerConfig{
		level:    slog.LevelInfo,
		writer:   aw,
		format:   formatKV,
		keyOrder: append([]string(nil), defaultKeyOrder...),
	})
	rawRID := "123:456:789"
	ctx := WithRID(Background(), rawRID)
	log := slog.New(handler).With("component", "app")
	LogEvent(ctx, log, slog.LevelInfo, "rid.test",
		slog.String("status", "ok"),
	)
	if err := aw.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if err := aw.Close(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	line := strings.TrimSpace(buf.String())
	if !strings.Contains(line, "rid="+CompactRID(rawRID)) {
		t.Fatalf("expected compact rid, got %s", line)
	}
	if strings.Contains(line, "rid_full=") {
		t.Fatalf("rid_full should be omitted in KV output, got %s", line)
	}
}

func TestStructuredHandlerCompactRIDJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	aw := newAsyncWriter([]io.Writer{buf}, 1024)
	handler := newStructuredHandler(handlerConfig{
		level:    slog.LevelInfo,
		writer:   aw,
		format:   formatJSON,
		keyOrder: append([]string(nil), defaultKeyOrder...),
	})
	rawRID := "12:34:56"
	ctx := WithRID(Background(), rawRID)
	log := slog.New(handler).With("component", "app")
	LogEvent(ctx, log, slog.LevelInfo, "rid.test",
		slog.String("status", "ok"),
	)
	if err := aw.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if err := aw.Close(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	line := strings.TrimSpace(buf.String())
	if !strings.Contains(line, `"rid":"`+CompactRID(rawRID)+`"`) {
		t.Fatalf("expected compact rid in JSON, got %s", line)
	}
	if !strings.Contains(line, `"rid_full":"`+rawRID+`"`) {
		t.Fatalf("expected rid_full in JSON output, got %s", line)
	}
	if !strings.Contains(line, `"ts_unix_nano"`) {
		t.Fatalf("expected ts_unix_nano to be present in JSON output, got %s", line)
	}
}

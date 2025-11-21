package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type logFormat string

const (
	formatJSON logFormat = "json"
	formatKV   logFormat = "kv"

	timeFormatMillis = "2006-01-02T15:04:05.000Z07:00"
)

type handlerConfig struct {
	level    slog.Leveler
	writer   *asyncWriter
	format   logFormat
	keyOrder []string
	stacks   bool
}

type structuredHandler struct {
	cfg    handlerConfig
	attrs  []slog.Attr
	groups []string
}

func newStructuredHandler(cfg handlerConfig) *structuredHandler {
	if cfg.level == nil {
		cfg.level = slog.LevelInfo
	}
	if cfg.keyOrder == nil {
		cfg.keyOrder = append([]string(nil), defaultKeyOrder...)
	}
	return &structuredHandler{cfg: cfg}
}

// Enabled reports whether the handler allows processing the provided level.
func (h *structuredHandler) Enabled(_ context.Context, level slog.Level) bool {
	min := slog.LevelInfo
	if h.cfg.level != nil {
		min = h.cfg.level.Level()
	}
	return level >= min
}

// Handle formats the slog.Record and writes it using the configured writer.
func (h *structuredHandler) Handle(ctx context.Context, r slog.Record) error {
	if h.cfg.writer == nil {
		return fmt.Errorf("logger: writer not initialized")
	}

	fields := make(map[string]any, 16)
	isJSON := h.cfg.format == formatJSON
	ts := r.Time.UTC()
	fields["ts"] = ts.Truncate(time.Millisecond).Format(timeFormatMillis)
	fields["level"] = normalizeLevel(r.Level.String())
	if isJSON {
		fields["ts_unix_nano"] = ts.UnixNano()
	}

	if len(h.attrs) > 0 {
		h.collectAttrs(fields, h.attrs)
	}

	r.Attrs(func(a slog.Attr) bool {
		h.collectAttr(fields, a)
		return true
	})

	addContextFields(ctx, fields)

	if rid, ok := stringField(fields, "rid"); ok && rid != "" {
		if compact := CompactRID(rid); compact != "" && compact != rid {
			if isJSON {
				if _, seen := fields["rid_full"]; !seen {
					fields["rid_full"] = rid
				}
			}
			fields["rid"] = compact
		}
	}

	if event, ok := stringField(fields, "event"); !ok || event == "" {
		if r.Message != "" {
			fields["event"] = r.Message
		} else {
			fields["event"] = "unknown"
		}
	}

	if component, ok := stringField(fields, "component"); !ok || component == "" {
		fields["component"] = "app"
	}

	sanitizeEnumerations(fields)
	pruneEmpty(fields)

	line, err := h.format(fields)
	if err != nil {
		return err
	}
	if len(line) == 0 || line[len(line)-1] != '\n' {
		line = append(line, '\n')
	}
	return h.cfg.writer.Write(line)
}

// WithAttrs returns a shallow copy of the handler enriched with attrs.
func (h *structuredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &clone
}

// WithGroup returns a shallow copy of the handler with an additional group prefix.
func (h *structuredHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groups = append(append([]string(nil), h.groups...), name)
	return &clone
}

func (h *structuredHandler) collectAttrs(fields map[string]any, attrs []slog.Attr) {
	for _, a := range attrs {
		h.collectAttr(fields, a)
	}
}

func (h *structuredHandler) collectAttr(fields map[string]any, attr slog.Attr) {
	flattenAttr(joinGroups(h.groups, ""), attr, func(k string, v slog.Value) {
		if k == "" {
			return
		}
		key, val, ok := normalizeAttr(k, v)
		if !ok {
			return
		}
		fields[key] = val
	})
}

func (h *structuredHandler) format(fields map[string]any) ([]byte, error) {
	switch h.cfg.format {
	case formatJSON:
		return formatJSONLine(fields, h.cfg.keyOrder)
	default:
		return formatKVLine(fields, h.cfg.keyOrder), nil
	}
}

func flattenAttr(prefix string, attr slog.Attr, fn func(string, slog.Value)) {
	key := attr.Key
	if key == "" {
		key = prefix
	} else if prefix != "" {
		key = prefix + "." + key
	}
	val := attr.Value
	switch val.Kind() {
	case slog.KindGroup:
		sub := val.Group()
		for _, child := range sub {
			flattenAttr(key, child, fn)
		}
	default:
		fn(key, val)
	}
}

func joinGroups(groups []string, leaf string) string {
	if len(groups) == 0 {
		return leaf
	}
	if leaf == "" {
		return strings.Join(groups, ".")
	}
	return strings.Join(groups, ".") + "." + leaf
}

func normalizeAttr(key string, val slog.Value) (string, any, bool) {
	if key == "" {
		return "", nil, false
	}
	switch val.Kind() {
	case slog.KindString:
		return key, strings.TrimSpace(val.String()), true
	case slog.KindBool:
		return key, val.Bool(), true
	case slog.KindInt64:
		return key, val.Int64(), true
	case slog.KindUint64:
		u := val.Uint64()
		if u <= math.MaxInt64 {
			return key, int64(u), true
		}
		return key, u, true
	case slog.KindFloat64:
		return key, val.Float64(), true
	case slog.KindDuration:
		ms := RoundMS(val.Duration()).Milliseconds()
		normalized := key
		switch {
		case key == "duration":
			normalized = "duration_ms"
		case strings.HasSuffix(key, "_duration"):
			normalized = strings.TrimSuffix(key, "_duration") + "_duration_ms"
		case !strings.HasSuffix(key, "_ms"):
			normalized = key + "_ms"
		}
		return normalized, ms, true
	case slog.KindTime:
		return key, val.Time().UTC().Format(time.RFC3339Nano), true
	case slog.KindAny:
		v := val.Any()
		switch x := v.(type) {
		case error:
			return key, x.Error(), true
		case string:
			return key, strings.TrimSpace(x), true
		case time.Duration:
			ms := RoundMS(x).Milliseconds()
			normalized := key
			switch {
			case key == "duration":
				normalized = "duration_ms"
			case strings.HasSuffix(key, "_duration"):
				normalized = strings.TrimSuffix(key, "_duration") + "_duration_ms"
			case !strings.HasSuffix(key, "_ms"):
				normalized = key + "_ms"
			}
			return normalized, ms, true
		case fmt.Stringer:
			return key, x.String(), true
		case nil:
			return key, nil, false
		default:
			return key, fmt.Sprint(v), true
		}
	default:
		return key, val.Any(), true
	}
}

func sanitizeEnumerations(fields map[string]any) {
	if level, ok := stringField(fields, "level"); ok {
		fields["level"] = normalizeLevel(level)
	}

	if s, ok := stringField(fields, "status"); ok && s != "" {
		if normalized, valid := normalizeStatus(s); valid {
			fields["status"] = normalized
		} else {
			fields["status"] = s
		}
	}
	if c, ok := stringField(fields, "cache"); ok && c != "" {
		if normalized, valid := normalizeCache(c); valid {
			fields["cache"] = normalized
		} else {
			delete(fields, "cache")
		}
	}
	if o, ok := stringField(fields, "outcome"); ok && o != "" {
		if normalized, valid := normalizeOutcome(o); valid {
			fields["outcome"] = normalized
		} else {
			delete(fields, "outcome")
		}
	}
}

func pruneEmpty(fields map[string]any) {
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			if val == "" {
				delete(fields, k)
			}
		case fmt.Stringer:
			if val.String() == "" {
				delete(fields, k)
			}
		case nil:
			delete(fields, k)
		}
	}
}

func formatJSONLine(fields map[string]any, order []string) ([]byte, error) {
	buf := strings.Builder{}
	buf.WriteByte('{')
	first := true
	visited := make(map[string]struct{}, len(fields))
	writeField := func(k string, v any) error {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		buf.WriteString(strconv.Quote(k))
		buf.WriteByte(':')
		buf.Write(data)
		visited[k] = struct{}{}
		return nil
	}
	for _, key := range order {
		val, ok := fields[key]
		if !ok {
			continue
		}
		if err := writeField(key, val); err != nil {
			return nil, err
		}
	}

	var remaining []string
	for k := range fields {
		if _, seen := visited[k]; seen {
			continue
		}
		remaining = append(remaining, k)
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		if err := writeField(key, fields[key]); err != nil {
			return nil, err
		}
	}
	buf.WriteByte('}')
	return []byte(buf.String()), nil
}

func formatKVLine(fields map[string]any, order []string) []byte {
	keys := orderedKeys(fields, order)
	var b strings.Builder
	for i, key := range keys {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(formatValueKV(fields[key]))
	}
	return []byte(b.String())
}

func orderedKeys(fields map[string]any, order []string) []string {
	keys := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, key := range order {
		if _, ok := fields[key]; ok {
			keys = append(keys, key)
			seen[key] = struct{}{}
		}
	}
	prefixLen := len(keys)
	for key := range fields {
		if _, ok := seen[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	sort.Slice(keys[prefixLen:], func(i, j int) bool {
		return keys[prefixLen+i] < keys[prefixLen+j]
	})
	return keys
}

func formatValueKV(val any) string {
	switch v := val.(type) {
	case string:
		if v == "" {
			return v
		}
		if strings.IndexFunc(v, needsQuote) >= 0 {
			return strconv.Quote(v)
		}
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int, int64, uint64, float64:
		return fmt.Sprint(v)
	default:
		s := fmt.Sprint(v)
		if strings.IndexFunc(s, needsQuote) >= 0 {
			return strconv.Quote(s)
		}
		return s
	}
}

func needsQuote(r rune) bool {
	return r <= 32 || r == '=' || r == '"'
}

func stringField(fields map[string]any, key string) (string, bool) {
	v, ok := fields[key]
	if !ok {
		return "", false
	}
	switch val := v.(type) {
	case string:
		return val, true
	case fmt.Stringer:
		return val.String(), true
	default:
		return fmt.Sprint(val), true
	}
}

func addContextFields(ctx context.Context, fields map[string]any) {
	if ctx == nil {
		return
	}
	if rid := RIDFrom(ctx); rid != "" {
		if _, ok := fields["rid"]; !ok {
			fields["rid"] = rid
		}
	}
	if traceID := TraceIDFrom(ctx); traceID != "" {
		if _, ok := fields["trace_id"]; !ok {
			fields["trace_id"] = traceID
		}
	}
	if spanID := SpanIDFrom(ctx); spanID != "" {
		if _, ok := fields["span_id"]; !ok {
			fields["span_id"] = spanID
		}
	}
	if uid := UserIDFrom(ctx); uid != 0 {
		if _, ok := fields["user_id"]; !ok {
			fields["user_id"] = uid
		}
	}
	if updateID := UpdateIDFrom(ctx); updateID != 0 {
		if _, ok := fields["update_id"]; !ok {
			fields["update_id"] = updateID
		}
	}
	if cid := ChatIDFrom(ctx); cid != 0 {
		if _, ok := fields["chat_id"]; !ok {
			fields["chat_id"] = cid
		}
	}
	if hid := HandlerFrom(ctx); hid != "" {
		if _, ok := fields["handler"]; !ok {
			fields["handler"] = hid
		}
	}
}

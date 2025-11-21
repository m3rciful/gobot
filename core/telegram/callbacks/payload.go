package callbacks

import (
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"
)

// PayloadInt64 parses callback payload as int64.
func PayloadInt64(c tele.Context) (int64, error) {
	p := CallbackPayload(c)
	return strconv.ParseInt(p, 10, 64)
}

// PayloadInt parses callback payload as int.
func PayloadInt(c tele.Context) (int, error) {
	p := CallbackPayload(c)
	return strconv.Atoi(p)
}

// PayloadFloat64 parses callback payload as float64.
func PayloadFloat64(c tele.Context) (float64, error) {
	p := CallbackPayload(c)
	return strconv.ParseFloat(p, 64)
}

// PayloadParts splits the callback payload into parts using the given separator.
func PayloadParts(c tele.Context, sep string) ([]string, error) {
	p := CallbackPayload(c)
	if p == "" {
		return nil, strconv.ErrSyntax
	}
	return strings.Split(p, sep), nil
}

// PayloadTwoInt64 parses callback payload like "123|456" into two int64 values.
func PayloadTwoInt64(c tele.Context, sep string) (int64, int64, error) {
	parts, err := PayloadParts(c, sep)
	if err != nil {
		return 0, 0, err
	}
	if len(parts) != 2 {
		return 0, 0, strconv.ErrSyntax
	}
	a, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	b, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return a, b, nil
}

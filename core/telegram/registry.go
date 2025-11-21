package telegram

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/m3rciful/gobot/core/logger"
	"github.com/m3rciful/gobot/core/telegram/commands"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// Registry holds bot commands and callbacks.
type Registry struct {
	commands         map[string]commands.Command
	callbacks        map[string]tele.HandlerFunc
	callbacksMu      sync.RWMutex
	callbackNotFound tele.HandlerFunc
	textFallback     tele.HandlerFunc
}

// NewRegistry creates an empty Registry with default fallbacks.
func NewRegistry() *Registry {
	return &Registry{
		commands:  make(map[string]commands.Command),
		callbacks: make(map[string]tele.HandlerFunc),
		callbackNotFound: func(c tele.Context) error {
			_ = c.Respond(&tele.CallbackResponse{Text: "Unsupported action"})
			return nil
		},
	}
}

// RegisterCommand adds a new command.
func (r *Registry) RegisterCommand(name string, cmd commands.Command) {
	if r == nil || name == "" || cmd.Handler == nil || cmd.Description == "" {
		logger.TWire.LogAttrs(context.Background(), slog.LevelWarn, "register.command.skip",
			slog.String("name", name),
			slog.String("reason", "invalid"),
		)
		return
	}
	if name[0] != '/' {
		logger.TWire.LogAttrs(context.Background(), slog.LevelWarn, "register.command.skip",
			slog.String("name", name),
			slog.String("reason", "no_slash_prefix"),
		)
		return
	}
	if _, exists := r.commands[name]; exists {
		logger.TWire.LogAttrs(context.Background(), slog.LevelWarn, "register.command.duplicate",
			slog.String("name", name),
		)
		return
	}
	r.commands[name] = cmd
}

// ListCommands returns a slice of tele.Command, optionally filtering out hidden and admin-only commands.
func (r *Registry) ListCommands(visibleOnly bool) []tele.Command {
	var list []tele.Command
	for cmd, meta := range r.commands {
		if visibleOnly && (meta.Hidden || meta.AdminOnly) {
			continue
		}
		list = append(list, tele.Command{Text: cmd, Description: meta.Description})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Text < list[j].Text })
	return list
}

// LookupCommand searches for a command by name or its aliases and returns the canonical key with metadata if found.
func (r *Registry) LookupCommand(name string) (string, commands.Command, bool) {
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	if cmd, ok := r.commands[name]; ok {
		return name, cmd, true
	}
	for key, cmd := range r.commands {
		for _, alias := range cmd.Aliases {
			if alias == name || "/"+alias == name {
				return key, cmd, true
			}
		}
	}
	return "", commands.Command{}, false
}

// Commands returns all registered commands.
func (r *Registry) Commands() map[string]commands.Command {
	return r.commands
}

// RegisterCallback adds a callback handler mapped to its key.
func (r *Registry) RegisterCallback(key string, handler tele.HandlerFunc) error {
	if r == nil || key == "" || handler == nil {
		logger.TWire.LogAttrs(context.Background(), slog.LevelWarn, "register.callback.skip",
			slog.String("key", key),
			slog.Bool("handler_nil", handler == nil),
		)
		return errors.New("invalid callback registration")
	}
	r.callbacksMu.Lock()
	defer r.callbacksMu.Unlock()
	if _, exists := r.callbacks[key]; exists {
		logger.TWire.LogAttrs(context.Background(), slog.LevelWarn, "register.callback.duplicate",
			slog.String("key", key),
		)
		return fmt.Errorf("callback already registered: %s", key)
	}
	r.callbacks[key] = handler
	return nil
}

// GetCallback safely returns handler by key.
func (r *Registry) GetCallback(key string) (tele.HandlerFunc, bool) {
	r.callbacksMu.RLock()
	defer r.callbacksMu.RUnlock()
	h, ok := r.callbacks[key]
	return h, ok
}

// ListCallbacks returns sorted keys (for diagnostics).
func (r *Registry) ListCallbacks() []string {
	r.callbacksMu.RLock()
	defer r.callbacksMu.RUnlock()
	names := make([]string, 0, len(r.callbacks))
	for k := range r.callbacks {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// SetCallbackNotFound replaces the fallback handler for unknown callbacks.
func (r *Registry) SetCallbackNotFound(h tele.HandlerFunc) {
	if h != nil {
		r.callbackNotFound = h
	}
}

// CallbackNotFound returns the current fallback callback handler.
func (r *Registry) CallbackNotFound() tele.HandlerFunc {
	return r.callbackNotFound
}

// SetTextFallback sets a global fallback handler for unknown text messages.
func (r *Registry) SetTextFallback(h tele.HandlerFunc) {
	r.textFallback = h
}

// TextFallback returns the current text fallback handler.
func (r *Registry) TextFallback() tele.HandlerFunc {
	return r.textFallback
}

// InitBotCommands sets the Telegram bot commands shown in the command menu.
func InitBotCommands(bot *tele.Bot, reg *Registry) {
	commands := reg.ListCommands(true)
	if err := bot.SetCommands(commands); err != nil {
		logger.TWire.LogAttrs(context.Background(), slog.LevelError, "register.commands.set_failed",
			slog.String("err", err.Error()),
		)
	}
}

package state

import (
	"sync"

	"github.com/m3rciful/gobot/core/logger"
	tghelpers "github.com/m3rciful/gobot/core/telegram/helpers"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

type memoryManager struct {
	mu       sync.RWMutex
	sessions map[int64]*Session
}

// NewMemoryManager constructs an in-memory Manager implementation for tests and development.
func NewMemoryManager() Manager {
	return &memoryManager{
		sessions: make(map[int64]*Session),
	}
}

// Get returns the session for a user if it exists, otherwise returns a default idle session.
func (m *memoryManager) Get(userID int64) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if session, ok := m.sessions[userID]; ok {
		return session
	}

	return &Session{State: StateIdle, TempData: make(map[string]interface{})}
}

// Set updates the state for a user, creating a new session if necessary.
func (m *memoryManager) Set(userID int64, state State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[userID]
	if !ok {
		session = &Session{TempData: make(map[string]interface{})}
		m.sessions[userID] = session
	}
	session.State = state
}

// SetTemp stores a temporary key/value pair for the given user session.
func (m *memoryManager) SetTemp(userID int64, key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[userID]
	if !ok {
		session = &Session{TempData: make(map[string]interface{})}
		m.sessions[userID] = session
	}
	session.TempData[key] = value
}

// GetTemp retrieves a temporary value by key for the given user session.
func (m *memoryManager) GetTemp(userID int64, key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[userID]
	if !ok {
		return nil, false
	}
	val, ok := session.TempData[key]
	return val, ok
}

// GetTempInt64 retrieves a temporary value by key and asserts it as int64.
func (m *memoryManager) GetTempInt64(userID int64, key string) (int64, bool) {
	val, found := m.GetTemp(userID, key)
	if !found {
		return 0, false
	}
	v, ok := val.(int64)
	if !ok {
		return 0, false
	}
	return v, true
}

// ClearTemp removes a temporary key/value pair for the given user session.
func (m *memoryManager) ClearTemp(userID int64, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[userID]
	if ok {
		delete(session.TempData, key)
	}
}

// Clear removes the entire session for a user.
func (m *memoryManager) Clear(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, userID)
}

// SetState sets the FSM state for the given user.
func (m *memoryManager) SetState(userID int64, st State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sess, ok := m.sessions[userID]
	if !ok {
		sess = &Session{TempData: make(map[string]interface{})}
		m.sessions[userID] = sess
	}
	sess.State = st
}

// GetState returns the current FSM state of a user, or StateIdle if none exists.
func (m *memoryManager) GetState(userID int64) State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if sess, ok := m.sessions[userID]; ok {
		return sess.State
	}
	return StateIdle
}

// ClearState resets the FSM state to idle for a user without removing session data.
func (m *memoryManager) ClearState(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sess, ok := m.sessions[userID]; ok {
		sess.State = StateIdle
	}
}

// HasState checks if a user has an active state other than idle.
func (m *memoryManager) HasState(userID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[userID]
	return ok && sess.State != StateIdle
}

// InProgress reports whether the user currently has an active FSM state.
func (m *memoryManager) InProgress(userID int64) bool {
	return m.HasState(userID)
}

// ManagerHandler executes the handler function registered for the user's current state, if any.
func (m *memoryManager) ManagerHandler(c tele.Context) error {
	userID := c.Sender().ID
	current := m.GetState(userID)
	ctx := tghelpers.BuildContext(c)
	logger.Debug(ctx, "tg", "fsm.manager",
		slog.String("status", "ok"),
		slog.Int64("user_id", userID),
		slog.String("state", string(current)),
	)

	if handler, ok := fsmHandlers[current]; ok {
		return handler(c)
	}
	return nil
}

package state

import tele "gopkg.in/telebot.v4"

// State identifies a finite-state-machine step used in conversations.
type State string

const (
	// StateIdle indicates there is no active conversation with the user.
	StateIdle State = "idle"
)

// Session stores conversation state and temporary data for a user.
type Session struct {
	State    State
	TempData map[string]interface{}
}

// Manager orchestrates user sessions and FSM state transitions.
type Manager interface {
	Get(userID int64) *Session
	Set(userID int64, state State)
	SetTemp(userID int64, key string, value interface{})
	ClearTemp(userID int64, key string)
	GetTemp(userID int64, key string) (interface{}, bool)
	GetTempInt64(userID int64, key string) (int64, bool)
	Clear(userID int64)

	// Dialog state
	SetState(userID int64, st State)
	GetState(userID int64) State
	HasState(userID int64) bool
	ClearState(userID int64)

	InProgress(userID int64) bool
	ManagerHandler(c tele.Context) error
}

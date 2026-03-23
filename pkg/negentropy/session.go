package negentropy

import (
	"sync"
	"time"

	"github.com/illuzen/go-negentropy"
)

// SessionManager mantém os vetores em memória para reconciliação rápida
type SessionManager struct {
	sessions map[string]*SessionItem
	mu       sync.RWMutex
}

type SessionItem struct {
	Vector    *negentropy.Vector
	CreatedAt time.Time
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*SessionItem),
	}
	// Inicia rotina de limpeza automática
	go sm.cleanupRoutine()
	return sm
}

func (sm *SessionManager) Set(subID string, vec *negentropy.Vector) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[subID] = &SessionItem{
		Vector:    vec,
		CreatedAt: time.Now(),
	}
}

func (sm *SessionManager) Get(subID string) (*negentropy.Vector, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	item, ok := sm.sessions[subID]
	if !ok {
		return nil, false
	}
	// Renova TTL se acessado? Opcional. Aqui mantemos fixo.
	return item.Vector, true
}

func (sm *SessionManager) Delete(subID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, subID)
}

func (sm *SessionManager) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, item := range sm.sessions {
			// TTL de 5 minutos para uma sessão de sync
			if now.Sub(item.CreatedAt) > 5*time.Minute {
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}

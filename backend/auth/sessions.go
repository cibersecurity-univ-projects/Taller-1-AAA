package auth

import (
	"crypto/rand"
	"sync"
	"time"
)

type session struct {
	userID    int
	expiresAt time.Time
}

// Sessions guarda en memoria los tokens emitidos en el login. Un token es
// un valor aleatorio de 128 bits, imposible de adivinar o fabricar.
type Sessions struct {
	mu      sync.Mutex
	ttl     time.Duration
	byToken map[string]session
}

func NewSessions(ttl time.Duration) *Sessions {
	return &Sessions{ttl: ttl, byToken: map[string]session{}}
}

func (s *Sessions) Create(userID int) string {
	token := rand.Text()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byToken[token] = session{userID: userID, expiresAt: time.Now().Add(s.ttl)}
	return token
}

// Resolve devuelve el usuario dueño del token si este existe y no ha expirado.
func (s *Sessions) Resolve(token string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.byToken[token]
	if !ok {
		return 0, false
	}
	if time.Now().After(sess.expiresAt) {
		delete(s.byToken, token)
		return 0, false
	}
	return sess.userID, true
}

package main

import (
	"crypto/rand"
	"sync"
	"time"
)

type flash struct {
	Kind string // "ok" o "error"
	Text string
}

// session guarda el token del backend en el servidor de la interfaz: el
// navegador solo recibe un identificador opaco en una cookie.
type session struct {
	Token   string
	Email   string
	Role    string
	flash   *flash
	expires time.Time
}

type sessionStore struct {
	mu    sync.Mutex
	items map[string]*session
}

func newSessionStore() *sessionStore {
	return &sessionStore{items: map[string]*session{}}
}

func (s *sessionStore) create(token, email, role string) string {
	id := rand.Text()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[id] = &session{Token: token, Email: email, Role: role, expires: time.Now().Add(sessionTTL)}
	return id
}

func (s *sessionStore) get(id string) (session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.items[id]
	if !ok {
		return session{}, false
	}
	if time.Now().After(sess.expires) {
		delete(s.items, id)
		return session{}, false
	}
	return *sess, true
}

func (s *sessionStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)
}

func (s *sessionStore) setFlash(id string, f flash) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.items[id]; ok {
		sess.flash = &f
	}
}

// popFlash devuelve el aviso pendiente una sola vez.
func (s *sessionStore) popFlash(id string) *flash {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.items[id]
	if !ok {
		return nil
	}
	f := sess.flash
	sess.flash = nil
	return f
}

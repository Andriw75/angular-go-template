package auth

import (
	"strings"
	"sync"
	"time"
)

// LoginLimiter hace rate limiting por combinación IP + usuario. nginx ya corta
// floods por IP; esto frena ataques dirigidos a una cuenta concreta y
// credential stuffing distribuido.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempt
	max      int
	window   time.Duration
	block    time.Duration
}

type loginAttempt struct {
	count        int
	first        time.Time
	blockedUntil time.Time
}

func NewLoginLimiter(max int, window, block time.Duration) *LoginLimiter {
	if max < 1 {
		max = 5
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	if block <= 0 {
		block = 15 * time.Minute
	}
	return &LoginLimiter{
		attempts: make(map[string]*loginAttempt),
		max:      max,
		window:   window,
		block:    block,
	}
}

// Key normaliza la clave para que mayúsculas/espacios no permitan evadir el
// límite de una cuenta.
func LoginKey(ip, username string) string {
	return strings.ToLower(strings.TrimSpace(ip)) + "|" + strings.ToLower(strings.TrimSpace(username))
}

// Allowed indica si el intento puede continuar. Si no, devuelve el tiempo
// restante de bloqueo para el header Retry-After.
func (l *LoginLimiter) Allowed(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.pruneLocked(now)

	a, ok := l.attempts[key]
	if !ok {
		return true, 0
	}
	if now.Before(a.blockedUntil) {
		return false, time.Until(a.blockedUntil)
	}
	if now.Sub(a.first) > l.window {
		delete(l.attempts, key)
		return true, 0
	}
	return true, 0
}

// Fail registra un intento fallido y bloquea la clave al superar el máximo.
func (l *LoginLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	a, ok := l.attempts[key]
	if !ok || now.Sub(a.first) > l.window {
		l.attempts[key] = &loginAttempt{count: 1, first: now}
		return
	}
	a.count++
	if a.count >= l.max {
		a.blockedUntil = now.Add(l.block)
	}
}

// Reset limpia los intentos tras un login exitoso.
func (l *LoginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func (l *LoginLimiter) pruneLocked(now time.Time) {
	if len(l.attempts) < 1024 {
		return
	}
	for key, a := range l.attempts {
		if now.After(a.blockedUntil) && now.Sub(a.first) > l.window {
			delete(l.attempts, key)
		}
	}
}

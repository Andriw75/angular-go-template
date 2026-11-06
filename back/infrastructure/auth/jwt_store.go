package auth

import (
	"sync"
	"time"
)

// jwtStorePruneThreshold evita recorrer todo el mapa en cada operación.
const jwtStorePruneThreshold = 1024

type jwtEntry struct {
	userID    int64
	expiresAt time.Time
}

// JWTStore mantiene los tokens vigentes (jti) para permitir revocación por
// token y por usuario. Es estado en memoria: no sobrevive reinicios.
type JWTStore struct {
	mu         sync.Mutex
	tokens     map[string]jwtEntry
	userTokens map[int64]map[string]struct{}
}

func NewJWTStore() *JWTStore {
	return &JWTStore{
		tokens:     make(map[string]jwtEntry),
		userTokens: make(map[int64]map[string]struct{}),
	}
}

// Add registra un token vigente hasta expiresAt.
func (s *JWTStore) Add(jti string, userID int64, expiresAt time.Time) {
	if jti == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())
	s.removeLocked(jti)
	s.tokens[jti] = jwtEntry{userID: userID, expiresAt: expiresAt}
	s.trackUserLocked(userID, jti)
}

// Rotate reemplaza oldJTI por newJTI durante una renovación deslizante.
func (s *JWTStore) Rotate(oldJTI, newJTI string, userID int64, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if oldJTI != "" {
		s.removeLocked(oldJTI)
	}
	if newJTI == "" {
		return
	}
	s.pruneLocked(time.Now())
	s.tokens[newJTI] = jwtEntry{userID: userID, expiresAt: expiresAt}
	s.trackUserLocked(userID, newJTI)
}

// Remove revoca una sesión concreta (logout).
func (s *JWTStore) Remove(jti string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeLocked(jti)
}

// RemoveUserTokens revoca todas las sesiones de un usuario.
func (s *JWTStore) RemoveUserTokens(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for jti := range s.userTokens[userID] {
		delete(s.tokens, jti)
	}
	delete(s.userTokens, userID)
}

// Exists indica si el token sigue vigente. Las entradas expiradas se eliminan.
func (s *JWTStore) Exists(jti string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())

	entry, ok := s.tokens[jti]
	if !ok {
		return false
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		s.removeLocked(jti)
		return false
	}
	return true
}

// CountUserTokens cuenta las sesiones vigentes de un usuario.
func (s *JWTStore) CountUserTokens(userID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0
	for jti := range s.userTokens[userID] {
		entry, ok := s.tokens[jti]
		if !ok {
			continue
		}
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			s.removeLocked(jti)
			continue
		}
		count++
	}
	return count
}

func (s *JWTStore) trackUserLocked(userID int64, jti string) {
	if s.userTokens[userID] == nil {
		s.userTokens[userID] = make(map[string]struct{})
	}
	s.userTokens[userID][jti] = struct{}{}
}

func (s *JWTStore) removeLocked(jti string) {
	entry, ok := s.tokens[jti]
	if !ok {
		return
	}
	delete(s.tokens, jti)
	if set := s.userTokens[entry.userID]; set != nil {
		delete(set, jti)
		if len(set) == 0 {
			delete(s.userTokens, entry.userID)
		}
	}
}

func (s *JWTStore) pruneLocked(now time.Time) {
	if len(s.tokens) < jwtStorePruneThreshold {
		return
	}
	for jti, entry := range s.tokens {
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			s.removeLocked(jti)
		}
	}
}

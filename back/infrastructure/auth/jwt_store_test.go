package auth

import (
	"testing"
	"time"
)

func TestJWTStoreAddAndExists(t *testing.T) {
	store := NewJWTStore()
	store.Add("jti-1", 7, time.Now().Add(time.Minute))

	if !store.Exists("jti-1") {
		t.Fatal("se esperaba que el jti existiera")
	}
	if store.Exists("jti-otro") {
		t.Fatal("un jti no registrado no debe existir")
	}
}

func TestJWTStoreExpiresEntries(t *testing.T) {
	store := NewJWTStore()
	store.Add("jti-viejo", 1, time.Now().Add(10*time.Millisecond))

	time.Sleep(30 * time.Millisecond)

	if store.Exists("jti-viejo") {
		t.Fatal("un jti expirado no debe existir")
	}
	if got := store.CountUserTokens(1); got != 0 {
		t.Fatalf("sesiones vigentes = %d, se esperaba 0", got)
	}
}

func TestJWTStoreRotate(t *testing.T) {
	store := NewJWTStore()
	exp := time.Now().Add(time.Minute)

	store.Add("viejo", 3, exp)
	store.Add("nuevo", 3, exp)
	store.Rotate("viejo", "nuevo", 3, exp)

	if store.Exists("viejo") {
		t.Fatal("el jti viejo debió eliminarse al rotar")
	}
	if !store.Exists("nuevo") {
		t.Fatal("el jti nuevo debe existir")
	}
	if got := store.CountUserTokens(3); got != 1 {
		t.Fatalf("sesiones vigentes = %d, se esperaba 1", got)
	}
}

func TestJWTStoreRemoveUserTokens(t *testing.T) {
	store := NewJWTStore()
	exp := time.Now().Add(time.Minute)

	store.Add("a", 5, exp)
	store.Add("b", 5, exp)
	store.Add("c", 6, exp)

	store.RemoveUserTokens(5)

	if store.Exists("a") || store.Exists("b") {
		t.Fatal("se esperaban eliminados los jti del usuario 5")
	}
	if !store.Exists("c") {
		t.Fatal("los jti de otros usuarios no deben tocarse")
	}
}

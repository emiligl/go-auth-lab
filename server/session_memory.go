package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

var sessions = make(map[string]User)
var sessionsMu sync.RWMutex

func newMemorySessionID() string {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}

func createMemorySession(user User) string {
	sessionID := newMemorySessionID()

	sessionsMu.Lock()
	sessions[sessionID] = user
	sessionsMu.Unlock()

	return sessionID
}

func getMemorySession(sessionID string) (User, bool) {
	sessionsMu.RLock()
	user, exists := sessions[sessionID]
	sessionsMu.RUnlock()

	return user, exists
}

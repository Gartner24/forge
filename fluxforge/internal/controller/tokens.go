package controller

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/gartner24/forge/fluxforge/internal/mesh"
)

// newJoinToken generates a cryptographically random 32-byte join token.
func newJoinToken() (mesh.Token, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return mesh.Token{}, err
	}
	now := time.Now().UTC()
	return mesh.Token{
		Value:     base64.URLEncoding.EncodeToString(b),
		CreatedAt: now,
		ExpiresAt: now.Add(mesh.TokenTTL),
	}, nil
}

// newAuthToken generates a cryptographically random bearer token for node auth.
func newAuthToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

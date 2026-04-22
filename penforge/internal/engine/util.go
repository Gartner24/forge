package engine

import (
	"crypto/rand"
	"encoding/hex"
)

// containerSuffix returns a short random hex string for unique container names.
func containerSuffix() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

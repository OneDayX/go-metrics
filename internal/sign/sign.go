// Package sign computes and checks the HMAC-SHA256 signature that the agent
// and the server attach to the bodies they exchange.
package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Header carries the hex-encoded signature of the message body.
const Header = "HashSHA256"

// Sum returns hash(data, key) as a hex string.
func Sum(data []byte, key string) string {
	return hex.EncodeToString(mac(data, key))
}

// Valid reports whether signature is hash(data, key). The comparison takes
// constant time, so it does not reveal how much of a forged hash was right.
func Valid(data []byte, key, signature string) bool {
	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	return hmac.Equal(got, mac(data, key))
}

func mac(data []byte, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return h.Sum(nil)
}

package pkce

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func codeVerifier() (string, error) {
	bs := make([]byte, 32)
	_, err := rand.Read(bs)
	if err != nil {
		return "", err
	}
	str := base64.RawURLEncoding.EncodeToString(bs)
	return str, nil
}

func codeChallenge(verifier string) string {
	bs := sha256.Sum256([]byte(verifier))
	str := base64.RawURLEncoding.EncodeToString(bs[:])
	return str
}

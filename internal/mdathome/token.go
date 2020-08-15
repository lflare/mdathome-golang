package mdathome

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/nacl/box"
)

func verifyToken(tokenString string, chapterHash string) (int, error) {
	// Check if given token string is empty
	if tokenString == "" {
		return 403, fmt.Errorf("Token is empty")
	}

	// Decode base64-encoded token & key
	tokenBytes, err := base64.RawURLEncoding.DecodeString(tokenString)
	if err != nil {
		return 403, fmt.Errorf("Cannot decode token - %v", err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(serverResponse.TokenKey)
	if err != nil {
		return 403, fmt.Errorf("Cannot decode key - %v", err)
	}

	// Copy over byte slices to fixed-length byte arrays for decryption
	var nonce [24]byte
	copy(nonce[:], tokenBytes[:24])
	var key [32]byte
	copy(key[:], keyBytes[:32])

	// Decrypt token
	data, ok := box.OpenAfterPrecomputation(nil, tokenBytes[24:], &nonce, &key)
	if !ok {
		return 403, fmt.Errorf("Failed to decrypt token")
	}

	// Unmarshal to struct
	token := Token{}
	if err := json.Unmarshal(data, &token); err != nil {
		return 403, fmt.Errorf("Failed to unmarshal token - %v", err)
	}

	// Parse expiry time
	expires, err := time.Parse(time.RFC3339, token.Expires)
	if err != nil {
		return 403, fmt.Errorf("Failed to parse expiry from token - %v", err)
	}

	// Check token expiry timing
	if time.Now().After(expires) {
		return 410, fmt.Errorf("Token expired")
	}

	// Check that chapter hashes are the same
	if token.Hash != chapterHash {
		return 403, fmt.Errorf("Token hash invalid")
	}

	// Token is valid
	return 0, nil
}

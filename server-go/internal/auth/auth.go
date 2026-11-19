// Package auth implements the v2 challenge-response authentication.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

const (
	// KDFPBKDF2SHA256 is the wire value for PBKDF2-HMAC-SHA256.
	KDFPBKDF2SHA256 uint8 = 1
	// DefaultIterations is the server's default PBKDF2 work factor.
	DefaultIterations = 210000
	// SaltLen and NonceLen are the standard random byte counts.
	SaltLen     = 16
	NonceLen    = 16
	VerifierLen = 32
	ProofLen    = 32
)

const proofDomain = "vxaos-login-v1"

// Derive computes a PBKDF2-HMAC-SHA256 verifier.
func Derive(password string, salt []byte, iterations, keyLen int) ([]byte, error) {
	if iterations <= 0 {
		return nil, fmt.Errorf("auth: invalid iteration count %d", iterations)
	}
	if keyLen <= 0 {
		return nil, fmt.Errorf("auth: invalid key length %d", keyLen)
	}
	return pbkdf2Key([]byte(password), salt, iterations, keyLen), nil
}

// NewVerifier generates a random salt and derives a verifier.
func NewVerifier(password string, iterations int) (salt, verifier []byte, err error) {
	salt, err = RandomBytes(SaltLen)
	if err != nil {
		return nil, nil, err
	}
	verifier, err = Derive(password, salt, iterations, VerifierLen)
	if err != nil {
		return nil, nil, err
	}
	return salt, verifier, nil
}

// Proof computes the client's challenge response.
func Proof(verifier []byte, username string, nonce, clientNonce []byte) []byte {
	mac := hmac.New(sha256.New, verifier)
	mac.Write([]byte(proofDomain))
	writeField(mac, []byte(username))
	writeField(mac, nonce)
	writeField(mac, clientNonce)
	return mac.Sum(nil)
}

// VerifyProof checks a proof in constant time.
func VerifyProof(verifier []byte, username string, nonce, clientNonce, proof []byte) bool {
	want := Proof(verifier, username, nonce, clientNonce)
	return hmac.Equal(want, proof)
}

const deleteDomain = "vxaos-delete-v1"

// DeleteProof proves the account password for destructive operations without
// sending it. It is bound to the login nonce, a fresh client nonce and the
// character name.
func DeleteProof(verifier []byte, username string, loginNonce, clientNonce []byte, characterName string) []byte {
	mac := hmac.New(sha256.New, verifier)
	mac.Write([]byte(deleteDomain))
	writeField(mac, []byte(username))
	writeField(mac, loginNonce)
	writeField(mac, clientNonce)
	writeField(mac, []byte(characterName))
	return mac.Sum(nil)
}

// VerifyDeleteProof checks a deletion proof in constant time.
func VerifyDeleteProof(verifier []byte, username string, loginNonce, clientNonce []byte, characterName string, proof []byte) bool {
	want := DeleteProof(verifier, username, loginNonce, clientNonce, characterName)
	return hmac.Equal(want, proof)
}

// RandomBytes returns n cryptographically random bytes.
func RandomBytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("auth: invalid random length %d", n)
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("auth: random: %w", err)
	}
	return b, nil
}

func writeField(mac interface{ Write([]byte) (int, error) }, b []byte) {
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(b)))
	mac.Write(lenBuf[:])
	mac.Write(b)
}

// pbkdf2Key is PBKDF2-HMAC-SHA256. It is implemented here to keep the auth
// package independent of (and pinned to) the protocol's behavior.
func pbkdf2Key(password, salt []byte, iterations, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen
	var out []byte
	var block [4]byte
	for i := 1; i <= numBlocks; i++ {
		prf.Reset()
		prf.Write(salt)
		binary.BigEndian.PutUint32(block[:], uint32(i))
		prf.Write(block[:])
		u := prf.Sum(nil)
		t := append([]byte(nil), u...)
		for j := 1; j < iterations; j++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for k := range t {
				t[k] ^= u[k]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

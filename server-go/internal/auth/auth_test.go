package auth

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestPBKDF2Vector(t *testing.T) {
	got, err := Derive("password", []byte("salt"), 4096, 32)
	if err != nil {
		t.Fatal(err)
	}
	want := "c5e478d59288c841aa530db6845c4c8d962893a001ce4e11a4963873aa98134a"
	if hex.EncodeToString(got) != want {
		t.Fatalf("PBKDF2 = %x, want %s", got, want)
	}
}

func TestProofVector(t *testing.T) {
	verifier, err := Derive("password", []byte("salt"), 4096, 32)
	if err != nil {
		t.Fatal(err)
	}
	proof := Proof(verifier, "admin", []byte("0123456789abcdef"), []byte("fedcba9876543210"))
	want := "50453a7ad16fe7b128d59edba2698447a83e48c55e84794d7cb1ac87a70e4fbc"
	if hex.EncodeToString(proof) != want {
		t.Fatalf("proof = %x, want %s", proof, want)
	}
	if !VerifyProof(verifier, "admin", []byte("0123456789abcdef"), []byte("fedcba9876543210"), proof) {
		t.Fatal("valid proof rejected")
	}
	if VerifyProof(verifier, "ADMIN", []byte("0123456789abcdef"), []byte("fedcba9876543210"), proof) {
		t.Fatal("proof accepted for different username")
	}
	if VerifyProof(verifier, "admin", []byte("0123456789abcdee"), []byte("fedcba9876543210"), proof) {
		t.Fatal("proof accepted for different nonce")
	}
}

func TestNewVerifier(t *testing.T) {
	salt, verifier, err := NewVerifier("secret", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(salt) != SaltLen || len(verifier) != VerifierLen {
		t.Fatalf("sizes = %d/%d", len(salt), len(verifier))
	}
	again, err := Derive("secret", salt, 1000, VerifierLen)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(hex.EncodeToString(again), hex.EncodeToString(verifier)) {
		t.Fatal("derive is not deterministic")
	}
	other, _, err := NewVerifier("secret", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(other) == hex.EncodeToString(salt) {
		t.Fatal("salt repeated")
	}
}

func TestInvalidParameters(t *testing.T) {
	if _, err := Derive("x", []byte("salt"), 0, 32); err == nil {
		t.Fatal("zero iterations accepted")
	}
	if _, err := Derive("x", []byte("salt"), 1, 0); err == nil {
		t.Fatal("zero key length accepted")
	}
	if _, err := RandomBytes(0); err == nil {
		t.Fatal("zero random length accepted")
	}
}

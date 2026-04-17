package cryptoutil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	plaintext := []byte(`{"id":"Alloc","type":"gauge","value":42.5}`)

	encrypted, err := Encrypt(&privateKey.PublicKey, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if bytes.Contains(encrypted, plaintext) {
		t.Fatalf("encrypted payload contains plaintext")
	}

	decrypted, err := Decrypt(privateKey, encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptInvalidPayload(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	if _, err := Decrypt(privateKey, []byte("not-json")); err == nil {
		t.Fatal("Decrypt() error = nil, want error")
	}
}

func TestLoadPublicKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}

	path := filepath.Join(t.TempDir(), "public.pem")
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	})

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	publicKey, err := LoadPublicKey(path)
	if err != nil {
		t.Fatalf("LoadPublicKey() error = %v", err)
	}

	if publicKey.N.Cmp(privateKey.PublicKey.N) != 0 {
		t.Fatal("loaded public key does not match original key")
	}
}

func TestLoadPrivateKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	path := filepath.Join(t.TempDir(), "private.pem")
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	loadedKey, err := LoadPrivateKey(path)
	if err != nil {
		t.Fatalf("LoadPrivateKey() error = %v", err)
	}

	if loadedKey.N.Cmp(privateKey.N) != 0 {
		t.Fatal("loaded private key does not match original key")
	}
}

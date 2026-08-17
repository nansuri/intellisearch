package services

import "testing"

func TestEncryptSecretRoundTrip(t *testing.T) {
	key := []byte("test-encryption-key-for-aes-gcm")
	sealed, err := EncryptSecret("sk-secret-token", key)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := DecryptSecret(sealed, key)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "sk-secret-token" {
		t.Fatalf("unexpected plaintext: %q", plain)
	}
}

func TestEncryptSecretWrongKeyFails(t *testing.T) {
	sealed, err := EncryptSecret("secret", []byte("key-one"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptSecret(sealed, []byte("key-two")); err == nil {
		t.Fatal("expected decryption with the wrong key to fail")
	}
}

func TestEncryptSecretProducesUniqueCiphertexts(t *testing.T) {
	key := []byte("test-encryption-key-for-aes-gcm")
	first, _ := EncryptSecret("same", key)
	second, _ := EncryptSecret("same", key)
	if first == second {
		t.Fatal("expected distinct ciphertexts (random nonce)")
	}
}

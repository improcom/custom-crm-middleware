package crypt

import (
	"crm-middleware/db/settings"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

type CipherText string

func (ct *CipherText) Encrypt(plain string) {
	*ct = ""

	// instance-id is 32-byte by design and never changes, making it a stable AES key
	block, err := aes.NewCipher([]byte(settings.Get[string](settings.Key("sys", "instance-id"))))
	if err != nil {
		return
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return
	}

	*ct = CipherText(base64.StdEncoding.EncodeToString(
		gcm.Seal(nonce, nonce, []byte(plain), nil),
	))
}

func (ct CipherText) Decrypt() string {
	ciphertext, err := base64.StdEncoding.DecodeString(string(ct))
	if err != nil {
		return ""
	}

	block, err := aes.NewCipher([]byte(settings.Get[string](settings.Key("sys", "instance-id"))))
	if err != nil {
		return ""
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return ""
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ""
	}
	return string(plain)
}

const randcharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const randmodulo = byte(len(randcharset) - 1)

// Random returns a cryptographically random alphanumeric string of length n (clamped to [1, 64]).
// If a prefix is given and len(prefix)+1 < n, the result is prefix + '_' + random suffix.
// Otherwise the prefix is ignored and a fully random string is returned.
func Random(n int, prefix ...string) string {
	n = min(max(1, n), 64)
	b := make([]byte, n)
	rand.Read(b)
	for i, c := range b {
		b[i] = randcharset[c&randmodulo]
	}
	if len(prefix) > 0 && len(prefix[0]) > 0 && n > len(prefix[0])+1 {
		p := prefix[0]
		copy(b, p)
		b[len(p)] = '_'
	}
	return string(b)
}

package crypt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// writeFile saves data to a temporary file and returns its path.
func writeFile(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

func TestEncryptDecrypt(t *testing.T) {
	key := newKey(t)
	data := []byte(`[{"id":"Alloc","type":"gauge","value":123.45}]`)

	encrypted, err := Encrypt(data, &key.PublicKey)
	require.NoError(t, err)
	assert.NotContains(t, string(encrypted), "Alloc")

	decrypted, err := Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted)

	_, err = Decrypt(encrypted, newKey(t))
	assert.Error(t, err, "a key from another pair")
}

func TestLoadKeys(t *testing.T) {
	key := newKey(t)

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	privatePath := writeFile(t, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	der, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	publicPath := writeFile(t, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))

	private, err := LoadPrivateKey(privatePath)
	require.NoError(t, err)
	assert.True(t, key.Equal(private))

	public, err := LoadPublicKey(publicPath)
	require.NoError(t, err)
	assert.True(t, key.PublicKey.Equal(public))
}

func TestLoadPublicKey_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "not PEM", data: []byte("not a key")},
		{name: "private key file", data: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("key")})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadPublicKey(writeFile(t, tt.data))
			assert.Error(t, err)
		})
	}
}

package common

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
)

func TestPasswordHashCompatibility(t *testing.T) {
	password := " 密码 compatibility "
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(password), salt, 2, 19456, 1, 32)
	encoded := "$argon2id$v=19$m=19456,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
	legacy, err := Password2Hash(password)
	require.NoError(t, err)
	for _, hash := range []string{legacy, encoded} {
		assert.True(t, ValidatePasswordAndHash(password, hash))
		assert.False(t, ValidatePasswordAndHash("wrong", hash))
		assert.False(t, ValidatePasswordAndHash(strings.TrimSpace(password), hash))
	}
	for _, hash := range []string{
		strings.Replace(encoded, "m=19456", "m=4294967295", 1),
		strings.Replace(encoded, "t=2", "t=999999999", 1),
		strings.Replace(encoded, "p=1", "p=0", 1),
		strings.Replace(encoded, "v=19", "v=16", 1),
		encoded + "=", "$argon2id$", strings.Repeat("x", 257),
	} {
		assert.False(t, ValidatePasswordAndHash(password, hash))
	}
	assert.False(t, ValidatePasswordAndHash(strings.Repeat("x", 513), encoded))
	assert.True(t, strings.HasPrefix(legacy, "$2"), "本轮仍写入 bcrypt")
}

package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConsumeVerificationCodeUsesOnlyOneMatchingUnexpiredCode(t *testing.T) {
	const key = "email-bind-once@example.com"
	RegisterVerificationCodeWithKey(key, "123456", EmailVerificationPurpose)
	t.Cleanup(func() { DeleteKey(key, EmailVerificationPurpose) })
	assert.False(t, ConsumeVerificationCode(key, "123456", PasswordResetPurpose))
	assert.False(t, ConsumeVerificationCode(key, "wrong", EmailVerificationPurpose))
	assert.True(t, ConsumeVerificationCode(key, "123456", EmailVerificationPurpose))
	assert.False(t, VerifyCodeWithKey(key, "123456", EmailVerificationPurpose))
	assert.False(t, ConsumeVerificationCode(key, "123456", EmailVerificationPurpose))

	RegisterVerificationCodeWithKey(key, "654321", EmailVerificationPurpose)
	assert.False(t, ConsumeVerificationCode(key, "123456", EmailVerificationPurpose))
	assert.True(t, ConsumeVerificationCode(key, "654321", EmailVerificationPurpose))
}

func TestConsumeVerificationCodeRejectsExpiredCode(t *testing.T) {
	const key = "email-bind-expired@example.com"
	verificationMutex.Lock()
	verificationMap[EmailVerificationPurpose+key] = verificationValue{
		code: "123456", time: time.Now().Add(-time.Duration(VerificationValidMinutes) * time.Minute),
	}
	verificationMutex.Unlock()
	t.Cleanup(func() { DeleteKey(key, EmailVerificationPurpose) })
	assert.False(t, ConsumeVerificationCode(key, "123456", EmailVerificationPurpose))
}

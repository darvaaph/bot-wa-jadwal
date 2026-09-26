package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinimumPasswordLength = 12
	PasswordHashCost      = 12
)

func NormalizeIdentity(identity string) string {
	return strings.ToLower(strings.TrimSpace(identity))
}

func ValidatePassword(identity, password string) error {
	if utf8.RuneCountInString(password) < MinimumPasswordLength {
		return fmt.Errorf("%w: kata sandi minimal %d karakter", ErrInvalidInput, MinimumPasswordLength)
	}
	if strings.EqualFold(strings.TrimSpace(password), NormalizeIdentity(identity)) {
		return fmt.Errorf("%w: kata sandi tidak boleh sama dengan identitas", ErrInvalidInput)
	}
	return nil
}

func HashPassword(identity, password string) (string, error) {
	if err := ValidatePassword(identity, password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), PasswordHashCost)
	if err != nil {
		return "", fmt.Errorf("membuat hash kata sandi: %w", err)
	}
	return string(hash), nil
}

func bcryptHashForDummyPassword() (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte("authentication-dummy-password"), PasswordHashCost)
	if err != nil {
		return "", fmt.Errorf("membuat dummy hash kata sandi: %w", err)
	}
	return string(hash), nil
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func keyedHash(key []byte, namespace, value string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(namespace))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

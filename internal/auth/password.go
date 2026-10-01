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
	s := strings.TrimSpace(identity)
	if s == "" {
		return ""
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return strings.ToLower(s)
		}
	}
	if strings.Contains(s, "@") {
		return strings.ToLower(s)
	}
	hasPlus := strings.HasPrefix(s, "+")
	var digits strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	if d == "" {
		return strings.ToLower(s)
	}
	if strings.HasPrefix(d, "00") && len(d) > 2 {
		d = d[2:]
		hasPlus = true
	}
	var canonical string
	switch {
	case strings.HasPrefix(d, "62"):
		canonical = "+" + d
	case strings.HasPrefix(d, "0"):
		canonical = "+62" + d[1:]
	case strings.HasPrefix(d, "8"):
		canonical = "+62" + d
	default:
		if len(d) >= 9 && len(d) <= 15 && hasPlus {
			canonical = "+" + d
		} else {
			return strings.ToLower(s)
		}
	}
	return canonical
}

// IsValidPhoneIdentity melaporkan apakah identitas sudah dalam bentuk
// kanonis WhatsApp Indonesia (+62... dengan digit cukup).
func IsValidPhoneIdentity(identity string) bool {
	s := strings.TrimSpace(identity)
	if !strings.HasPrefix(s, "+62") {
		return false
	}
	digits := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits++
		} else if r != '+' {
			return false
		}
	}
	return digits >= 10 && digits <= 15
}

func ValidatePassword(identity, password string) error {
	if utf8.RuneCountInString(password) < MinimumPasswordLength {
		return fmt.Errorf("%w: kata sandi minimal %d karakter", ErrInvalidInput, MinimumPasswordLength)
	}
	normalizedIdentity := NormalizeIdentity(identity)
	if strings.EqualFold(strings.TrimSpace(password), normalizedIdentity) {
		return fmt.Errorf("%w: kata sandi tidak boleh sama dengan identitas", ErrInvalidInput)
	}
	if normalizedPassword := NormalizeIdentity(password); normalizedPassword != "" && normalizedPassword == normalizedIdentity {
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
	return KeyedHash(key, namespace, value)
}

// KeyedHash menghitung HMAC-SHA256 namespaced untuk fingerprint identity
// dan source rate limiter. Diekspor agar server API memakai helper kanonis
// yang sama (BE-012/BE-013); jangan dipakai untuk enkripsi password atau
// sebagai session token.
func KeyedHash(key []byte, namespace, value string) string {
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

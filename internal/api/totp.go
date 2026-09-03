package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

var b32Encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateTOTPSecret returns a 20-byte (160-bit) random secret as an unpadded Base32 string.
func GenerateTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return b32Encoding.EncodeToString(buf), nil
}

// GenerateTOTP generates a 6-digit TOTP code for the given secret at time t (RFC 6238).
func GenerateTOTP(secret string, t time.Time) (string, error) {
	key, err := b32Encoding.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", fmt.Errorf("invalid base32 secret: %w", err)
	}

	counter := uint64(t.Unix() / 30)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	h := mac.Sum(nil)

	offset := h[len(h)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(h[offset:offset+4]) & 0x7fffffff
	code := truncated % 1000000

	return fmt.Sprintf("%06d", code), nil
}

// ValidateTOTP validates a 6-digit code against a secret with +/- 1 time step tolerance (RFC 6238).
func ValidateTOTP(secret, code string, t time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}

	for step := -1; step <= 1; step++ {
		targetTime := t.Add(time.Duration(step*30) * time.Second)
		expected, err := GenerateTOTP(secret, targetTime)
		if err == nil && subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

// BuildOTPAuthURI constructs the standard otpauth:// URL for authenticator apps.
func BuildOTPAuthURI(username, secret string) string {
	label := url.PathEscape(fmt.Sprintf("Aptify:%s", username))
	params := url.Values{}
	params.Set("secret", strings.ToUpper(strings.TrimSpace(secret)))
	params.Set("issuer", "Aptify")
	params.Set("algorithm", "SHA1")
	params.Set("digits", "6")
	params.Set("period", "30")

	return fmt.Sprintf("otpauth://totp/%s?%s", label, params.Encode())
}

// GenerateQRCodeDataURI generates a PNG QR code encoded as a data URI.
func GenerateQRCodeDataURI(otpURI string) (string, error) {
	png, err := qrcode.Encode(otpURI, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("generate qr code: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

// GenerateRecoveryCodes generates count random recovery codes (e.g. "a1b2-c3d4")
// and returns both the plaintext codes (for one-time user display) and bcrypt hashes (for storage).
func GenerateRecoveryCodes(count int) (plain []string, hashed []string, err error) {
	plain = make([]string, count)
	hashed = make([]string, count)

	for i := 0; i < count; i++ {
		buf := make([]byte, 4)
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, fmt.Errorf("read random bytes: %w", err)
		}
		code := fmt.Sprintf("%02x%02x-%02x%02x", buf[0], buf[1], buf[2], buf[3])
		h, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, fmt.Errorf("hash recovery code: %w", err)
		}
		plain[i] = code
		hashed[i] = string(h)
	}
	return plain, hashed, nil
}

package token

import (
	"context"
	crypto "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	math "math/rand"
	"time"

	"github.com/geruz/rizotto/logger"
)

const charset = "0123456789abcdef"
const hexBytesPerChar = 2

var ErrOddLength = errors.New("token length must be positive and even")

var seededRand *math.Rand = math.New(math.NewSource(time.Now().UnixNano())) //nolint:gosec

func fallback(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

func cryptoGen(length int) (string, error) {
	buf := make([]byte, length/hexBytesPerChar)
	_, err := crypto.Read(buf)
	if err == nil {
		return hex.EncodeToString(buf), nil
	}

	return "", err
}

func Generate(ctx context.Context, length int) string {
	token, err := cryptoGen(length)
	if err == nil {
		return token
	}
	logger.Warn(ctx, "Failed to crypto token, go to fallback "+err.Error())

	return fallback(length)
}

func Hash(value string) string {
	digest := sha256.Sum256([]byte(value))

	return hex.EncodeToString(digest[:])
}

func Secure(length int) (string, error) {
	if length <= 0 || length%hexBytesPerChar != 0 {
		return "", fmt.Errorf("%w, got %d", ErrOddLength, length)
	}

	token, err := cryptoGen(length)
	if err != nil {
		return "", fmt.Errorf("generate secure token: %w", err)
	}

	return token, nil
}

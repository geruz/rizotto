package token

import (
	"context"
	crypto "crypto/rand"
	"encoding/hex"
	math "math/rand"
	"time"

	"github.com/geruz/rizotto/logger"
)

const charset = "0123456789abcdef"

var seededRand *math.Rand = math.New(math.NewSource(time.Now().UnixNano())) //nolint:gosec

func fallback(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

func cryptoGen(length int) (string, error) {
	buf := make([]byte, length/2)
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

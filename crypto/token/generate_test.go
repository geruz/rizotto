package token

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	ctx := context.Background()
	token := Generate(ctx, 40)
	assert.Len(t, token, len("ae3ecf44d9fc5bf66a8825df24c3223b18cde2d8"))
}

func TestFallback(t *testing.T) {
	token1 := fallback(40)
	assert.Len(t, token1, len("ae3ecf44d9fc5bf66a8825df24c3223b18cde2d8"))
	token2 := fallback(40)
	assert.NotEqual(t, token1, token2)
}

func TestCrypto(t *testing.T) {
	token1, _ := cryptoGen(40)
	assert.Len(t, token1, len("ae3ecf44d9fc5bf66a8825df24c3223b18cde2d8"))
	token2, _ := cryptoGen(40)
	assert.NotEqual(t, token1, token2)
}

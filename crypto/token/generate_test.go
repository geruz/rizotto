package token

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	token := Generate(ctx, 40)
	assert.Len(t, token, len("ae3ecf44d9fc5bf66a8825df24c3223b18cde2d8"))
}

func TestFallback(t *testing.T) {
	t.Parallel()

	token1 := fallback(40)
	assert.Len(t, token1, len("ae3ecf44d9fc5bf66a8825df24c3223b18cde2d8"))
	token2 := fallback(40)
	assert.NotEqual(t, token1, token2)
}

func TestCrypto(t *testing.T) {
	t.Parallel()

	token1, _ := cryptoGen(40)
	assert.Len(t, token1, len("ae3ecf44d9fc5bf66a8825df24c3223b18cde2d8"))
	token2, _ := cryptoGen(40)
	assert.NotEqual(t, token1, token2)
}

func TestSecure(t *testing.T) {
	t.Parallel()

	token1, err := Secure(40)
	require.NoError(t, err)
	assert.Len(t, token1, len("ae3ecf44d9fc5bf66a8825df24c3223b18cde2d8"))

	token2, err := Secure(40)
	require.NoError(t, err)
	assert.NotEqual(t, token1, token2)
}

// TestSecureRefusesLengthItCannotProduce pins that Secure never answers a token
// shorter than asked: two hex characters come from one random byte.
func TestSecureRefusesLengthItCannotProduce(t *testing.T) {
	t.Parallel()

	for _, length := range []int{0, -2, 41} {
		token, err := Secure(length)
		require.ErrorIs(t, err, ErrOddLength)
		assert.Empty(t, token)
	}
}

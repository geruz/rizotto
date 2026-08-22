package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashMatches(t *testing.T) {
	t.Parallel()

	hash, err := Hash("correct horse battery staple")
	require.NoError(t, err)
	assert.NotEqual(t, "correct horse battery staple", hash)
	assert.True(t, Matches(hash, "correct horse battery staple"))
	assert.False(t, Matches(hash, "Correct horse battery staple"))
}

// TestHashIsSaltedPerCall pins that two hashes of the same password differ, so
// that equal passwords cannot be spotted in the database.
func TestHashIsSaltedPerCall(t *testing.T) {
	t.Parallel()

	first, err := Hash("same password")
	require.NoError(t, err)
	second, err := Hash("same password")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.True(t, Matches(first, "same password"))
	assert.True(t, Matches(second, "same password"))
}

func TestHashRefusesEmptyPassword(t *testing.T) {
	t.Parallel()

	hash, err := Hash("")
	require.ErrorIs(t, err, ErrEmptyPassword)
	assert.Empty(t, hash)
}

// TestHashRefusesTruncatablePassword pins that a password bcrypt would silently
// cut short is refused instead, so nobody authenticates with its first 72 bytes.
func TestHashRefusesTruncatablePassword(t *testing.T) {
	t.Parallel()

	hash, err := Hash(strings.Repeat("a", maxLength+1))
	require.ErrorIs(t, err, ErrPasswordTooLong)
	assert.Empty(t, hash)
}

func TestMatchesRejectsGarbageHash(t *testing.T) {
	t.Parallel()

	assert.False(t, Matches("", "any password"))
	assert.False(t, Matches("not a bcrypt hash", "any password"))
}

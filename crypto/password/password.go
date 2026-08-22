package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const cost = 12
const maxLength = 72

var (
	ErrEmptyPassword   = errors.New("password is empty")
	ErrPasswordTooLong = fmt.Errorf("password is longer than %d bytes", maxLength)
)

func Hash(plain string) (string, error) {
	if plain == "" {
		return "", ErrEmptyPassword
	}

	if len(plain) > maxLength {
		return "", ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return string(hash), nil
}

func Matches(hash, plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))

	return err == nil
}

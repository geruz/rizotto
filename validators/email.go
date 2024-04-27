package validators

import "net/mail"

func Email(email string) bool {
	_, err := mail.ParseAddress(email)

	return err == nil
}

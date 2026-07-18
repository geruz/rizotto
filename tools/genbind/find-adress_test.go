package main

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

type findBindAddressExpected struct {
	address             string
	isEvent             bool
	isAuthTokenRequired bool
	tokenScope          string
}

func TestFindBindAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		commentText string
		expected    findBindAddressExpected
	}{
		{
			commentText: "// bind-method: http://kyc-service/kyc-credentials/get",
			expected: findBindAddressExpected{
				address:             "http://kyc-service/kyc-credentials/get",
				isEvent:             false,
				isAuthTokenRequired: false,
				tokenScope:          "",
			},
		},
		{
			commentText: "// bind-event: rabbitmq://gamification-service/event/register?worker_count=10",
			expected: findBindAddressExpected{
				address:             "rabbitmq://gamification-service/event/register",
				isEvent:             true,
				isAuthTokenRequired: false,
				tokenScope:          "",
			},
		},
		{
			commentText: "// bind-method: http://kyc-service/verification-files/get?auth_token=required",
			expected: findBindAddressExpected{
				address:             "http://kyc-service/verification-files/get",
				isEvent:             false,
				isAuthTokenRequired: true,
				tokenScope:          "",
			},
		},
		{
			commentText: "// bind-method: http://kyc-service/verification-files/get?auth_token=boa",
			expected: findBindAddressExpected{
				address:             "http://kyc-service/verification-files/get",
				isEvent:             false,
				isAuthTokenRequired: true,
				tokenScope:          "boa",
			},
		},
	}

	for _, test := range tests {
		comment := &ast.Comment{Slash: token.NoPos, Text: test.commentText}

		address, isEvent, isAuthTokenRequired, tokenScope := findBindAddress(comment)

		actual := findBindAddressExpected{
			address:             address,
			isEvent:             isEvent,
			isAuthTokenRequired: isAuthTokenRequired,
			tokenScope:          tokenScope,
		}
		assert.Equal(t, test.expected, actual, test.commentText)
	}
}

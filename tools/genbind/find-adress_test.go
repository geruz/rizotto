package main

import (
	"go/ast"
	"testing"
)

func TestFindBindAddress(t *testing.T) {
	type Expected struct {
		address             string
		isEvent             bool
		isAuthTokenRequired bool
		tokenScope          string
	}

	tests := []struct {
		commentText string
		expected    Expected
	}{
		{
			commentText: "// bind-method: http://kyc-service/kyc-credentials/get",
			expected: Expected{
				address:             "http://kyc-service/kyc-credentials/get",
				isEvent:             false,
				isAuthTokenRequired: false,
				tokenScope:          "",
			},
		},
		{
			commentText: "// bind-event: rabbitmq://gamification-service/event/register?worker_count=10",
			expected: Expected{
				address:             "rabbitmq://gamification-service/event/register",
				isEvent:             true,
				isAuthTokenRequired: false,
				tokenScope:          "",
			},
		},
		{
			commentText: "// bind-method: http://kyc-service/verification-files/get?auth_token=required",
			expected: Expected{
				address:             "http://kyc-service/verification-files/get",
				isEvent:             false,
				isAuthTokenRequired: true,
				tokenScope:          "",
			},
		},
		{
			commentText: "// bind-method: http://kyc-service/verification-files/get?auth_token=boa",
			expected: Expected{
				address:             "http://kyc-service/verification-files/get",
				isEvent:             false,
				isAuthTokenRequired: true,
				tokenScope:          "boa",
			},
		},
	}

	for _, test := range tests {
		comment := &ast.Comment{
			Text: test.commentText,
		}

		address, isEvent, isAuthTokenRequired, tokenScore := findBindAddress(comment)

		if address != test.expected.address || isEvent != test.expected.isEvent || isAuthTokenRequired != test.expected.isAuthTokenRequired || tokenScore != test.expected.tokenScope {
			t.Errorf("findBindAddress(%q) = %q, %t, %t, '%s'; want %q, %t, %t, '%s'",
				test.commentText, address, isEvent, isAuthTokenRequired, tokenScore,
				test.expected.address, test.expected.isEvent, test.expected.isAuthTokenRequired, test.expected.tokenScope)
		}
	}
}

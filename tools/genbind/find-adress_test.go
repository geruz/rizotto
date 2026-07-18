package main

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

type findBindAddressExpected struct {
	address string
	isEvent bool
}

func TestFindBindAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		commentText string
		expected    findBindAddressExpected
	}{
		{
			commentText: "// bind-method: http://service/action/get",
			expected: findBindAddressExpected{
				address: "http://service/action/get",
				isEvent: false,
			},
		},
		{
			commentText: "// bind-event: rabbitmq://service/event/register?worker_count=10",
			expected: findBindAddressExpected{
				address: "rabbitmq://service/event/register",
				isEvent: true,
			},
		},
	}

	for _, test := range tests {
		comment := &ast.Comment{Slash: token.NoPos, Text: test.commentText}

		address, isEvent := findBindAddress(comment)

		actual := findBindAddressExpected{
			address: address,
			isEvent: isEvent,
		}
		assert.Equal(t, test.expected, actual, test.commentText)
	}
}

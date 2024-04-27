package user

import (
	"context"

	"github.com/geruz/rizotto/bb"
)

type UserClientMock struct{}

func (c UserClientMock) GetUserRPC(ctx context.Context, req GetUserRequest) (User, bb.ServiceError) {
	panic("not implemented1")
}

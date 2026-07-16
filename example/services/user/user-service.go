package usersrv

import (
	"context"

	"github.com/geruz/rizotto/bb"
	"github.com/geruz/rizotto/example/services/user/user-client"
)

type UserService struct{}

func NewUserService() UserService {
	return UserService{}
}

func (srv UserService) GetUserRPC(ctx context.Context, req user.GetUserRequest) (user.User, bb.ServiceError) {
	return user.User{
		ID:   req.UserID,
		Name: "John",
	}, nil
}
func (srv UserService) SelectUsersRPC(
	ctx context.Context,
	req user.SelectUserRequest,
) (user.UserList, bb.ServiceError) {
	return user.UserList{
		Offset: req.Offset,
		Limit:  req.Limit,
		Users:  []user.User{{ID: 1, Name: "John"}},
	}, nil
}

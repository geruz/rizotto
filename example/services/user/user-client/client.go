package user

import (
	"context"

	"github.com/geruz/rizotto/bb"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GetUserRequest struct {
	UserID int
}

type SelectUserRequest struct {
	Offset int
	Limit  int
}
type UserList struct {
	Offset int
	Limit  int

	Users []User
}
type (
	GetUserRPC          func(ctx context.Context, req GetUserRequest) (User, bb.ServiceError)
	SelectUsersRPC      func(ctx context.Context, req SelectUserRequest) (UserList, bb.ServiceError)
	UserServiceContract interface {
		GetUserRPC(ctx context.Context, req GetUserRequest) (User, bb.ServiceError)
		SelectUsersRPC(ctx context.Context, req SelectUserRequest) (UserList, bb.ServiceError)
	}
)

type userClient struct {
	_getUserRPC     GetUserRPC
	_selectUsersRPC SelectUsersRPC
}

func (c userClient) GetUserRPC(ctx context.Context, req GetUserRequest) (User, bb.ServiceError) {
	return c._getUserRPC(ctx, req)
}

func (c userClient) SelectUsersRPC(ctx context.Context, req SelectUserRequest) (UserList, bb.ServiceError) {
	return c._selectUsersRPC(ctx, req)
}

func MustBind() UserServiceContract {
	return userClient{
		_getUserRPC: func(ctx context.Context, req GetUserRequest) (User, bb.ServiceError) {
			return User{
				ID:   req.UserID,
				Name: "John",
			}, nil
		},
		_selectUsersRPC: func(ctx context.Context, req SelectUserRequest) (UserList, bb.ServiceError) {
			return UserList{
				Offset: req.Offset,
				Limit:  req.Limit,
				Users:  []User{{ID: 1, Name: "John"}},
			}, nil
		},
	}
}

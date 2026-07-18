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

//go:generate genbind --client=UserService
type UserService interface {
	// bind-method: http://user-service/user/get
	GetUserRPC(ctx context.Context, req GetUserRequest) (User, bb.ServiceError)

	// bind-method: http://user-service/user/select
	SelectUsersRPC(ctx context.Context, req SelectUserRequest) (UserList, bb.ServiceError)
}

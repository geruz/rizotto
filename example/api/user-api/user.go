package api

import (
	"github.com/geruz/rizotto"
	"github.com/geruz/rizotto/example/api"
	"github.com/geruz/rizotto/example/services/user/user-client"
	"github.com/samber/lo"
)

type (
	UserController struct {
		rizotto.Controller

		getUserRPC     user.GetUserRPC
		selectUsersRPC user.SelectUsersRPC
	}
)

func NewUserController() UserController {
	userClient := user.MustBind()
	ctrl := UserController{
		Controller:     rizotto.Controller{},
		getUserRPC:     userClient.GetUserRPC,
		selectUsersRPC: userClient.SelectUsersRPC,
	}
	ctrl.AddRoutes(
		rizotto.JSONMethod("GET /api/v1/user/{user_id}", api.PrivateArea, ctrl.getUser),
		rizotto.JSONMethod("GET /api/v1/users", api.PrivateArea, ctrl.UserList),
	)

	return ctrl
}

// Errors.
var (
	userNotFound = rizotto.NewNotFoundError(
		rizotto.ErrorCode("user_not_found"), "User not found", "User with id %s not found",
	)
	internalError = rizotto.NewInternalError()
)

type UserRequest struct {
	UserID int `in:"path=user_id"`
}
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type UserListRequest struct {
	Offset int `in:"query=offset;default=0"`
	Limit  int `in:"query=limit;default=10"`
}

type UserList struct {
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
	Users  []User `json:"users"`
}

func (ctrl UserController) UserList(ctx api.UserHTTPContext, req UserListRequest) (UserList, rizotto.HTTPError) {
	users, err := ctrl.selectUsersRPC(ctx, user.SelectUserRequest{
		Offset: req.Offset,
		Limit:  req.Limit,
	})
	if err != nil {
		return UserList{Offset: 0, Limit: 0, Users: nil}, rizotto.MapError(err).
			Others(internalError)
	}

	return UserList{
		Offset: users.Offset,
		Limit:  users.Limit,
		Users:  lo.Map(users.Users, ServiceUserToAPIUser),
	}, nil
}

func (ctrl UserController) getUser(ctx api.UserHTTPContext, req UserRequest) (User, rizotto.HTTPError) {
	user, err := ctrl.getUserRPC(ctx, user.GetUserRequest{UserID: req.UserID})
	if err != nil {
		return User{ID: 0, Name: ""}, rizotto.MapError(err).
			IfNotFound(userNotFound).
			Others(internalError)
	}

	return ServiceUserToAPIUser(user, 0), nil
}

func ServiceUserToAPIUser(user user.User, _ int) User {
	return User{
		ID:   user.ID,
		Name: user.Name,
	}
}

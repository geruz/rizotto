package api

import (
	"errors"
	"testing"

	"github.com/geruz/rizotto/bb"
	"github.com/geruz/rizotto/example/api/doc"
	"github.com/geruz/rizotto/example/services/user/user-client"

	"github.com/geruz/rizotto/documentation/openapi"
	"github.com/geruz/rizotto/testutils/api"
	"github.com/geruz/rizotto/testutils/mock"
)

const testUserName = "John"

var errInternalExample = errors.New("test error")

func Test_GetUserRoute(t *testing.T) {
	t.Parallel()

	api.GroupRouteTests(t, "GET /api/v1/user/{user_id}", NewUserController).
		Documentation(
			doc.ApiPublicDocumentationV1,
			openapi.Description("Returns a `User` object with the given `id`."),
		).
		Case(
			"existed user",
			func(t *testing.T, ctr UserController, cfg api.RequestConfiguration) {
				t.Helper()
				// arrange
				mock.RPC(&ctr.getUserRPC).Success(user.User{ID: 1, Name: testUserName})

				// act
				api.TestAPICall(t, ctr, ctr.getUser, cfg.WithPathVar("user_id", "1")).
					AddInDocumentation(t, "OK").
					ExpectedRequest(t, UserRequest{UserID: 1}).
					ExpectedResponse(t, User{ID: 1, Name: testUserName})
			},
		).
		Case(
			"user not found",
			func(t *testing.T, ctr UserController, cfg api.RequestConfiguration) {
				t.Helper()
				// arrange
				mock.RPC(&ctr.getUserRPC).Error(bb.NewNotFoundError("User with id not found"))

				// act
				api.TestAPICall(t, ctr, ctr.getUser, cfg.WithPathVar("user_id", "2")).
					AddInDocumentation(t, "A user not found").
					ExpectedRequest(t, UserRequest{UserID: 2}).
					ExpectedError(t, userNotFound)
			},
		).
		Case(
			"internal error",
			func(t *testing.T, ctr UserController, cfg api.RequestConfiguration) {
				t.Helper()
				// arrange
				mock.RPC(&ctr.getUserRPC).Error(bb.NewInternalError("Internal error", errInternalExample))

				// act
				api.TestAPICall(t, ctr, ctr.getUser, cfg.WithPathVar("user_id", "2")).
					AddInDocumentation(t, "An internal error").
					ExpectedRequest(t, UserRequest{UserID: 2}).
					ExpectedError(t, internalError)
			},
		)
}

func Test_UserListRoute(t *testing.T) {
	t.Parallel()

	api.GroupRouteTests(t, "GET /api/v1/users", NewUserController).
		Documentation(
			doc.ApiPublicDocumentationV1,
			openapi.Description("Returns a `User` object with the given `id`."),
		).
		Case("user list", testUserListCase).
		Case("user list2", testUserListDefaultValuesCase)
}

func testUserListCase(t *testing.T, ctr UserController, cfg api.RequestConfiguration) {
	t.Helper()
	// arrange
	mock.RPC(&ctr.selectUsersRPC).Success(user.UserList{
		Offset: 2,
		Limit:  10,
		Users: []user.User{{
			ID:   1,
			Name: testUserName,
		}},
	})

	// act
	api.TestAPICall(t, ctr, ctr.UserList, cfg.WithQuery("limit=10&offset=2")).
		AddInDocumentation(t, "OK").
		ExpectedRequest(t, UserListRequest{
			Limit:  10,
			Offset: 2,
		}).
		ExpectedResponse(t, UserList{
			Limit:  10,
			Offset: 2,
			Users: []User{{
				ID:   1,
				Name: testUserName,
			}},
		})
}

func testUserListDefaultValuesCase(t *testing.T, ctr UserController, cfg api.RequestConfiguration) {
	t.Helper()

	// arrange
	mock.RPC(&ctr.selectUsersRPC).Success(user.UserList{
		Offset: 0,
		Limit:  10,
		Users: []user.User{{
			ID:   1,
			Name: testUserName,
		}},
	})

	// act
	api.TestAPICall(t, ctr, ctr.UserList, cfg.WithQuery("")).
		AddInDocumentation(t, "DefaultValues").
		ExpectedRequest(t, UserListRequest{
			Limit:  10,
			Offset: 0,
		}).
		ExpectedResponse(t, UserList{
			Limit:  10,
			Offset: 0,
			Users: []User{{
				ID:   1,
				Name: testUserName,
			}},
		})
}

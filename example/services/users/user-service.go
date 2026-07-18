package usersrv

import (
	"context"
	"math"

	"github.com/geruz/rizotto/bb"
	"github.com/geruz/rizotto/example/services/users/repository"
	"github.com/geruz/rizotto/example/services/users/repository/db"
	"github.com/geruz/rizotto/example/services/users/user-client"
	"github.com/geruz/rizotto/pg"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(pool *pgxpool.Pool) UserService {
	return UserService{repo: repository.NewUserRepository(pool)}
}

func (srv UserService) GetUserRPC(ctx context.Context, req user.GetUserRequest) (user.User, bb.ServiceError) {
	if req.UserID < 0 || req.UserID > math.MaxInt32 {
		return user.User{}, bb.NewInvalidRequestError("invalid user id")
	}

	dbUser, err := srv.repo.GetUserByID(ctx, int32(req.UserID))
	if err != nil {
		if pg.IsNotFoundError(err) {
			return user.User{}, bb.NewNotFoundError("user not found")
		}

		return user.User{}, bb.NewInternalError("failed to get user", err)
	}

	return toUser(dbUser), nil
}

func (srv UserService) SelectUsersRPC(
	ctx context.Context,
	req user.SelectUserRequest,
) (user.UserList, bb.ServiceError) {
	dbUsers, err := srv.repo.GetAllUsers(ctx)
	if err != nil {
		return user.UserList{}, bb.NewInternalError("failed to get users", err)
	}

	users := make([]user.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = toUser(dbUser)
	}

	return user.UserList{
		Offset: req.Offset,
		Limit:  req.Limit,
		Users:  users,
	}, nil
}

func toUser(dbUser db.User) user.User {
	return user.User{
		ID:   int(dbUser.ID),
		Name: dbUser.Name,
	}
}

package repository

import (
	"context"

	"github.com/geruz/rizotto/example/services/users/repository/db"
	"github.com/geruz/rizotto/pg"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pg.Repository

	queries *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return UserRepository{
		Repository: pg.NewRepository(pool),
		queries:    db.New(pool),
	}
}

func (r UserRepository) GetAllUsers(ctx context.Context) ([]db.User, error) {
	return r.queries.GetAllUsers(ctx)
}

func (r UserRepository) GetUserByID(ctx context.Context, id int64) (db.User, error) {
	return r.queries.GetUser(ctx, id)
}

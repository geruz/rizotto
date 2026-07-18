package pg

import "github.com/jackc/pgx/v5/pgxpool"

type Repository struct {
	conn *pgxpool.Pool
}

func NewRepository(conn *pgxpool.Pool) Repository {
	return Repository{conn: conn}
}

func (r Repository) Conn() *pgxpool.Pool {
	return r.conn
}

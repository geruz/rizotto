package sql

import (
	"context"
	"errors"

	"github.com/geruz/rizotto/err"
	"github.com/geruz/rizotto/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTransactionAlreadyExists = errors.New("transaction already exists")
	ErrTransactionDoesNotExist  = errors.New("transaction does not exist")
	ErrNoSavePoint              = errors.New("no save point")
)

type PoolInterface interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Transaction struct {
	conn      *pgxpool.Pool
	tx        pgx.Tx
	err       error
	pointName string
}

func NewTransaction(ctx context.Context, conn *pgxpool.Pool) (*Transaction, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		conn:      conn,
		tx:        tx,
		err:       nil,
		pointName: "",
	}, nil
}

func (t Transaction) Tx() pgx.Tx {
	return t.tx
}

func (t Transaction) PointName() string {
	return t.pointName
}

func (t Transaction) Error() error {
	return t.err
}

func (t *Transaction) Commit(ctx context.Context) error {
	if t.tx == nil {
		logger.Trace(ctx, "Ignoring Commit because transaction is nil")

		return nil
	}

	if t.err = t.tx.Commit(ctx); t.err != nil {
		return err.Wrap("Error while committing transaction", t.err)
	}
	t.tx = nil
	t.pointName = ""

	return nil
}

func (t *Transaction) Rollback(ctx context.Context) error {
	if t.tx == nil {
		logger.Trace(ctx, "Ignoring Rollback because transaction is nil")

		return nil
	}
	if t.err = t.tx.Rollback(ctx); t.err != nil {
		return err.Wrap("Error while rolling back transaction", t.err)
	}
	t.tx = nil
	t.pointName = ""

	return nil
}

func (t *Transaction) Close(ctx context.Context) error {
	if t.tx == nil {
		logger.Trace(ctx, "Ignoring Close because transaction is nil")

		return nil
	}
	if t.pointName == "" {
		logger.Trace(ctx, "PointName is empty, calling t.Rollback(ctx)")
		if t.err = t.Rollback(ctx); t.err != nil {
			return err.Wrap("Error while t.Rollback", t.err)
		}

		return nil
	}

	logger.Trace(ctx, " PointName is not empty, calling t.RollbackToSavePoint(ctx)")
	if t.err = t.RollbackToSavePoint(ctx); t.err != nil {
		return err.Wrap("Error while t.RollbackToSavePoint", t.err)
	}
	logger.Trace(ctx, "Calling t.Commit(ctx)")
	if t.err = t.Commit(ctx); t.err != nil {
		return err.Wrap("Error while t.Commit", t.err)
	}
	logger.Trace(ctx, "Transaction End")

	return nil
}

func (t *Transaction) RollbackToSavePoint(ctx context.Context) error {
	if t.tx == nil {
		return ErrTransactionDoesNotExist
	}
	if t.pointName == "" {
		return ErrNoSavePoint
	}

	if _, t.err = t.tx.Exec(ctx, "ROLLBACK TO "+t.pointName); t.err != nil {
		return err.Wrap("Error while ROLLBACK TO to save point", t.err)
	}

	return t.Commit(ctx)
}

func (t *Transaction) SavePoint(ctx context.Context) error {
	if t.tx == nil {
		return ErrTransactionDoesNotExist
	}
	pointName := "somePoint"
	if _, t.err = t.tx.Exec(ctx, "SAVEPOINT "+pointName); t.err != nil {
		return err.Wrap("Error while SAVEPOINT", t.err)
	}
	t.pointName = pointName

	return nil
}

func (t *Transaction) Begin(ctx context.Context) error {
	if t.tx != nil {
		return ErrTransactionAlreadyExists
	}
	t.tx, t.err = t.conn.Begin(ctx)
	if t.err != nil {
		return err.Wrap("Error while t.conn.Begin", t.err)
	}
	t.pointName = ""

	return nil
}

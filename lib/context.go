package lib

import (
	"context"
	sql "database/sql"
	"errors"
)

type dbContextKeyType struct{}

var DbContextKey = dbContextKeyType{}

func ContextWithDB(
	ctx context.Context,
	db *sql.DB,
) context.Context {
	return context.WithValue(ctx, DbContextKey, db)
}

func DBFromContext(
	ctx context.Context,
) (*sql.DB, error) {

	db := ctx.Value(DbContextKey)

	if db == nil {
		return nil, errors.New("no DB on context")
	}

	sqlDB, ok := db.(*sql.DB)
	if !ok {
		return nil, errors.New("DB context value is not an sql.DB type")
	}

	return sqlDB, nil
}

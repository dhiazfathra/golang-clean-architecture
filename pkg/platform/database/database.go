package database

import (
	"fmt"

	_ "github.com/lib/pq"

	"github.com/jmoiron/sqlx"
)

func MustConnect(dsn string) *sqlx.DB {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		panic(fmt.Sprintf("database: connect: %v", err))
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db
}

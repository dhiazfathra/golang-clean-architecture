package database

import (
	"fmt"

	sqltrace "github.com/DataDog/dd-trace-go/contrib/database/sql/v2"
	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func init() {
	// Register traced postgres driver under a new name.
	// sqltrace.Register wraps the driver transparently.
	sqltrace.Register("postgres-traced", &pq.Driver{},
		sqltrace.WithService("golang-clean-arch-db"),
		sqltrace.WithDBMPropagation(ddtracer.DBMPropagationModeFull),
	)
}

// MustConnect connects using the traced postgres driver.
// Signature unchanged — callers need no modification.
func MustConnect(dsn string) *sqlx.DB {
	db, err := sqlx.Open("postgres-traced", dsn)
	if err != nil {
		panic(fmt.Sprintf("database: open: %v", err))
	}
	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("database: ping: %v", err))
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db
}

package database

import (
	"fmt"
	"sync"

	sqltrace "github.com/DataDog/dd-trace-go/contrib/database/sql/v2"
	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// PoolConfig holds connection pool and tracing settings for the database.
type PoolConfig struct {
	MaxOpenConns int
	MaxIdleConns int
	ServiceName  string // Datadog APM service name (e.g. "golang-clean-arch-db")
}

var registerOnce sync.Once

// MustConnect connects using the traced postgres driver.
func MustConnect(dsn string, pool PoolConfig) *sqlx.DB {
	registerOnce.Do(func() {
		sqltrace.Register("postgres-traced", &pq.Driver{},
			sqltrace.WithService(pool.ServiceName),
			sqltrace.WithDBMPropagation(ddtracer.DBMPropagationModeFull),
		)
	})

	db, err := sqlx.Open("postgres-traced", dsn)
	if err != nil {
		panic(fmt.Sprintf("database: open: %v", err))
	}
	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("database: ping: %v", err))
	}
	db.SetMaxOpenConns(pool.MaxOpenConns)
	db.SetMaxIdleConns(pool.MaxIdleConns)
	return db
}

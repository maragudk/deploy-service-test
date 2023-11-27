package sql

import (
	"context"
	"database/sql"
	"embed"
	"io"
	"io/fs"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/maragudk/errors"
	"github.com/maragudk/migrate"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Database struct {
	DB                    *sqlx.DB
	url                   string
	maxOpenConnections    int
	maxIdleConnections    int
	connectionMaxLifetime time.Duration
	connectionMaxIdleTime time.Duration
	log                   *log.Logger
	metrics               *prometheus.Registry
}

type NewDatabaseOptions struct {
	URL                   string
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
	ConnectionMaxIdleTime time.Duration
	Log                   *log.Logger
	Metrics               *prometheus.Registry
}

// NewDatabase with the given options.
// If no logger is provided, logs are discarded.
func NewDatabase(opts NewDatabaseOptions) *Database {
	if opts.Log == nil {
		opts.Log = log.New(io.Discard, "", 0)
	}

	if opts.Metrics == nil {
		opts.Metrics = prometheus.NewRegistry()
	}

	// - Set WAL mode (not strictly necessary each time because it's persisted in the database, but good for first run)
	// - Set busy timeout, so concurrent writers wait on each other instead of erroring immediately
	// - Enable foreign key checks
	opts.URL += "?_journal=WAL&_timeout=5000&_fk=true"

	return &Database{
		url:                   opts.URL,
		maxOpenConnections:    opts.MaxOpenConnections,
		maxIdleConnections:    opts.MaxIdleConnections,
		connectionMaxLifetime: opts.ConnectionMaxLifetime,
		connectionMaxIdleTime: opts.ConnectionMaxIdleTime,
		log:                   opts.Log,
		metrics:               opts.Metrics,
	}
}

func (d *Database) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	d.log.Println("Connecting to database at", d.url)

	var err error
	d.DB, err = sqlx.ConnectContext(ctx, "sqlite3", d.url)
	if err != nil {
		return err
	}

	d.log.Println("Setting connection pool options (",
		"max open connections:", d.maxOpenConnections,
		", max idle connections:", d.maxIdleConnections,
		", connection max lifetime:", d.connectionMaxLifetime,
		", connection max idle time:", d.connectionMaxIdleTime,
		")")
	d.DB.SetMaxOpenConns(d.maxOpenConnections)
	d.DB.SetMaxIdleConns(d.maxIdleConnections)
	d.DB.SetConnMaxLifetime(d.connectionMaxLifetime)
	d.DB.SetConnMaxIdleTime(d.connectionMaxIdleTime)

	d.metrics.MustRegister(collectors.NewDBStatsCollector(d.DB.DB, "app"))

	return nil
}

// inTransaction runs callback in a transaction, and makes sure to handle rollbacks, commits etc.
func (d *Database) inTransaction(ctx context.Context, callback func(tx *sqlx.Tx) error) (err error) {
	tx, err := d.DB.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return errors.Wrap(err, "error beginning transaction")
	}
	defer func() {
		if rec := recover(); rec != nil {
			err = rollback(tx, errors.Newf("panic: %v", rec))
		}
	}()
	if err := callback(tx); err != nil {
		return rollback(tx, err)
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error committing transaction")
	}

	return nil
}

// rollback a transaction, handling both the original error and any transaction rollback errors.
func rollback(tx *sqlx.Tx, err error) error {
	if txErr := tx.Rollback(); txErr != nil {
		return errors.Wrap(err, "error rolling back transaction after error (transaction error: %v), original error", txErr)
	}
	return err
}

//go:embed migrations
var migrations embed.FS

func (d *Database) MigrateUp(ctx context.Context) error {
	fsys := d.getMigrations()
	return migrate.Up(ctx, d.DB.DB, fsys)
}

func (d *Database) MigrateDown(ctx context.Context) error {
	fsys := d.getMigrations()
	return migrate.Down(ctx, d.DB.DB, fsys)
}

func (d *Database) getMigrations() fs.FS {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		panic(err)
	}
	return fsys
}

func (d *Database) Ping(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

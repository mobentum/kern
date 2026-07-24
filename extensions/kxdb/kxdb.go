// Package kxdb provides first-class xdb integration for kern applications.
//
// It supports multiple named database connections, request-scoped injection
// via context, and per-request transaction middleware.
package kxdb

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/mobentum/kern"
	"github.com/mobentum/xdb"
)

// Config for a single named database connection.
type Config struct {
	Driver          string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Migrations      fs.FS
	MigrationsPath  string
	LogQueries      bool
}

// Databases holds named *xdb.DB instances.
type Databases struct {
	mu     sync.Mutex
	dbs    map[string]*xdb.DB
	logger *slog.Logger
}

// New creates named *xdb.DB instances from a config map.
// Panics if any required connection fails (fail-fast at startup).
func New(configs map[string]Config) *Databases {
	dbs := &Databases{
		dbs:    make(map[string]*xdb.DB, len(configs)),
		logger: slog.Default(),
	}
	for name, cfg := range configs {
		db, err := xdb.New(xdb.DBConfig{
			Driver:          cfg.Driver,
			DSN:             cfg.DSN,
			MaxOpenConns:    cfg.MaxOpenConns,
			MaxIdleConns:    cfg.MaxIdleConns,
			ConnMaxLifetime: cfg.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.ConnMaxIdleTime,
			Logger:          dbs.logger,
			LogQueries:      cfg.LogQueries,
		})
		if err != nil {
			panic(fmt.Sprintf("kxdb: %s: %v", name, err))
		}
		if cfg.Migrations != nil {
			if err := db.MigrateUp(cfg.Migrations, cfg.MigrationsPath); err != nil {
				dbs.logger.Warn("kxdb: migration failed",
					slog.String("db", name),
					slog.String("error", err.Error()),
				)
			}
		}
		dbs.dbs[name] = db
	}
	return dbs
}

// Add creates an additional named database connection at runtime.
func (dbs *Databases) Add(name string, cfg Config) error {
	db, err := xdb.New(xdb.DBConfig{
		Driver:          cfg.Driver,
		DSN:             cfg.DSN,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		Logger:          dbs.logger,
		LogQueries:      cfg.LogQueries,
	})
	if err != nil {
		return fmt.Errorf("kxdb: %s: %w", name, err)
	}
	if cfg.Migrations != nil {
		if err := db.MigrateUp(cfg.Migrations, cfg.MigrationsPath); err != nil {
			dbs.logger.Warn("kxdb: migration failed",
				slog.String("db", name),
				slog.String("error", err.Error()),
			)
		}
	}
	dbs.mu.Lock()
	dbs.dbs[name] = db
	dbs.mu.Unlock()
	return nil
}

// Get returns a named *xdb.DB. Returns nil if not found.
func (dbs *Databases) Get(name string) *xdb.DB {
	dbs.mu.Lock()
	defer dbs.mu.Unlock()
	return dbs.dbs[name]
}

// Close closes all databases. Returns the first error encountered.
func (dbs *Databases) Close() error {
	dbs.mu.Lock()
	defer dbs.mu.Unlock()
	var firstErr error
	for name, db := range dbs.dbs {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("kxdb: %s: %w", name, err)
		}
	}
	return firstErr
}

// ── Context keys ──────────────────────────────────────────

type ctxKeyDatabases struct{}
type ctxKeyTx struct{ dbName string }

// Middleware injects Databases into the request context.
// Use CtxDB to retrieve a named database from within a handler.
func Middleware(dbs *Databases) kern.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxKeyDatabases{}, dbs)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CtxDB retrieves a named *xdb.DB from the context.
// Returns nil if not found. When inside a MiddlewareWithTx scope,
// returns the transaction-scoped DB for the given name.
func CtxDB(ctx context.Context, name string) *xdb.DB {
	if txDB, ok := ctx.Value(ctxKeyTx{dbName: name}).(*xdb.DB); ok && txDB != nil {
		return txDB
	}
	dbs, ok := ctx.Value(ctxKeyDatabases{}).(*Databases)
	if !ok || dbs == nil {
		return nil
	}
	return dbs.Get(name)
}

// DefaultDB is a shortcut for CtxDB(ctx, "default").
func DefaultDB(ctx context.Context) *xdb.DB {
	return CtxDB(ctx, "default")
}

// MiddlewareWithTx wraps the specified named database in a transaction
// for the duration of the request. Commits on success, rolls back on
// error or panic. The transactional DB replaces the named connection
// for the scope of that request via CtxDB.
func MiddlewareWithTx(dbName string, dbs *Databases) kern.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			base := dbs.Get(dbName)
			if base == nil {
				panic(fmt.Sprintf("kxdb: MiddlewareWithTx: database %q not found", dbName))
			}

			tx, err := base.Underlying().BeginTxx(r.Context(), nil)
			if err != nil {
				panic(fmt.Sprintf("kxdb: begin tx %s: %v", dbName, err))
			}

			txDB := base.WithTx(tx)
			ctx := context.WithValue(r.Context(), ctxKeyTx{dbName: dbName}, txDB)

			panicked := true
			defer func() {
				if panicked {
					tx.Rollback()
				}
			}()

			next.ServeHTTP(w, r.WithContext(ctx))

			panicked = false
			if commitErr := tx.Commit(); commitErr != nil {
				panic(fmt.Sprintf("kxdb: commit tx %s: %v", dbName, commitErr))
			}
		})
	}
}

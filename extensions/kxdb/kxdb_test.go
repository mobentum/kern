package kxdb

import (
	"context"
	"testing"

	"github.com/mobentum/kern"
	"github.com/mobentum/xdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func TestNew_SingleDB(t *testing.T) {
	dbs := New(map[string]Config{
		"default": {Driver: "sqlite3", DSN: ":memory:"},
	})
	defer dbs.Close()

	db := dbs.Get("default")
	require.NotNil(t, db)
}

func TestNew_MultiDB(t *testing.T) {
	dbs := New(map[string]Config{
		"primary": {Driver: "sqlite3", DSN: ":memory:"},
		"cache":   {Driver: "sqlite3", DSN: ":memory:"},
	})
	defer dbs.Close()

	assert.NotNil(t, dbs.Get("primary"))
	assert.NotNil(t, dbs.Get("cache"))
}

func TestGet_Missing(t *testing.T) {
	dbs := New(map[string]Config{
		"default": {Driver: "sqlite3", DSN: ":memory:"},
	})
	defer dbs.Close()

	assert.Nil(t, dbs.Get("nonexistent"))
}

func TestAdd_Runtime(t *testing.T) {
	dbs := New(map[string]Config{})
	defer dbs.Close()

	err := dbs.Add("runtime", Config{Driver: "sqlite3", DSN: ":memory:"})
	require.NoError(t, err)
	assert.NotNil(t, dbs.Get("runtime"))
}

func TestDBFromContext_Middleware(t *testing.T) {
	dbs := New(map[string]Config{
		"default": {Driver: "sqlite3", DSN: ":memory:"},
	})
	defer dbs.Close()

	app := kern.Default()
	app.Use(Middleware(dbs))

	var resultDB *xdb.DB
	app.GET("/test", func(c *kern.Context) {
		resultDB = DBFromContext(c.Context(), "default")
		c.NoContent(200)
	})

	client := kern.NewTestClient(app)
	resp := client.Get("/test")
	assert.Equal(t, 200, resp.Code)
	assert.NotNil(t, resultDB)
}

func TestDefaultDB(t *testing.T) {
	dbs := New(map[string]Config{
		"default": {Driver: "sqlite3", DSN: ":memory:"},
	})
	defer dbs.Close()

	app := kern.Default()
	app.Use(Middleware(dbs))

	var resultDB *xdb.DB
	app.GET("/test", func(c *kern.Context) {
		resultDB = DefaultDBFromContext(c.Context())
		c.NoContent(200)
	})

	client := kern.NewTestClient(app)
	client.Get("/test")
	assert.NotNil(t, resultDB)
}

func TestDBFromContext_NoMiddleware(t *testing.T) {
	db := DBFromContext(context.Background(), "default")
	assert.Nil(t, db)
}

func TestMiddlewareWithTx(t *testing.T) {
	dbs := New(map[string]Config{
		"default": {Driver: "sqlite3", DSN: fileDB(t)},
	})
	defer dbs.Close()

	defaultDB := dbs.Get("default")
	_, err := defaultDB.RawExec(context.Background(),
		"CREATE TABLE test_tx (id INTEGER PRIMARY KEY, val TEXT)")
	require.NoError(t, err)

	app := kern.Default()
	app.Use(Middleware(dbs))

	txGroup := app.Group("/tx", MiddlewareWithTx("default", dbs))

	txGroup.POST("", func(c *kern.Context) {
		db := DBFromContext(c.Context(), "default")
		_, err := db.Insert("test_tx").Columns("val").Values("committed").Exec(c.Context())
		if err != nil {
			c.Error(500, err.Error())
			return
		}
		c.NoContent(201)
	})

	client := kern.NewTestClient(app)
	resp := client.PostJSON("/tx", map[string]string{"x": "y"})
	require.Equal(t, 201, resp.Code)

	var val string
	err = defaultDB.RawOne(context.Background(), &val, "SELECT val FROM test_tx WHERE id = 1")
	require.NoError(t, err)
	assert.Equal(t, "committed", val)
}

func fileDB(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/test.db"
}

package xconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadAndOverload(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	if err := os.WriteFile(file, []byte("APP_NAME=from-file\nAPP_PORT=8080\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("APP_NAME", "from-env")
	if err := Load(file); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := os.Getenv("APP_NAME"); got != "from-env" {
		t.Fatalf("Load overwrote existing value: %q", got)
	}

	if err := Overload(file); err != nil {
		t.Fatalf("overload: %v", err)
	}
	if got := os.Getenv("APP_NAME"); got != "from-file" {
		t.Fatalf("Overload did not replace value: %q", got)
	}
	if got := os.Getenv("APP_PORT"); got != "8080" {
		t.Fatalf("expected APP_PORT from file, got %q", got)
	}
}

func TestReadAndParse(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	if err := os.WriteFile(file, []byte("A=1\nB=two\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	values, err := Read(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if values["A"] != "1" || values["B"] != "two" {
		t.Fatalf("unexpected values: %+v", values)
	}

	parsed, err := Parse(strings.NewReader("C=3\nD=four\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed["C"] != "3" || parsed["D"] != "four" {
		t.Fatalf("unexpected parsed values: %+v", parsed)
	}
}

func TestLoaderTypedGetters(t *testing.T) {
	lookup := func(key string) (string, bool) {
		values := map[string]string{
			"APP_HOST":            "127.0.0.1",
			"APP_PORT":            "8080",
			"APP_DEBUG":           "true",
			"APP_ALLOWED_ORIGINS": "https://a.test, https://b.test",
			"APP_TIMEOUT":         "5s",
		}
		value, ok := values[key]
		return value, ok
	}

	loader, err := New(WithPrefix("APP"), WithLookup(lookup))
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	if got, ok := loader.LookupString("HOST"); !ok || got != "127.0.0.1" {
		t.Fatalf("unexpected lookup host: %q ok=%t", got, ok)
	}
	if got := loader.String("HOST", "localhost"); got != "127.0.0.1" {
		t.Fatalf("unexpected host: %q", got)
	}
	port, err := loader.Int("PORT", 3000)
	if err != nil || port != 8080 {
		t.Fatalf("unexpected port: %d err=%v", port, err)
	}
	debug, err := loader.Bool("DEBUG", false)
	if err != nil || !debug {
		t.Fatalf("unexpected debug=%t err=%v", debug, err)
	}
	timeout, err := loader.Duration("TIMEOUT", time.Second)
	if err != nil || timeout != 5*time.Second {
		t.Fatalf("unexpected timeout: %v err=%v", timeout, err)
	}
	origins := loader.Strings("ALLOWED_ORIGINS", nil)
	if len(origins) != 2 || origins[0] != "https://a.test" || origins[1] != "https://b.test" {
		t.Fatalf("unexpected origins: %+v", origins)
	}
}

func TestInvalidValueErrors(t *testing.T) {
	loader, err := New(WithLookup(func(key string) (string, bool) {
		if key == "APP_PORT" {
			return "nope", true
		}
		return "", false
	}), WithPrefix("APP"))
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	port, parseErr := loader.Int("PORT", 8080)
	if port != 8080 {
		t.Fatalf("expected default port, got %d", port)
	}
	if parseErr == nil {
		t.Fatal("expected invalid value error")
	}
	if !errors.Is(parseErr, ErrInvalidValue) {
		t.Fatalf("expected ErrInvalidValue, got %v", parseErr)
	}
}

func TestInvalidDurationErrors(t *testing.T) {
	loader, err := New(WithLookup(func(key string) (string, bool) {
		if key == "APP_TIMEOUT" {
			return "later", true
		}
		return "", false
	}), WithPrefix("APP"))
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	timeout, parseErr := loader.Duration("TIMEOUT", time.Second)
	if timeout != time.Second {
		t.Fatalf("expected default timeout, got %v", timeout)
	}
	if parseErr == nil {
		t.Fatal("expected invalid duration error")
	}
	if !errors.Is(parseErr, ErrInvalidValue) {
		t.Fatalf("expected ErrInvalidValue, got %v", parseErr)
	}
}

func TestRequiredValues(t *testing.T) {
	loader, err := New(WithLookup(func(key string) (string, bool) {
		values := map[string]string{
			"SERVICE_NAME": "billing",
		}
		value, ok := values[key]
		return value, ok
	}))
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	name, err := loader.RequiredString("SERVICE_NAME")
	if err != nil || name != "billing" {
		t.Fatalf("unexpected required string value=%q err=%v", name, err)
	}

	_, err = loader.RequiredString("DATABASE_URL")
	if err == nil {
		t.Fatal("expected required error")
	}
	if !errors.Is(err, ErrMissingRequired) {
		t.Fatalf("expected ErrMissingRequired, got %v", err)
	}
}

func TestWithDotEnv(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	if err := os.WriteFile(file, []byte("SERVICE_NAME=orders\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	loader, err := New(WithDotEnv(file))
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}
	if got := loader.String("SERVICE_NAME", ""); got != "orders" {
		t.Fatalf("unexpected dotenv value: %q", got)
	}
}

func TestWithDotEnvError(t *testing.T) {
	_, err := New(WithDotEnv("missing.env"))
	if err == nil {
		t.Fatal("expected dotenv load error")
	}
}

func TestMustLoadPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustLoad("missing.env")
}

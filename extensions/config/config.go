package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var (
	ErrMissingRequired = errors.New("missing required config value")
	ErrInvalidValue    = errors.New("invalid config value")
)

type lookupFn func(string) (string, bool)

// Error represents a configuration lookup or parsing error.
type Error struct {
	Key   string
	Value string
	Err   error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Value == "" {
		return fmt.Sprintf("%s: %v", e.Key, e.Err)
	}
	return fmt.Sprintf("%s: invalid value %q", e.Key, e.Value)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Loader reads typed configuration values from the environment.
type Loader struct {
	prefix string
	lookup lookupFn
}

// Option configures a Loader.
type Option func(*Loader) error

// Load reads environment variables from .env files without overriding existing values.
func Load(files ...string) error {
	if len(files) == 0 {
		return godotenv.Load()
	}
	return godotenv.Load(files...)
}

// MustLoad loads .env files and panics on error.
func MustLoad(files ...string) {
	if err := Load(files...); err != nil {
		panic(err)
	}
}

// Overload reads environment variables from .env files and overrides existing values.
func Overload(files ...string) error {
	if len(files) == 0 {
		return godotenv.Overload()
	}
	return godotenv.Overload(files...)
}

// Read parses .env files into a map without mutating the process environment.
func Read(files ...string) (map[string]string, error) {
	if len(files) == 0 {
		files = []string{".env"}
	}
	return godotenv.Read(files...)
}

// Parse reads environment variables from a reader.
func Parse(r io.Reader) (map[string]string, error) {
	return godotenv.Parse(r)
}

// WithPrefix prepends a constant prefix to all requested keys.
func WithPrefix(prefix string) Option {
	return func(l *Loader) error {
		p := strings.TrimSpace(prefix)
		if p == "" {
			l.prefix = ""
			return nil
		}
		l.prefix = strings.TrimSuffix(p, "_") + "_"
		return nil
	}
}

// WithDotEnv loads dotenv files when the loader is constructed.
func WithDotEnv(paths ...string) Option {
	return func(_ *Loader) error {
		if len(paths) == 0 {
			return godotenv.Load()
		}
		return godotenv.Load(paths...)
	}
}

// WithLookup overrides environment lookup behavior.
func WithLookup(fn func(string) (string, bool)) Option {
	return func(l *Loader) error {
		if fn != nil {
			l.lookup = fn
		}
		return nil
	}
}

// New constructs a typed environment loader.
func New(opts ...Option) (*Loader, error) {
	l := &Loader{lookup: os.LookupEnv}
	for _, opt := range opts {
		if err := opt(l); err != nil {
			return nil, err
		}
	}
	return l, nil
}

func (l *Loader) key(key string) string {
	if l.prefix == "" {
		return key
	}
	return l.prefix + key
}

func (l *Loader) lookupValue(key string) (string, bool) {
	value, ok := l.lookup(l.key(key))
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func (l *Loader) missing(key string) error {
	return &Error{Key: l.key(key), Err: ErrMissingRequired}
}

func (l *Loader) invalid(key, value string, err error) error {
	return &Error{Key: l.key(key), Value: value, Err: errors.Join(ErrInvalidValue, err)}
}

// LookupString returns a raw string and whether it was present.
func (l *Loader) LookupString(key string) (string, bool) {
	return l.lookupValue(key)
}

// String returns the value for a key or the provided default.
func (l *Loader) String(key, def string) string {
	if value, ok := l.lookupValue(key); ok {
		return value
	}
	return def
}

// RequiredString returns a value or an error when the key is missing.
func (l *Loader) RequiredString(key string) (string, error) {
	if value, ok := l.lookupValue(key); ok {
		return value, nil
	}
	return "", l.missing(key)
}

// Bool returns a parsed bool or the provided default when the key is absent.
func (l *Loader) Bool(key string, def bool) (bool, error) {
	value, ok := l.lookupValue(key)
	if !ok {
		return def, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return def, l.invalid(key, value, err)
	}
	return parsed, nil
}

// Int returns a parsed int or the provided default when the key is absent.
func (l *Loader) Int(key string, def int) (int, error) {
	value, ok := l.lookupValue(key)
	if !ok {
		return def, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return def, l.invalid(key, value, err)
	}
	return parsed, nil
}

// Duration returns a parsed duration or the provided default when the key is absent.
func (l *Loader) Duration(key string, def time.Duration) (time.Duration, error) {
	value, ok := l.lookupValue(key)
	if !ok {
		return def, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return def, l.invalid(key, value, err)
	}
	return parsed, nil
}

// Strings returns a comma-separated list or the provided default.
func (l *Loader) Strings(key string, def []string) []string {
	value, ok := l.lookupValue(key)
	if !ok {
		return def
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

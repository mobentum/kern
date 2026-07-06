package middleware

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/mobentum/kern"
)

var (
	ErrSessionSigningKeyMissing = errors.New("session signing key is required")
	ErrSessionMalformedCookie   = errors.New("session cookie is malformed")
	ErrSessionInvalidSignature  = errors.New("session cookie signature is invalid")
)

type sessionContextKey struct{}

type sessionPayload struct {
	ID        string                   `json:"id"`
	CreatedAt int64                    `json:"created_at"`
	ExpiresAt int64                    `json:"expires_at"`
	Values    map[string]interface{}   `json:"values,omitempty"`
	Flashes   map[string][]interface{} `json:"flashes,omitempty"`
}

// SessionConfig configures cookie-based session middleware.
type SessionConfig struct {
	SigningKey []byte
	VerifyKeys [][]byte

	EncryptionKey []byte
	DecryptKeys   [][]byte

	CookieName string
	Path       string
	Domain     string
	Secure     bool
	HTTPOnly   bool
	SameSite   http.SameSite
	MaxAge     time.Duration

	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
}

// SessionState represents request-scoped mutable session state.
type SessionState struct {
	payload sessionPayload
	dirty   bool
	destroy bool
}

type sessionCodec struct {
	signingKey    []byte
	verifyKeys    [][]byte
	encryptionKey []byte
	decryptKeys   [][]byte
}

// Session applies signed cookie session management.
func SessionMiddleware(configs ...SessionConfig) kern.MiddlewareFunc {
	config := defaultSessionConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if len(provided.SigningKey) > 0 {
			config.SigningKey = provided.SigningKey
		}
		if len(provided.VerifyKeys) > 0 {
			config.VerifyKeys = append([][]byte(nil), provided.VerifyKeys...)
		}
		if len(provided.EncryptionKey) > 0 {
			config.EncryptionKey = provided.EncryptionKey
		}
		if len(provided.DecryptKeys) > 0 {
			config.DecryptKeys = append([][]byte(nil), provided.DecryptKeys...)
		}
		if provided.CookieName != "" {
			config.CookieName = provided.CookieName
		}
		if provided.Path != "" {
			config.Path = provided.Path
		}
		if provided.Domain != "" {
			config.Domain = provided.Domain
		}
		if provided.MaxAge > 0 {
			config.MaxAge = provided.MaxAge
		}
		config.Secure = provided.Secure
		config.HTTPOnly = provided.HTTPOnly
		if provided.SameSite != 0 {
			config.SameSite = provided.SameSite
		}
		if provided.ErrorHandler != nil {
			config.ErrorHandler = provided.ErrorHandler
		}
	}

	if len(config.SigningKey) == 0 {
		panic("middleware.SessionMiddleware: SigningKey must not be empty")
	}
	if len(config.EncryptionKey) > 0 {
		if err := validateAEADKey(config.EncryptionKey); err != nil {
			panic("middleware.SessionMiddleware: " + err.Error())
		}
	}
	for _, key := range config.DecryptKeys {
		if err := validateAEADKey(key); err != nil {
			panic("middleware.SessionMiddleware: " + err.Error())
		}
	}

	codec := buildSessionCodec(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := loadSession(r, config, codec)
			ctx := context.WithValue(r.Context(), sessionContextKey{}, sess)
			r = r.WithContext(ctx)

			sw := &sessionResponseWriter{ResponseWriter: w, request: r, config: config, session: sess, codec: codec}
			next.ServeHTTP(sw, r)

			if !sw.wroteHeader {
				if err := sw.commit(); err != nil && config.ErrorHandler != nil {
					config.ErrorHandler(w, r, err)
					return
				}
			}
		})
	}
}

// Session is an alias to keep API concise (middleware.Session(...)).
func Session(configs ...SessionConfig) kern.MiddlewareFunc {
	return SessionMiddleware(configs...)
}

// GetSession fetches the current request session from context.
func GetSession(ctx context.Context) (*SessionState, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(*SessionState)
	return session, ok && session != nil
}

// ID returns the stable session ID.
func (s *SessionState) ID() string {
	return s.payload.ID
}

// Get returns a session value by key.
func (s *SessionState) Get(key string) (interface{}, bool) {
	if s.payload.Values == nil {
		return nil, false
	}
	v, ok := s.payload.Values[key]
	return v, ok
}

// Set sets a session value by key and marks the session dirty.
func (s *SessionState) Set(key string, value interface{}) {
	if s.payload.Values == nil {
		s.payload.Values = map[string]interface{}{}
	}
	s.payload.Values[key] = value
	s.dirty = true
}

// Delete removes a value by key and marks the session dirty.
func (s *SessionState) Delete(key string) {
	if s.payload.Values == nil {
		return
	}
	delete(s.payload.Values, key)
	s.dirty = true
}

// Clear removes all values and flashes and marks the session dirty.
func (s *SessionState) Clear() {
	s.payload.Values = map[string]interface{}{}
	s.payload.Flashes = map[string][]interface{}{}
	s.dirty = true
}

// Destroy clears server-side state and expires the session cookie.
func (s *SessionState) Destroy() {
	s.destroy = true
	s.dirty = false
}

// RegenerateID rotates the session identifier.
func (s *SessionState) RegenerateID() {
	s.payload.ID = generateSessionID()
	s.dirty = true
}

// AddFlash stores a value for one-time retrieval.
func (s *SessionState) AddFlash(value interface{}, key ...string) {
	flashKey := "default"
	if len(key) > 0 && key[0] != "" {
		flashKey = key[0]
	}
	if s.payload.Flashes == nil {
		s.payload.Flashes = map[string][]interface{}{}
	}
	s.payload.Flashes[flashKey] = append(s.payload.Flashes[flashKey], value)
	s.dirty = true
}

// Flashes returns and clears one-time values.
func (s *SessionState) Flashes(key ...string) []interface{} {
	flashKey := "default"
	if len(key) > 0 && key[0] != "" {
		flashKey = key[0]
	}
	if s.payload.Flashes == nil {
		return nil
	}
	values := s.payload.Flashes[flashKey]
	if len(values) == 0 {
		return nil
	}
	delete(s.payload.Flashes, flashKey)
	s.dirty = true
	return values
}

func defaultSessionConfig() SessionConfig {
	return SessionConfig{
		CookieName: "_session",
		Path:       "/",
		HTTPOnly:   true,
		SameSite:   http.SameSiteLaxMode,
		MaxAge:     24 * time.Hour,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		},
	}
}

func buildSessionCodec(config SessionConfig) sessionCodec {
	verifyKeys := make([][]byte, 0, 1+len(config.VerifyKeys))
	verifyKeys = append(verifyKeys, config.SigningKey)
	for _, key := range config.VerifyKeys {
		if len(key) == 0 {
			continue
		}
		verifyKeys = append(verifyKeys, key)
	}

	decryptKeys := make([][]byte, 0)
	if len(config.EncryptionKey) > 0 {
		decryptKeys = append(decryptKeys, config.EncryptionKey)
	}
	for _, key := range config.DecryptKeys {
		if len(key) == 0 {
			continue
		}
		decryptKeys = append(decryptKeys, key)
	}

	return sessionCodec{
		signingKey:    config.SigningKey,
		verifyKeys:    verifyKeys,
		encryptionKey: config.EncryptionKey,
		decryptKeys:   decryptKeys,
	}
}

func loadSession(r *http.Request, config SessionConfig, codec sessionCodec) *SessionState {
	cookie, err := r.Cookie(config.CookieName)
	if err != nil || cookie.Value == "" {
		return newSession(config.MaxAge)
	}

	payload, err := decodeSessionWithCodec(cookie.Value, codec)
	if err != nil || payload.ExpiresAt <= time.Now().Unix() {
		return newSession(config.MaxAge)
	}

	if payload.Values == nil {
		payload.Values = map[string]interface{}{}
	}
	if payload.Flashes == nil {
		payload.Flashes = map[string][]interface{}{}
	}

	return &SessionState{payload: payload}
}

func newSession(maxAge time.Duration) *SessionState {
	now := time.Now()
	return &SessionState{
		payload: sessionPayload{
			ID:        generateSessionID(),
			CreatedAt: now.Unix(),
			ExpiresAt: now.Add(maxAge).Unix(),
			Values:    map[string]interface{}{},
			Flashes:   map[string][]interface{}{},
		},
	}
}

func generateSessionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func encodeSession(payload sessionPayload, key []byte) (string, error) {
	return encodeSessionWithCodec(payload, sessionCodec{signingKey: key})
}

func encodeSessionWithCodec(payload sessionPayload, codec sessionCodec) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if len(codec.encryptionKey) > 0 {
		body, err = encryptSessionBody(body, codec.encryptionKey)
		if err != nil {
			return "", err
		}
	}

	mac := hmac.New(sha256.New, codec.signingKey)
	_, _ = mac.Write(body)
	sig := mac.Sum(nil)

	bodyPart := base64.RawURLEncoding.EncodeToString(body)
	sigPart := base64.RawURLEncoding.EncodeToString(sig)
	return bodyPart + "." + sigPart, nil
}

func decodeSession(value string, key []byte) (sessionPayload, error) {
	return decodeSessionWithCodec(value, sessionCodec{verifyKeys: [][]byte{key}})
}

func decodeSessionWithCodec(value string, codec sessionCodec) (sessionPayload, error) {
	bodyPart, sigPart, ok := strings.Cut(value, ".")
	if !ok || bodyPart == "" || sigPart == "" {
		return sessionPayload{}, ErrSessionMalformedCookie
	}

	body, err := base64.RawURLEncoding.DecodeString(bodyPart)
	if err != nil {
		return sessionPayload{}, ErrSessionMalformedCookie
	}

	sig, err := base64.RawURLEncoding.DecodeString(sigPart)
	if err != nil {
		return sessionPayload{}, ErrSessionMalformedCookie
	}

	if len(codec.verifyKeys) == 0 {
		return sessionPayload{}, ErrSessionSigningKeyMissing
	}

	validSig := false
	for _, verifyKey := range codec.verifyKeys {
		mac := hmac.New(sha256.New, verifyKey)
		_, _ = mac.Write(body)
		expected := mac.Sum(nil)
		if hmac.Equal(sig, expected) {
			validSig = true
			break
		}
	}
	if !validSig {
		return sessionPayload{}, ErrSessionInvalidSignature
	}

	if len(codec.decryptKeys) > 0 {
		decrypted := []byte(nil)
		for _, decryptKey := range codec.decryptKeys {
			plain, err := decryptSessionBody(body, decryptKey)
			if err == nil {
				decrypted = plain
				break
			}
		}
		if len(decrypted) == 0 {
			return sessionPayload{}, ErrSessionMalformedCookie
		}
		body = decrypted
	}

	payload := sessionPayload{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return sessionPayload{}, ErrSessionMalformedCookie
	}

	return payload, nil
}

func validateAEADKey(key []byte) error {
	switch len(key) {
	case 16, 24, 32:
		return nil
	default:
		return errors.New("encryption key must be 16, 24, or 32 bytes")
	}
}

func encryptSessionBody(plain, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nil, nonce, plain, nil)
	out := make([]byte, 0, len(nonce)+len(sealed))
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

func decryptSessionBody(blob, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(blob) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce := blob[:nonceSize]
	cipherText := blob[nonceSize:]
	return gcm.Open(nil, nonce, cipherText, nil)
}

type sessionResponseWriter struct {
	http.ResponseWriter
	request     *http.Request
	config      SessionConfig
	session     *SessionState
	codec       sessionCodec
	committed   bool
	wroteHeader bool
}

func (w *sessionResponseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		_ = w.commit()
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *sessionResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		_ = w.commit()
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(data)
}

func (w *sessionResponseWriter) commit() error {
	if w.committed || w.session == nil {
		return nil
	}
	w.committed = true

	if w.session.destroy {
		http.SetCookie(w.ResponseWriter, &http.Cookie{
			Name:     w.config.CookieName,
			Value:    "",
			Path:     w.config.Path,
			Domain:   w.config.Domain,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			Secure:   w.config.Secure,
			HttpOnly: w.config.HTTPOnly,
			SameSite: w.config.SameSite,
		})
		return nil
	}

	if !w.session.dirty {
		return nil
	}

	w.session.payload.ExpiresAt = time.Now().Add(w.config.MaxAge).Unix()
	encoded, err := encodeSessionWithCodec(w.session.payload, w.codec)
	if err != nil {
		return err
	}

	http.SetCookie(w.ResponseWriter, &http.Cookie{
		Name:     w.config.CookieName,
		Value:    encoded,
		Path:     w.config.Path,
		Domain:   w.config.Domain,
		MaxAge:   int(w.config.MaxAge.Seconds()),
		Expires:  time.Now().Add(w.config.MaxAge),
		Secure:   w.config.Secure,
		HttpOnly: w.config.HTTPOnly,
		SameSite: w.config.SameSite,
	})
	return nil
}

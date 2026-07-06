package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mobentum/kern"
)

func TestSession_PersistsValuesAcrossRequests(t *testing.T) {
	app := kern.New()
	app.Use(Session(SessionConfig{SigningKey: []byte("secret-key")}))

	app.GET("/set", func(c *kern.Context) {
		session, ok := GetSession(c.Context())
		if !ok {
			c.NoContent(http.StatusInternalServerError)
			return
		}
		session.Set("user", "alice")
		_ = c.Text(http.StatusOK, "ok")
	})

	app.GET("/get", func(c *kern.Context) {
		session, ok := GetSession(c.Context())
		if !ok {
			c.NoContent(http.StatusInternalServerError)
			return
		}
		user, exists := session.Get("user")
		if !exists {
			_ = c.Text(http.StatusOK, "")
			return
		}
		_ = c.Text(http.StatusOK, "%v", user)
	})

	setReq := httptest.NewRequest(http.MethodGet, "/set", nil)
	setRes := httptest.NewRecorder()
	app.ServeHTTP(setRes, setReq)

	if setRes.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", setRes.Code)
	}
	cookies := setRes.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/get", nil)
	getReq.AddCookie(cookies[0])
	getRes := httptest.NewRecorder()
	app.ServeHTTP(getRes, getReq)

	if got := getRes.Body.String(); got != "alice" {
		t.Fatalf("got %q, want %q", got, "alice")
	}
}

func TestSession_TamperedCookieStartsFreshSession(t *testing.T) {
	config := SessionConfig{SigningKey: []byte("secret-key")}
	app := kern.New()
	app.Use(Session(config))

	app.GET("/set", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.Set("role", "admin")
		_ = c.Text(http.StatusOK, "ok")
	})

	app.GET("/get", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		role, _ := session.Get("role")
		_ = c.Text(http.StatusOK, "%v", role)
	})

	setReq := httptest.NewRequest(http.MethodGet, "/set", nil)
	setRes := httptest.NewRecorder()
	app.ServeHTTP(setRes, setReq)

	cookie := setRes.Result().Cookies()[0]
	cookie.Value = cookie.Value + "tampered"

	getReq := httptest.NewRequest(http.MethodGet, "/get", nil)
	getReq.AddCookie(cookie)
	getRes := httptest.NewRecorder()
	app.ServeHTTP(getRes, getReq)

	if got := getRes.Body.String(); got != "<nil>" {
		t.Fatalf("got %q, want %q", got, "<nil>")
	}
}

func TestSession_DoesNotSetCookieWhenUnchanged(t *testing.T) {
	app := kern.New()
	app.Use(Session(SessionConfig{SigningKey: []byte("secret-key")}))
	app.GET("/get", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/get", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if cookies := res.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("expected no session cookies, got %d", len(cookies))
	}
}

func TestSession_FlashesAreOneTime(t *testing.T) {
	app := kern.New()
	app.Use(Session(SessionConfig{SigningKey: []byte("secret-key")}))

	app.GET("/set", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.AddFlash("welcome")
		_ = c.Text(http.StatusOK, "ok")
	})

	app.GET("/flash", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		values := session.Flashes()
		if len(values) == 0 {
			_ = c.Text(http.StatusOK, "none")
			return
		}
		_ = c.Text(http.StatusOK, "%v", values[0])
	})

	setReq := httptest.NewRequest(http.MethodGet, "/set", nil)
	setRes := httptest.NewRecorder()
	app.ServeHTTP(setRes, setReq)
	cookie := setRes.Result().Cookies()[0]

	flashReq1 := httptest.NewRequest(http.MethodGet, "/flash", nil)
	flashReq1.AddCookie(cookie)
	flashRes1 := httptest.NewRecorder()
	app.ServeHTTP(flashRes1, flashReq1)
	if got := flashRes1.Body.String(); got != "welcome" {
		t.Fatalf("got %q, want %q", got, "welcome")
	}

	cookies := flashRes1.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected updated session cookie after consuming flash")
	}

	flashReq2 := httptest.NewRequest(http.MethodGet, "/flash", nil)
	flashReq2.AddCookie(cookies[0])
	flashRes2 := httptest.NewRecorder()
	app.ServeHTTP(flashRes2, flashReq2)
	if got := flashRes2.Body.String(); got != "none" {
		t.Fatalf("got %q, want %q", got, "none")
	}
}

func TestSession_ExpiredPayloadCreatesNewSession(t *testing.T) {
	config := SessionConfig{SigningKey: []byte("secret-key"), MaxAge: time.Hour}
	expiredPayload := sessionPayload{
		ID:        "old",
		CreatedAt: time.Now().Add(-2 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-time.Hour).Unix(),
		Values:    map[string]interface{}{"user": "alice"},
	}

	encoded, err := encodeSession(expiredPayload, config.SigningKey)
	if err != nil {
		t.Fatalf("encode session: %v", err)
	}

	app := kern.New()
	app.Use(Session(config))
	app.GET("/id", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		_ = c.Text(http.StatusOK, "%s", session.ID())
	})

	req := httptest.NewRequest(http.MethodGet, "/id", nil)
	req.AddCookie(&http.Cookie{Name: "_session", Value: encoded})
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if got := res.Body.String(); got == "old" {
		t.Fatalf("expected new session ID, got %q", got)
	}
}

func TestSession_VerifyKeyRotation(t *testing.T) {
	oldKey := []byte("old-signing-key")
	newKey := []byte("new-signing-key")
	payload := sessionPayload{
		ID:        "rotated",
		CreatedAt: time.Now().Add(-time.Minute).Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Values:    map[string]interface{}{"user": "alice"},
		Flashes:   map[string][]interface{}{},
	}

	encoded, err := encodeSession(payload, oldKey)
	if err != nil {
		t.Fatalf("encode session: %v", err)
	}

	app := kern.New()
	app.Use(Session(SessionConfig{
		SigningKey: newKey,
		VerifyKeys: [][]byte{oldKey},
	}))
	app.GET("/user", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		user, _ := session.Get("user")
		_ = c.Text(http.StatusOK, "%v", user)
	})

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	req.AddCookie(&http.Cookie{Name: "_session", Value: encoded})
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if got := res.Body.String(); got != "alice" {
		t.Fatalf("got %q, want %q", got, "alice")
	}
}

func TestSession_EncryptionRoundTrip(t *testing.T) {
	app := kern.New()
	app.Use(Session(SessionConfig{
		SigningKey:    []byte("signing-key"),
		EncryptionKey: []byte("0123456789abcdef0123456789abcdef"),
	}))

	app.GET("/set", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.Set("user", "alice")
		_ = c.Text(http.StatusOK, "%s", "ok")
	})
	app.GET("/get", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		user, _ := session.Get("user")
		_ = c.Text(http.StatusOK, "%v", user)
	})

	setReq := httptest.NewRequest(http.MethodGet, "/set", nil)
	setRes := httptest.NewRecorder()
	app.ServeHTTP(setRes, setReq)

	cookies := setRes.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
	if strings.Contains(cookies[0].Value, "alice") {
		t.Fatal("expected encrypted cookie value")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/get", nil)
	getReq.AddCookie(cookies[0])
	getRes := httptest.NewRecorder()
	app.ServeHTTP(getRes, getReq)

	if got := getRes.Body.String(); got != "alice" {
		t.Fatalf("got %q, want %q", got, "alice")
	}
}

func TestSession_Delete(t *testing.T) {
	app := kern.New()
	app.Use(Session(SessionConfig{SigningKey: []byte("secret-key")}))

	app.GET("/set", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.Set("key", "value")
		_ = c.Text(http.StatusOK, "set")
	})

	app.GET("/delete", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.Delete("key")
		_, exists := session.Get("key")
		if exists {
			_ = c.Text(http.StatusOK, "still-exists")
			return
		}
		_ = c.Text(http.StatusOK, "deleted")
	})

	setReq := httptest.NewRequest(http.MethodGet, "/set", nil)
	setRes := httptest.NewRecorder()
	app.ServeHTTP(setRes, setReq)
	cookie := setRes.Result().Cookies()[0]

	delReq := httptest.NewRequest(http.MethodGet, "/delete", nil)
	delReq.AddCookie(cookie)
	delRes := httptest.NewRecorder()
	app.ServeHTTP(delRes, delReq)
	if got := delRes.Body.String(); got != "deleted" {
		t.Fatalf("got %q, want %q", got, "deleted")
	}
}

func TestSession_Clear(t *testing.T) {
	app := kern.New()
	app.Use(Session(SessionConfig{SigningKey: []byte("secret-key")}))

	app.GET("/set", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.Set("a", "1")
		session.Set("b", "2")
		session.AddFlash("flash-msg")
		_ = c.Text(http.StatusOK, "set")
	})

	app.GET("/check", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		_, aExists := session.Get("a")
		_, bExists := session.Get("b")
		flashes := session.Flashes()
		if aExists || bExists || len(flashes) > 0 {
			_ = c.Text(http.StatusOK, "dirty")
			return
		}
		_ = c.Text(http.StatusOK, "cleared")
	})

	setReq := httptest.NewRequest(http.MethodGet, "/set", nil)
	setRes := httptest.NewRecorder()
	app.ServeHTTP(setRes, setReq)
	cookie := setRes.Result().Cookies()[0]

	app2 := kern.New()
	app2.Use(Session(SessionConfig{SigningKey: []byte("secret-key")}))
	app2.GET("/clear", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.Clear()
		_ = c.Text(http.StatusOK, "cleared")
	})
	app2.GET("/check", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		_, aExists := session.Get("a")
		_, bExists := session.Get("b")
		flashes := session.Flashes()
		if aExists || bExists || len(flashes) > 0 {
			_ = c.Text(http.StatusOK, "dirty")
			return
		}
		_ = c.Text(http.StatusOK, "cleared")
	})

	clearReq := httptest.NewRequest(http.MethodGet, "/clear", nil)
	clearReq.AddCookie(cookie)
	clearRes := httptest.NewRecorder()
	app2.ServeHTTP(clearRes, clearReq)
	newCookie := clearRes.Result().Cookies()[0]

	checkReq := httptest.NewRequest(http.MethodGet, "/check", nil)
	checkReq.AddCookie(newCookie)
	checkRes := httptest.NewRecorder()
	app2.ServeHTTP(checkRes, checkReq)
	if got := checkRes.Body.String(); got != "cleared" {
		t.Fatalf("got %q, want %q", got, "cleared")
	}
}

func TestSession_Destroy(t *testing.T) {
	app := kern.New()
	app.Use(Session(SessionConfig{SigningKey: []byte("secret-key")}))

	app.GET("/destroy", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.Destroy()
		_ = c.Text(http.StatusOK, "done")
	})

	app.GET("/check", func(c *kern.Context) {
		session, ok := GetSession(c.Context())
		if !ok {
			_ = c.Text(http.StatusOK, "no-session")
			return
		}
		_ = c.Text(http.StatusOK, "session-%s", session.ID())
	})

	// Set a session first
	app2 := kern.New()
	app2.Use(Session(SessionConfig{SigningKey: []byte("secret-key")}))
	app2.GET("/set", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.Set("k", "v")
		_ = c.Text(http.StatusOK, "set")
	})

	setReq := httptest.NewRequest(http.MethodGet, "/set", nil)
	setRes := httptest.NewRecorder()
	app2.ServeHTTP(setRes, setReq)
	cookie := setRes.Result().Cookies()[0]

	destroyReq := httptest.NewRequest(http.MethodGet, "/destroy", nil)
	destroyReq.AddCookie(cookie)
	destroyRes := httptest.NewRecorder()
	app.ServeHTTP(destroyRes, destroyReq)

	// Cookie should be expired (MaxAge < 0)
	cookies := destroyRes.Result().Cookies()
	if len(cookies) > 0 && cookies[0].MaxAge >= 0 {
		t.Fatalf("expected expired cookie (MaxAge < 0), got MaxAge=%d", cookies[0].MaxAge)
	}
}

func TestSession_RegenerateID(t *testing.T) {
	app := kern.New()
	app.Use(Session(SessionConfig{SigningKey: []byte("secret-key")}))

	app.GET("/set", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		session.Set("k", "v")
		_ = c.Text(http.StatusOK, "set")
	})

	app.GET("/regenerate", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		oldID := session.ID()
		session.RegenerateID()
		if session.ID() == oldID {
			_ = c.Text(http.StatusOK, "same")
			return
		}
		_ = c.Text(http.StatusOK, "regenerated")
	})

	setReq := httptest.NewRequest(http.MethodGet, "/set", nil)
	setRes := httptest.NewRecorder()
	app.ServeHTTP(setRes, setReq)
	cookie := setRes.Result().Cookies()[0]

	regReq := httptest.NewRequest(http.MethodGet, "/regenerate", nil)
	regReq.AddCookie(cookie)
	regRes := httptest.NewRecorder()
	app.ServeHTTP(regRes, regReq)
	if got := regRes.Body.String(); got != "regenerated" {
		t.Fatalf("got %q, want %q", got, "regenerated")
	}
}

func TestSession_DecodeSession_Error(t *testing.T) {
	// Invalid base64
	_, err := decodeSession("invalid---", []byte("key"))
	if err == nil {
		t.Fatal("expected error for malformed cookie")
	}

	// No dot separator
	_, err = decodeSession("no-separator", []byte("key"))
	if err == nil {
		t.Fatal("expected error for missing dot")
	}

	// Empty parts
	_, err = decodeSession(".", []byte("key"))
	if err == nil {
		t.Fatal("expected error for empty parts")
	}

	// No verify keys (use valid base64 to pass decode step)
	validPart := base64.RawURLEncoding.EncodeToString([]byte("data"))
	_, err = decodeSessionWithCodec(validPart+"."+validPart, sessionCodec{})
	if err != ErrSessionSigningKeyMissing {
		t.Fatalf("got %v, want ErrSessionSigningKeyMissing", err)
	}
}

func TestSession_DecodeSession_SigningKey(t *testing.T) {
	// Invalid signature should fail
	payload := sessionPayload{ID: "test", Values: map[string]interface{}{}}
	encoded, err := encodeSession(payload, []byte("real-key"))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	_, err = decodeSession(encoded, []byte("wrong-key"))
	if err == nil {
		t.Fatal("expected error for wrong signing key")
	}
}

func TestSession_DecryptKeyRotation(t *testing.T) {
	oldEncKey := []byte("11111111111111112222222222222222")
	newEncKey := []byte("33333333333333334444444444444444")
	signingKey := []byte("signing-key")
	payload := sessionPayload{
		ID:        "enc-rotated",
		CreatedAt: time.Now().Add(-time.Minute).Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Values:    map[string]interface{}{"user": "alice"},
		Flashes:   map[string][]interface{}{},
	}

	encoded, err := encodeSessionWithCodec(payload, sessionCodec{
		signingKey:    signingKey,
		verifyKeys:    [][]byte{signingKey},
		encryptionKey: oldEncKey,
		decryptKeys:   [][]byte{oldEncKey},
	})
	if err != nil {
		t.Fatalf("encode session: %v", err)
	}

	app := kern.New()
	app.Use(Session(SessionConfig{
		SigningKey:    signingKey,
		EncryptionKey: newEncKey,
		DecryptKeys:   [][]byte{oldEncKey},
	}))
	app.GET("/user", func(c *kern.Context) {
		session, _ := GetSession(c.Context())
		user, _ := session.Get("user")
		_ = c.Text(http.StatusOK, "%v", user)
	})

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	req.AddCookie(&http.Cookie{Name: "_session", Value: encoded})
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if got := res.Body.String(); got != "alice" {
		t.Fatalf("got %q, want %q", got, "alice")
	}
}

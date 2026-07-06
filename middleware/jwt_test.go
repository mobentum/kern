package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mobentum/kern"
)

func TestJWT_ValidToken(t *testing.T) {
	secret := []byte("secret")
	app := kern.New()
	app.Use(JWT(JWTConfig{SigningKey: secret}))
	app.GET("/secure", func(c *kern.Context) {
		claims, ok := GetJWTClaims(c.Context())
		if !ok {
			_ = c.Error(http.StatusUnauthorized, "missing claims")
			return
		}
		_ = c.Text(http.StatusOK, "%v", claims["sub"])
	})

	token := createHS256Token(t, secret, map[string]interface{}{
		"sub": "user-1",
		"exp": time.Now().Add(time.Minute).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if got := res.Body.String(); got != "user-1" {
		t.Fatalf("got %q, want %q", got, "user-1")
	}
}

func TestJWT_InvalidSignature(t *testing.T) {
	secret := []byte("secret")
	app := kern.New()
	app.Use(JWT(JWTConfig{SigningKey: secret}))
	app.GET("/secure", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	token := createHS256Token(t, []byte("different"), map[string]interface{}{
		"sub": "user-1",
		"exp": time.Now().Add(time.Minute).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", res.Code)
	}
}

func TestJWT_ExpiredToken(t *testing.T) {
	secret := []byte("secret")
	app := kern.New()
	app.Use(JWT(JWTConfig{SigningKey: secret}))
	app.GET("/secure", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	token := createHS256Token(t, secret, map[string]interface{}{
		"sub": "user-1",
		"exp": time.Now().Add(-time.Minute).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", res.Code)
	}
}

func TestJWT_CustomValidator(t *testing.T) {
	secret := []byte("secret")
	app := kern.New()
	app.Use(JWT(JWTConfig{
		SigningKey: secret,
		ValidateClaims: func(claims map[string]interface{}, r *http.Request) error {
			if claims["tenant"] != "acme" {
				return ErrJWTMalformed
			}
			return nil
		},
	}))
	app.GET("/secure", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	token := createHS256Token(t, secret, map[string]interface{}{
		"sub":    "user-1",
		"tenant": "acme",
		"exp":    time.Now().Add(time.Minute).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func createHS256Token(t *testing.T, secret []byte, claims map[string]interface{}) string {
	t.Helper()

	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadPart := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerPart + "." + payloadPart

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	signaturePart := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signaturePart
}

func TestJWT_VerifyHS256TokenTyped(t *testing.T) {
	secret := []byte("my-secret")

	t.Run("valid token", func(t *testing.T) {
		type CustomClaims struct {
			Sub string `json:"sub"`
		}

		token := createHS256Token(t, secret, map[string]interface{}{
			"sub": "user-1",
		})

		result, err := verifyHS256TokenTyped(token, secret, &CustomClaims{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		claims, ok := result.(*CustomClaims)
		if !ok || claims.Sub != "user-1" {
			t.Fatalf("got %+v, want sub=user-1", result)
		}
	})

	t.Run("malformed token - missing parts", func(t *testing.T) {
		_, err := verifyHS256TokenTyped("only-two-parts", secret, &struct{}{})
		if err != ErrJWTMalformed {
			t.Fatalf("got %v, want ErrJWTMalformed", err)
		}
	})

	t.Run("malformed base64 header", func(t *testing.T) {
		_, err := verifyHS256TokenTyped("!!!.payload.signature", secret, &struct{}{})
		if err != ErrJWTMalformed {
			t.Fatalf("got %v, want ErrJWTMalformed", err)
		}
	})

	t.Run("unsupported algorithm", func(t *testing.T) {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
		payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"u1"}`))
		sig := base64.RawURLEncoding.EncodeToString([]byte("sig"))
		token := header + "." + payload + "." + sig

		_, err := verifyHS256TokenTyped(token, secret, &struct{}{})
		if err != ErrJWTUnsupportedAlg {
			t.Fatalf("got %v, want ErrJWTUnsupportedAlg", err)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
		payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"u1"}`))
		sig := base64.RawURLEncoding.EncodeToString([]byte("invalidsignature"))
		token := header + "." + payload + "." + sig

		_, err := verifyHS256TokenTyped(token, secret, &struct{}{})
		if err != ErrJWTInvalidSig {
			t.Fatalf("got %v, want ErrJWTInvalidSig", err)
		}
	})

	t.Run("malformed payload json", func(t *testing.T) {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
		payload := base64.RawURLEncoding.EncodeToString([]byte(`not-json`))
		mac := hmac.New(sha256.New, secret)
		mac.Write([]byte(header + "." + payload))
		sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
		token := header + "." + payload + "." + sig

		_, err := verifyHS256TokenTyped(token, secret, &struct{}{})
		if err != ErrJWTMalformed {
			t.Fatalf("got %v, want ErrJWTMalformed", err)
		}
	})
}

func TestJWT_NumericClaimToInt64(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		v, err := numericClaimToInt64(float64(123))
		if err != nil || v != 123 {
			t.Fatalf("got %d, %v", v, err)
		}
	})

	t.Run("int64", func(t *testing.T) {
		v, err := numericClaimToInt64(int64(456))
		if err != nil || v != 456 {
			t.Fatalf("got %d, %v", v, err)
		}
	})

	t.Run("int", func(t *testing.T) {
		v, err := numericClaimToInt64(789)
		if err != nil || v != 789 {
			t.Fatalf("got %d, %v", v, err)
		}
	})

	t.Run("json.Number", func(t *testing.T) {
		v, err := numericClaimToInt64(json.Number("999"))
		if err != nil || v != 999 {
			t.Fatalf("got %d, %v", v, err)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := numericClaimToInt64([]string{"a"})
		if err == nil {
			t.Fatal("expected error for invalid type")
		}
	})
}

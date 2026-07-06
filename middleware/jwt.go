package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mobentum/kern"
)

var (
	ErrJWTMalformed      = errors.New("jwt token is malformed")
	ErrJWTInvalidScheme  = errors.New("jwt authorization scheme is invalid")
	ErrJWTUnsupportedAlg = errors.New("jwt algorithm is unsupported")
	ErrJWTInvalidSig     = errors.New("jwt signature is invalid")
	ErrJWTExpired        = errors.New("jwt token is expired")
	ErrJWTNotYetValid    = errors.New("jwt token is not active")
)

const jwtClaimsContextKey contextKey = "jwtClaims"

var jwtDot = []byte{"."[0]}

// JWTConfig configures JWT middleware.
type JWTConfig struct {
	SigningKey []byte
	AuthScheme string
	ContextKey interface{}
	ClaimsType interface{}

	ValidateClaims func(claims map[string]interface{}, r *http.Request) error
	ErrorHandler   func(w http.ResponseWriter, r *http.Request, err error)
}

// JWT authenticates HS256 bearer tokens and stores claims in request context.
func JWT(config JWTConfig) kern.MiddlewareFunc {
	if len(config.SigningKey) == 0 {
		panic("middleware.JWT: SigningKey must not be empty")
	}

	authScheme := config.AuthScheme
	if authScheme == "" {
		authScheme = "Bearer"
	}

	claimsKey := config.ContextKey
	if claimsKey == nil {
		claimsKey = jwtClaimsContextKey
	}

	useTypedClaims := config.ClaimsType != nil
	var genericValidator func(map[string]interface{}, *http.Request) error
	validateClaims := config.ValidateClaims
	if validateClaims == nil && !useTypedClaims {
		genericValidator = validateJWTTimeClaims
	}

	errorHandler := config.ErrorHandler
	if errorHandler == nil {
		errorHandler = defaultJWTErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := parseAuthorizationToken(r.Header.Get("Authorization"), authScheme)
			if err != nil {
				errorHandler(w, r, err)
				return
			}

			var claims interface{}
			var err2 error
			if useTypedClaims {
				claims, err2 = verifyHS256TokenTyped(token, config.SigningKey, config.ClaimsType)
			} else {
				claims, err2 = verifyHS256Token(token, config.SigningKey)
			}
			if err2 != nil {
				errorHandler(w, r, err2)
				return
			}

			if genericValidator != nil {
				claimsMap, _ := claims.(map[string]interface{})
				if err := genericValidator(claimsMap, r); err != nil {
					errorHandler(w, r, err)
					return
				}
			} else if validateClaims != nil {
				claimsMap, _ := claims.(map[string]interface{})
				if err := validateClaims(claimsMap, r); err != nil {
					errorHandler(w, r, err)
					return
				}
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetJWTClaims returns JWT claims from request context.
func GetJWTClaims(ctx context.Context) (map[string]interface{}, bool) {
	claims, ok := ctx.Value(jwtClaimsContextKey).(map[string]interface{})
	return claims, ok
}

func defaultJWTErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="restricted"`)
	http.Error(w, err.Error(), http.StatusUnauthorized)
}

func parseAuthorizationToken(value, scheme string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrJWTMalformed
	}

	sep := strings.IndexByte(value, ' ')
	if sep <= 0 {
		return "", ErrJWTMalformed
	}

	if !strings.EqualFold(value[:sep], scheme) {
		return "", ErrJWTInvalidScheme
	}

	token := strings.TrimSpace(value[sep+1:])
	if token == "" {
		return "", ErrJWTMalformed
	}

	return token, nil
}

func verifyHS256Token(token string, signingKey []byte) (map[string]interface{}, error) {
	headerPart, payloadPart, signaturePart, ok := splitJWTToken(token)
	if !ok {
		return nil, ErrJWTMalformed
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(headerPart)
	if err != nil {
		return nil, ErrJWTMalformed
	}

	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrJWTMalformed
	}

	if header.Alg != "HS256" {
		return nil, ErrJWTUnsupportedAlg
	}

	mac := hmac.New(sha256.New, signingKey)
	_, _ = io.WriteString(mac, headerPart)
	_, _ = mac.Write(jwtDot)
	_, _ = io.WriteString(mac, payloadPart)
	expected := mac.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		return nil, ErrJWTMalformed
	}

	if !hmac.Equal(signature, expected) {
		return nil, ErrJWTInvalidSig
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return nil, ErrJWTMalformed
	}

	claims := map[string]interface{}{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, ErrJWTMalformed
	}

	return claims, nil
}

func splitJWTToken(token string) (headerPart, payloadPart, signaturePart string, ok bool) {
	firstDot := strings.IndexByte(token, '.')
	if firstDot <= 0 {
		return "", "", "", false
	}

	secondRel := strings.IndexByte(token[firstDot+1:], '.')
	if secondRel <= 0 {
		return "", "", "", false
	}

	secondDot := firstDot + 1 + secondRel
	if secondDot >= len(token)-1 {
		return "", "", "", false
	}

	if strings.IndexByte(token[secondDot+1:], '.') >= 0 {
		return "", "", "", false
	}

	return token[:firstDot], token[firstDot+1 : secondDot], token[secondDot+1:], true
}

func verifyHS256TokenTyped(token string, signingKey []byte, claimsType interface{}) (interface{}, error) {
	headerPart, payloadPart, signaturePart, ok := splitJWTToken(token)
	if !ok {
		return nil, ErrJWTMalformed
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(headerPart)
	if err != nil {
		return nil, ErrJWTMalformed
	}

	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrJWTMalformed
	}

	if header.Alg != "HS256" {
		return nil, ErrJWTUnsupportedAlg
	}

	mac := hmac.New(sha256.New, signingKey)
	_, _ = io.WriteString(mac, headerPart)
	_, _ = mac.Write(jwtDot)
	_, _ = io.WriteString(mac, payloadPart)
	expected := mac.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		return nil, ErrJWTMalformed
	}

	if !hmac.Equal(signature, expected) {
		return nil, ErrJWTInvalidSig
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return nil, ErrJWTMalformed
	}

	result := claimsType
	if err := json.Unmarshal(payloadBytes, &result); err != nil {
		return nil, ErrJWTMalformed
	}

	return result, nil
}

func validateJWTTimeClaims(claims map[string]interface{}, _ *http.Request) error {
	now := time.Now().Unix()

	if expRaw, ok := claims["exp"]; ok {
		exp, err := numericClaimToInt64(expRaw)
		if err != nil {
			return fmt.Errorf("invalid exp claim: %w", err)
		}
		if now >= exp {
			return ErrJWTExpired
		}
	}

	if nbfRaw, ok := claims["nbf"]; ok {
		nbf, err := numericClaimToInt64(nbfRaw)
		if err != nil {
			return fmt.Errorf("invalid nbf claim: %w", err)
		}
		if now < nbf {
			return ErrJWTNotYetValid
		}
	}

	return nil
}

func numericClaimToInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, err
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}

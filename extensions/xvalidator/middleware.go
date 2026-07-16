package xvalidator

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mobentum/kern"
)

type validatedKey[T any] struct{}

func Validated[T any](ctx context.Context) (T, bool) {
	v, ok := ctx.Value(validatedKey[T]{}).(T)
	return v, ok
}

func BodyValidator[T any]() kern.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body T
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
				return
			}
			if err := Validate(body); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				json.NewEncoder(w).Encode(map[string]any{"error": "validation failed", "fields": err})
				return
			}
			ctx := context.WithValue(r.Context(), validatedKey[T]{}, body)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

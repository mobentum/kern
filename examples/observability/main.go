package main

import (
	"log"
	"net/http"

	"github.com/mobentum/kern"
	"github.com/mobentum/kern/middleware"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func ObserveGuardDenies(route string) kern.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, req)
			if rec.status == http.StatusBadRequest {
				log.Printf("guard_deny route=%s method=%s path=%s", route, req.Method, req.URL.Path)
			}
		})
	}
}

func main() {
	app := kern.New()
	app.Use(kern.Logger())
	app.Use(kern.Recovery())
	app.Use(middleware.Session(middleware.SessionConfig{
		SigningKey:    []byte("new-signing-key"),
		VerifyKeys:    [][]byte{[]byte("old-signing-key")},
		EncryptionKey: []byte("0123456789abcdef0123456789abcdef"),
		DecryptKeys:   [][]byte{[]byte("abcdef0123456789abcdef0123456789")},
	}))

	app.POST("/api/login", func(c *kern.Context) {
		session, ok := middleware.GetSession(c.Context())
		if !ok {
			c.NoContent(http.StatusInternalServerError)
			return
		}
		session.Set("user_id", "demo-user")
		_ = c.JSON(http.StatusOK, map[string]string{"status": "logged_in"})
	})

	app.RouteWithMiddleware(http.MethodPost, "/api/upload", func(c *kern.Context) {
		_ = c.JSON(http.StatusAccepted, map[string]string{"status": "accepted"})
	},
		ObserveGuardDenies("upload"),
		middleware.RequestGuard(middleware.RequestGuardConfig{
			RequireBody:       true,
			RequireHeaders:    []string{"X-Tenant"},
			AllowContentTypes: []string{"application/json"},
			MaxBodyBytes:      1 << 20,
		}),
	)

	app.GET("/api/me", func(c *kern.Context) {
		rawCookie, _ := c.Cookie("_session")
		session, ok := middleware.GetSession(c.Context())
		if !ok {
			c.NoContent(http.StatusInternalServerError)
			return
		}

		userID, hasUser := session.Get("user_id")
		if rawCookie != nil && !hasUser {
			log.Printf("session_cookie_rejected path=%s", c.Request.URL.Path)
			_ = c.JSON(http.StatusUnauthorized, map[string]string{"error": "session expired"})
			return
		}
		if !hasUser {
			_ = c.JSON(http.StatusUnauthorized, map[string]string{"error": "not logged in"})
			return
		}

		_ = c.JSON(http.StatusOK, map[string]interface{}{"user_id": userID})
	})

	log.Println("Server running at http://localhost:8000")
	log.Println("Try:")
	log.Println("  curl -i -X POST http://localhost:8000/api/upload")
	log.Println("  curl -i -X POST http://localhost:8000/api/upload -H 'X-Tenant: acme' -H 'Content-Type: application/json' -d '{\"name\":\"demo\"}'")
	log.Println("  curl -i -X POST http://localhost:8000/api/login")
	log.Println("  curl -i http://localhost:8000/api/me")

	log.Fatal(app.Run("localhost:8000"))
}

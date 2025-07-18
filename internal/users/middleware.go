package users

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var rateLimiters = make(map[string]*rateLimiter)
var rlMu sync.Mutex

type rateLimiter struct {
	last  time.Time
	count int
}

func RequireRole(role string, jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Missing or invalid token"}`))
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			token, _ := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				if claims["role"] != role {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"error":"Insufficient permissions"}`))
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Invalid token"}`))
		})
	}
}

func RateLimitMiddleware(limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			rlMu.Lock()
			lim, ok := rateLimiters[ip]
			if !ok || time.Since(lim.last) > window {
				lim = &rateLimiter{last: time.Now(), count: 0}
				rateLimiters[ip] = lim
			}
			if lim.count >= limit {
				rlMu.Unlock()
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Too many requests, slow down"}`))
				return
			}
			lim.count++
			rlMu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

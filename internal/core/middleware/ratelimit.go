package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/response"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// visitors keeps track of rate limiters per IP address.
var (
	visitors = make(map[string]*visitor)
	mu       sync.Mutex
)

// init démarre une goroutine pour nettoyer la map et éviter les fuites de mémoire.
func init() {
	go cleanupVisitors()
}

func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

func getVisitor(ip string, rps int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		// Le burst autorise le double du débit nominal pour absorber les rafales du panel.
		limiter := rate.NewLimiter(rate.Limit(rps), rps*2)
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// clientIP retourne l'IP du client, en tenant compte du reverse proxy si celui-ci est déclaré de confiance.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			if first, _, found := strings.Cut(forwarded, ","); found {
				forwarded = first
			}
			if ip := strings.TrimSpace(forwarded); ip != "" {
				return ip
			}
		}
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-Ip")); realIP != "" {
			return realIP
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// RateLimit is a middleware that applies rate limiting per IP address.
func RateLimit(cfg *config.Config) func(http.Handler) http.Handler {
	rps := cfg.RateLimit
	if rps <= 0 {
		rps = 10
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			limiter := getVisitor(clientIP(r, cfg.TrustProxy), rps)
			if !limiter.Allow() {
				response.SendError(
					w,
					http.StatusTooManyRequests,
					"Rate limit exceeded. Try again later.",
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

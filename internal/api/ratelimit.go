package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientData
}

type clientData struct {
	tokens int
	last   time.Time
}

func newRateLimiter() *rateLimiter {
	rl := &rateLimiter{
		clients: make(map[string]*clientData),
	}
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.mu.Lock()
			for ip, client := range rl.clients {
				if time.Since(client.last) > 3*time.Minute {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		rl.mu.Lock()
		client, exists := rl.clients[ip]
		if !exists {
			client = &clientData{tokens: 5, last: time.Now()}
			rl.clients[ip] = client
		}

		if time.Since(client.last) > time.Minute {
			client.tokens = 5
		}
		client.last = time.Now()

		if client.tokens <= 0 {
			rl.mu.Unlock()
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		client.tokens--
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

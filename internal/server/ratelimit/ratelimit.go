package ratelimit

import (
	"sync"
	"time"
)

// Limiter implements a thread-safe token bucket rate limiter for multiple IP addresses.
type Limiter struct {
	mu     sync.Mutex
	ips    map[string]*bucket
	rate   float64
	burst  int
}

// bucket stores the state of a token bucket for a specific IP.
type bucket struct {
	tokens float64
	last   time.Time
}

// New creates a new Limiter with the specified rate and burst.
func New(rate float64, burst int) *Limiter {
	return &Limiter{
		ips:   make(map[string]*bucket),
		rate:  rate,
		burst: burst,
	}
}

// Allow checks if a request from the given IP address is allowed.
func (l *Limiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, exists := l.ips[ip]
	if !exists {
		// New IP, initialize bucket with burst-1 tokens and return true.
		b = &bucket{
			tokens: float64(l.burst) - 1,
			last:   time.Now(),
		}
		l.ips[ip] = b
		return true
	}

	now := time.Now()
	delta := now.Sub(b.last).Seconds()
	b.tokens += delta * l.rate
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}
	b.last = now

	// Consume a token if available.
	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}

	return false
}

// Cleanup removes old entries from the map.
func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for ip, b := range l.ips {
		// Remove entries that haven't been seen for more than 10 minutes.
		if now.Sub(b.last) > time.Minute*10 {
			delete(l.ips, ip)
		}
	}
}

package ratelimit

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestLimiter_Allow(t *testing.T) {
	t.Run("should allow first request and initialize with burst-1 tokens", func(t *testing.T) {
		l := New(10, 5)
		ip := "127.0.0.1"

		assert.True(t, l.Allow(ip))
		
		l.mu.Lock()
		b := l.ips[ip]
		assert.Equal(t, float64(4), b.tokens)
		l.mu.Unlock()
	})

	t.Run("should respect burst limit", func(t *testing.T) {
		l := New(10, 5)
		ip := "127.0.0.1"

		// Consume all 5 tokens
		for i := range 5 {
			assert.True(t, l.Allow(ip), "Request %d should be allowed", i)
		}

		// 6th request should be denied
		assert.False(t, l.Allow(ip))
	})

	t.Run("should refill tokens over time", func(t *testing.T) {
		rate := 100.0 // 100 tokens per second
		l := New(rate, 5)
		ip := "127.0.0.1"

		// Consume all 5 tokens
		for range 5 {
			l.Allow(ip)
		}
		assert.False(t, l.Allow(ip))

		// Wait for 25ms, should refill 0.025 * 100 = 2.5 tokens
		time.Sleep(25 * time.Millisecond)

		assert.True(t, l.Allow(ip))
		assert.True(t, l.Allow(ip))
	})
	
	t.Run("should handle multiple IPs independently", func(t *testing.T) {
		l := New(10, 1)
		ip1 := "1.1.1.1"
		ip2 := "2.2.2.2"

		assert.True(t, l.Allow(ip1))
		assert.False(t, l.Allow(ip1))

		assert.True(t, l.Allow(ip2))
		assert.False(t, l.Allow(ip2))
	})
}

func TestLimiter_Cleanup(t *testing.T) {
	t.Run("should remove old entries", func(t *testing.T) {
		l := New(10, 5)
		now := time.Now()
		
		l.mu.Lock()
		l.ips["old"] = &bucket{tokens: 5, last: now.Add(-11 * time.Minute)}
		l.ips["new"] = &bucket{tokens: 5, last: now.Add(-1 * time.Minute)}
		l.mu.Unlock()

		l.Cleanup()

		l.mu.Lock()
		_, existsOld := l.ips["old"]
		_, existsNew := l.ips["new"]
		l.mu.Unlock()

		assert.False(t, existsOld)
		assert.True(t, existsNew)
	})
}

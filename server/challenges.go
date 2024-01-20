package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"go.uber.org/zap"
	"sync"
	"time"
)

var challengeTTL = 5 * time.Second

type Challenge []byte

func (c Challenge) Equals(other Challenge) bool {
	return bytes.Equal(c, other)
}

func (c Challenge) Copy() Challenge {
	return bytes.Clone(c)
}

type ChallengeMap struct {
	log        *zap.SugaredLogger
	mu         sync.Mutex
	challenges map[string]Challenge
	chTimes    map[string]time.Time
}

func NewChallengeMap(log *zap.SugaredLogger) *ChallengeMap {
	return &ChallengeMap{
		log:        log,
		mu:         sync.Mutex{},
		challenges: make(map[string]Challenge),
		chTimes:    make(map[string]time.Time),
	}
}

func (c *ChallengeMap) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// cleanup checks for and removes any expired challenges.
func (c *ChallengeMap) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var toBeRemoved []string

	// Collect every key that has expired.
	for key, t := range c.chTimes {
		// Check if challenge expired, and if so remove it.
		if now.After(t.Add(challengeTTL)) {
			toBeRemoved = append(toBeRemoved, key)
		}
	}

	// Remove everything that expired.
	for _, key := range toBeRemoved {
		c.remove(key)
	}
}

func (c *ChallengeMap) AddChallenge(key string) Challenge {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := c.genChallenge()
	c.challenges[key] = ch
	c.chTimes[key] = time.Now()

	return ch.Copy()
}

func (c *ChallengeMap) Validate(key string, ch Challenge) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	actualCH, exists := c.challenges[key]
	if !exists || !actualCH.Equals(ch) {
		return false
	}

	// Challenge was valid, remove before returning true.
	c.remove(key)
	return true
}

func (c *ChallengeMap) remove(key string) {
	delete(c.challenges, key)
	delete(c.chTimes, key)
}

func (c *ChallengeMap) genChallenge() Challenge {
	ch := make(Challenge, 4)
	_, err := rand.Read(ch)
	if err != nil {
		c.log.Errorf("unable to generate random challenge: %v", err)
		ch = []byte{0x0, 0x0, 0x0, 0x0}
	}

	return ch
}

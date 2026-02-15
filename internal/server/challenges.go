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

type challengeEntry struct {
	challenge Challenge
	createdAt time.Time
}

type ChallengeMap struct {
	log        *zap.SugaredLogger
	mu         sync.Mutex
	challenges map[string]challengeEntry
}

func NewChallengeMap(log *zap.SugaredLogger) *ChallengeMap {
	return &ChallengeMap{
		log:        log,
		mu:         sync.Mutex{},
		challenges: make(map[string]challengeEntry),
	}
}

func (c *ChallengeMap) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
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

	// Collect every key that has expired.
	for key, entry := range c.challenges {
		// Check if challenge expired, and if so remove it.
		if now.After(entry.createdAt.Add(challengeTTL)) {
			delete(c.challenges, key)
		}
	}
}

func (c *ChallengeMap) AddChallenge(key string) Challenge {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := c.genChallenge()
	c.challenges[key] = challengeEntry{
		challenge: ch,
		createdAt: time.Now(),
	}

	return ch.Copy()
}

func (c *ChallengeMap) Validate(key string, ch Challenge) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.challenges[key]
	if !exists || !entry.challenge.Equals(ch) {
		return false
	}

	// Challenge was valid, remove before returning true.
	delete(c.challenges, key)
	return true
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

package server

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
	"time"
)

func Test_cleanup(t *testing.T) {
	test := func(name string, before, expected map[string]challengeEntry) {
		t.Run(name, func(t0 *testing.T) {
			c := NewChallengeMap(zap.NewNop().Sugar())
			c.challenges = before

			c.cleanup()

			assert.Equal(t0, expected, c.challenges)
		})
	}

	now := time.Now()
	test(
		"should not remove challenges that haven't expired",
		map[string]challengeEntry{"a": {challenge: []byte{0x0, 0x1, 0x2, 0x3}, createdAt: now}},
		map[string]challengeEntry{"a": {challenge: []byte{0x0, 0x1, 0x2, 0x3}, createdAt: now}},
	)

	test(
		"should remove challenges that have expired",
		map[string]challengeEntry{"a": {challenge: []byte{0x0, 0x1, 0x2, 0x3}, createdAt: now.Add(-time.Hour)}},
		map[string]challengeEntry{},
	)
}

func Test_AddChallenge(t *testing.T) {
	t.Run("should generate, add to internal map and return copy of challenge", func(t0 *testing.T) {
		c := NewChallengeMap(zap.NewNop().Sugar())
		key := "test"

		chCopy := c.AddChallenge(key)

		// Verify that the challenge exists
		entry, exists := c.challenges[key]
		assert.True(t0, exists)

		// Verify that we set the challenge time.
		assert.True(t0, entry.createdAt.After(time.Now().Add(-10*time.Second)))

		// Alter the returned challenge, verify we did not change the original.
		chCopy[1] = 0xf2
		assert.NotEqual(t0, entry.challenge, chCopy)
	})
}

func Test_Validate(t *testing.T) {
	t.Run("should return false if challenge does not exist", func(t0 *testing.T) {
		c := NewChallengeMap(zap.NewNop().Sugar())
		key := "test"
		ch := []byte("test")

		isValid := c.Validate(key, ch)
		assert.False(t0, isValid)
	})

	t.Run("should remove challenge and return true if exists", func(t0 *testing.T) {
		c := NewChallengeMap(zap.NewNop().Sugar())
		key := "test"

		// Add challenge.
		ch := c.AddChallenge(key)

		// Validate the challenge.
		isValid := c.Validate(key, ch)
		assert.True(t0, isValid)

		// Verify the challenge was removed from the map.
		_, exists := c.challenges[key]
		assert.False(t0, exists)
	})
}

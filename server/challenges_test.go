package server

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
	"time"
)

func Test_cleanup(t *testing.T) {
	test := func(name string, beforeCH, expectedCH map[string]Challenge, beforeTTL, expectedTTL map[string]time.Time) {
		t.Run(name, func(t0 *testing.T) {
			c := NewChallengeMap(zap.NewNop().Sugar())
			c.challenges = beforeCH
			c.chTimes = beforeTTL

			c.cleanup()

			assert.Equal(t0, expectedCH, c.challenges)
			for key, _ := range c.chTimes {
				_, exists := expectedTTL[key]
				assert.True(t0, exists)
			}
		})
	}

	now := time.Now()
	test(
		"should not remove challenges that haven't expired",
		map[string]Challenge{"a": []byte{0x0, 0x1, 0x2, 0x3}},
		map[string]Challenge{"a": []byte{0x0, 0x1, 0x2, 0x3}},
		map[string]time.Time{"a": now},
		map[string]time.Time{"a": now},
	)

	test(
		"should remove challenges that have expired",
		map[string]Challenge{"a": []byte{0x0, 0x1, 0x2, 0x3}},
		map[string]Challenge{},
		map[string]time.Time{"a": time.Now().Add(-time.Hour)},
		map[string]time.Time{},
	)
}

func Test_AddChallenge(t *testing.T) {
	t.Run("should generate, add to internal maps and return copy of challenge", func(t0 *testing.T) {
		c := NewChallengeMap(zap.NewNop().Sugar())
		key := "test"

		chCopy := c.AddChallenge(key)

		// Verify that the challenge exists
		ch, exists := c.challenges[key]
		assert.True(t0, exists)

		// Verify that we set the challenge time.
		t := c.chTimes[key]
		assert.True(t0, t.After(time.Now().Add(-10*time.Second)))

		// Alter the returned challenge, verify we did not change the original.
		chCopy[1] = 0xf2
		assert.NotEqual(t0, ch, chCopy)
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

		// Verify the challenge was removed from both maps.
		_, exists := c.challenges[key]
		assert.False(t0, exists)

		_, exists = c.chTimes[key]
		assert.False(t0, exists)
	})
}

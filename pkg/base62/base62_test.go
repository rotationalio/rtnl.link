package base62_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/rotationalio/rtnl.link/pkg/base62"
	"github.com/stretchr/testify/require"
)

func TestBase62(t *testing.T) {
	source := rand.NewSource(time.Now().UnixNano())
	rand := rand.New(source)

	// Test 1000 random uint64 encode and decode strings
	for i := 0; i < 1000; i++ {
		num := rand.Uint64()
		encoded := base62.Encode(num)

		length := len(encoded)
		require.GreaterOrEqual(t, length, 5, "expected length to be at least 5 chars")
		require.LessOrEqual(t, length, 11, "expected length to be at most 11 chars")

		decoded, err := base62.Decode(encoded)
		require.NoError(t, err, "unable to decode encoded base62 string")
		require.Equal(t, num, decoded, "decoded num does not match original")
	}
}

func TestBase62Invalid(t *testing.T) {
	testCases := []string{
		"a&b",
		"abcdefghijklmnopqrsg",
	}

	for i, tc := range testCases {
		_, err := base62.Decode(tc)
		require.Error(t, err, "expected error on test case %d", i)
	}
}

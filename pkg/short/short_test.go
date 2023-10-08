package short_test

import (
	"testing"

	"github.com/rotationalio/rtnl.link/pkg/short"
	"github.com/stretchr/testify/require"
)

func TestURL(t *testing.T) {
	testCases := []struct {
		in       string
		expected string
	}{
		{"https://rotational.io", "okV7czZRVbs"},
		{"https://intersog.com/blog/how-to-write-a-custom-url-shortener-using-golang-and-redis/#toc-anchor-4", "2ZnactvxBpn"},
		{"https://example.com?foo=bar&color=red", "DN2TaxzJOVe"},
		{"http://localhost:8080/example", "u1QbsHxRQfh"},
		{"https://example.com/index.html", "D822dZX23Ym"},
	}

	for i, tc := range testCases {
		actual, err := short.URL(tc.in)
		require.NoError(t, err, "could not shorten test case %d", i)
		require.Equal(t, tc.expected, actual, "mismatch on test case %d", i)
	}
}

func TestShorten(t *testing.T) {
	testCases := []struct {
		in       string
		expected string
	}{
		{"", ""},
		{"this is the song that never ends", "NgNExPA3nOd"},
		{"https://rotational.io", "okV7czZRVbs"},
	}

	for i, tc := range testCases {
		actual, err := short.Shorten(tc.in)
		require.NoError(t, err, "could not shorten test case %d", i)
		require.Equal(t, tc.expected, actual, "mismatch on test case %d", i)
	}
}

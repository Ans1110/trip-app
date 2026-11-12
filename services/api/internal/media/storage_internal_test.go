package media

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEndpoint(t *testing.T) {
	cases := []struct {
		raw, scheme, host string
		ssl               bool
	}{
		{"localhost:9010", "http", "localhost:9010", false},
		{"cdn.example.com", "https", "cdn.example.com", true},
		{"https://media.example.com", "https", "media.example.com", false},
		{"http://127.0.0.1:9000", "http", "127.0.0.1:9000", true},
	}
	for _, c := range cases {
		u, err := parseEndpoint(c.raw, c.ssl)
		require.NoError(t, err, c.raw)
		assert.Equal(t, c.scheme, u.Scheme, c.raw)
		assert.Equal(t, c.host, u.Host, c.raw)
	}
}

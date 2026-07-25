package valkey

import (
	"crypto/tls"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/v2/js/modulestest"
	"go.k6.io/k6/v2/lib"
	"go.k6.io/k6/v2/lib/netext"
	"go.k6.io/k6/v2/lib/types"
	"go.k6.io/k6/v2/metrics"
	"gopkg.in/guregu/null.v3"
)

func newVUState(t testing.TB) (*modulestest.Runtime, *lib.State) {
	t.Helper()

	runtime := modulestest.NewRuntime(t)
	samples := make(chan metrics.SampleContainer, 1000)

	state := &lib.State{
		Dialer: netext.NewDialer(
			net.Dialer{},
			netext.NewResolver(net.LookupIP, 0, types.DNSfirst, types.DNSpreferIPv4),
		),
		Options: lib.Options{
			SystemTags: metrics.NewSystemTagSet(
				metrics.TagURL,
				metrics.TagProto,
				metrics.TagStatus,
				metrics.TagSubproto,
			),
			UserAgent: null.StringFrom("TestUserAgent"),
		},
		Samples:        samples,
		TLSConfig:      &tls.Config{},
		BuiltinMetrics: metrics.RegisterBuiltinMetrics(metrics.NewRegistry()),
		Tags:           lib.NewVUStateTags(metrics.NewRegistry().RootTagSet()),
	}
	runtime.MoveToVUContext(state)

	return runtime, state
}

func TestSharedClientRegistry_GetOrCreate(t *testing.T) {
	t.Parallel()

	rs := RunT(t)

	registry := newSharedClientRegistry()
	_, state := newVUState(t)

	opts, _, err := readOptions(map[string]any{
		"socket": map[string]any{"host": rs.Addr().IP.String(), "port": int64(rs.Addr().Port)},
	})
	require.NoError(t, err)

	key := optionsKey(opts)

	// First call creates a new client.
	client1, err := registry.getOrCreate(key, opts, state)
	require.NoError(t, err)
	require.NotNil(t, client1)

	// Second call with the same key returns the same client.
	client2, err := registry.getOrCreate(key, opts, state)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%p", client1), fmt.Sprintf("%p", client2), "should return the same client instance")

	// Only one TCP connection should have been established.
	assert.Equal(t, 1, rs.HandledConnectionsCount())

	// Cleanup.
	registry.release(key)
	registry.release(key)
}

func TestSharedClientRegistry_Release(t *testing.T) {
	t.Parallel()

	rs := RunT(t)

	registry := newSharedClientRegistry()
	_, state := newVUState(t)

	opts, _, err := readOptions(map[string]any{
		"socket": map[string]any{"host": rs.Addr().IP.String(), "port": int64(rs.Addr().Port)},
	})
	require.NoError(t, err)

	key := optionsKey(opts)

	// Two references.
	_, err = registry.getOrCreate(key, opts, state)
	require.NoError(t, err)
	_, err = registry.getOrCreate(key, opts, state)
	require.NoError(t, err)

	// First release should not remove the entry.
	registry.release(key)
	registry.mu.Lock()
	_, exists := registry.clients[key]
	registry.mu.Unlock()
	assert.True(t, exists, "entry should still exist after first release")

	// Second release should close and remove the entry.
	registry.release(key)
	registry.mu.Lock()
	_, exists = registry.clients[key]
	registry.mu.Unlock()
	assert.False(t, exists, "entry should be removed after last release")
}

func TestSharedClientRegistry_DifferentKeys(t *testing.T) {
	t.Parallel()

	rs1 := RunT(t)
	rs2 := RunT(t)

	registry := newSharedClientRegistry()
	_, state := newVUState(t)

	opts1, _, err := readOptions(map[string]any{
		"socket": map[string]any{"host": rs1.Addr().IP.String(), "port": int64(rs1.Addr().Port)},
	})
	require.NoError(t, err)

	opts2, _, err := readOptions(map[string]any{
		"socket": map[string]any{"host": rs2.Addr().IP.String(), "port": int64(rs2.Addr().Port)},
	})
	require.NoError(t, err)

	key1 := optionsKey(opts1)
	key2 := optionsKey(opts2)
	assert.NotEqual(t, key1, key2)

	client1, err := registry.getOrCreate(key1, opts1, state)
	require.NoError(t, err)

	client2, err := registry.getOrCreate(key2, opts2, state)
	require.NoError(t, err)

	assert.NotEqual(t, fmt.Sprintf("%p", client1), fmt.Sprintf("%p", client2), "different keys should produce different clients")

	registry.release(key1)
	registry.release(key2)
}

func TestOptionsKey(t *testing.T) {
	t.Parallel()

	opts1, _, err := readOptions(map[string]any{
		"socket": map[string]any{"host": "localhost", "port": int64(6379)},
	})
	require.NoError(t, err)

	opts2, _, err := readOptions(map[string]any{
		"socket": map[string]any{"host": "localhost", "port": int64(6379)},
	})
	require.NoError(t, err)

	assert.Equal(t, optionsKey(opts1), optionsKey(opts2))

	opts3, _, err := readOptions(map[string]any{
		"socket":   map[string]any{"host": "localhost", "port": int64(6379)},
		"username": "user1",
	})
	require.NoError(t, err)

	assert.NotEqual(t, optionsKey(opts1), optionsKey(opts3))

	opts4, _, err := readOptions(map[string]any{
		"socket":   map[string]any{"host": "localhost", "port": int64(6379)},
		"password": "pass1",
	})
	require.NoError(t, err)

	opts5, _, err := readOptions(map[string]any{
		"socket":   map[string]any{"host": "localhost", "port": int64(6379)},
		"password": "pass2",
	})
	require.NoError(t, err)

	assert.NotEqual(t, optionsKey(opts4), optionsKey(opts5), "different passwords should produce different keys")

	opts6, _, err := readOptions(map[string]any{
		"socket": map[string]any{
			"host": "localhost", "port": int64(6379),
			"tls": map[string]any{"skipVerify": true},
		},
	})
	require.NoError(t, err)

	opts7, _, err := readOptions(map[string]any{
		"socket": map[string]any{
			"host": "localhost", "port": int64(6379),
			"tls": map[string]any{"skipVerify": false},
		},
	})
	require.NoError(t, err)

	assert.NotEqual(t, optionsKey(opts6), optionsKey(opts7), "different skipVerify should produce different keys")
}

func TestSharedClientRegistry_ReleaseOneKeepsOther(t *testing.T) {
	t.Parallel()

	rs := RunT(t)
	rs.RegisterCommandHandler("GET", func(c *Connection, args []string) {
		c.WriteBulkString("value")
	})

	registry := newSharedClientRegistry()
	_, state := newVUState(t)

	opts, _, err := readOptions(map[string]any{
		"socket": map[string]any{"host": rs.Addr().IP.String(), "port": int64(rs.Addr().Port)},
	})
	require.NoError(t, err)

	key := optionsKey(opts)

	// Two references.
	client, err := registry.getOrCreate(key, opts, state)
	require.NoError(t, err)
	_, err = registry.getOrCreate(key, opts, state)
	require.NoError(t, err)

	// Release one reference.
	registry.release(key)

	// The client should still be usable (not closed).
	resp := client.Do(t.Context(), client.B().Get().Key("test").Build())
	val, err := resp.ToString()
	require.NoError(t, err)
	assert.Equal(t, "value", val)

	// Final release.
	registry.release(key)
}

func TestApplyDialerSkipVerify(t *testing.T) {
	t.Parallel()

	// The per-client and global flags are OR-ed: either one enabling
	// verification skipping is enough, and neither may clobber the other.
	testCases := []struct {
		name     string
		client   bool
		global   bool
		expected bool
	}{
		{"neither", false, false, false},
		{"client_only", true, false, true},
		{"global_only", false, true, true},
		{"both", true, true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, state := newVUState(t)
			state.TLSConfig.InsecureSkipVerify = tc.global

			opts, _, err := readOptions(map[string]any{
				"socket": map[string]any{
					"host": "localhost",
					"port": int64(6379),
					"tls":  map[string]any{"skipVerify": tc.client},
				},
			})
			require.NoError(t, err)

			applyDialer(&opts, state)

			require.NotNil(t, opts.TLSConfig)
			assert.Equal(t, tc.expected, opts.TLSConfig.InsecureSkipVerify)
		})
	}
}

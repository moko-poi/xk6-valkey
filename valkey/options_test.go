package valkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsResp3(t *testing.T) {
	t.Parallel()

	t.Run("single_node/default_resp2", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"socket": map[string]any{"host": "localhost", "port": int64(6379)},
		})
		require.NoError(t, err)
		assert.True(t, opts.AlwaysRESP2)
	})

	t.Run("single_node/resp3_enabled", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"socket": map[string]any{"host": "localhost", "port": int64(6379)},
			"resp3":  true,
		})
		require.NoError(t, err)
		assert.False(t, opts.AlwaysRESP2)
	})

	t.Run("cluster/resp3_enabled", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"cluster": map[string]any{
				"resp3": true,
				"nodes": []any{"redis://host1:6379", "redis://host2:6379"},
			},
		})
		require.NoError(t, err)
		assert.False(t, opts.AlwaysRESP2)
	})

	t.Run("sentinel/resp3_enabled", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"socket":     map[string]any{"host": "localhost", "port": int64(26379)},
			"masterName": "mymaster",
			"resp3":      true,
		})
		require.NoError(t, err)
		assert.False(t, opts.AlwaysRESP2)
	})
}

func TestOptionsClientSideCache(t *testing.T) {
	t.Parallel()

	t.Run("single_node/default_disabled", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"socket": map[string]any{"host": "localhost", "port": int64(6379)},
		})
		require.NoError(t, err)
		assert.True(t, opts.DisableCache)
		assert.True(t, opts.AlwaysRESP2)
	})

	t.Run("single_node/enabled", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"socket":          map[string]any{"host": "localhost", "port": int64(6379)},
			"clientSideCache": true,
		})
		require.NoError(t, err)
		assert.False(t, opts.DisableCache)
		assert.False(t, opts.AlwaysRESP2, "client-side cache requires RESP3")
	})

	t.Run("cluster/enabled", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"cluster": map[string]any{
				"clientSideCache": true,
				"nodes":           []any{"redis://host1:6379"},
			},
		})
		require.NoError(t, err)
		assert.False(t, opts.DisableCache)
		assert.False(t, opts.AlwaysRESP2)
	})

	t.Run("sentinel/enabled", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"socket":          map[string]any{"host": "localhost", "port": int64(26379)},
			"masterName":      "mymaster",
			"clientSideCache": true,
		})
		require.NoError(t, err)
		assert.False(t, opts.DisableCache)
		assert.False(t, opts.AlwaysRESP2)
	})
}

func TestOptionsReadOnly(t *testing.T) {
	t.Parallel()

	t.Run("cluster/readOnly_sets_sendToReplicas", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"cluster": map[string]any{
				"readOnly": true,
				"nodes":    []any{"redis://host1:6379"},
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, opts.SendToReplicas, "readOnly should set SendToReplicas function")
	})

	t.Run("cluster/readOnly_default_nil", func(t *testing.T) {
		t.Parallel()

		opts, _, err := newOptionsFromObject(map[string]any{
			"cluster": map[string]any{
				"nodes": []any{"redis://host1:6379"},
			},
		})
		require.NoError(t, err)
		assert.Nil(t, opts.SendToReplicas)
	})
}

func TestOptionsShared(t *testing.T) {
	t.Parallel()

	t.Run("shared_true", func(t *testing.T) {
		t.Parallel()

		opts, shared, err := newOptionsFromObject(map[string]any{
			"socket": map[string]any{"host": "localhost", "port": int64(6379)},
			"shared": true,
		})
		require.NoError(t, err)
		assert.True(t, shared)
		assert.Equal(t, []string{"localhost:6379"}, opts.InitAddress)
	})

	t.Run("shared_false_explicit", func(t *testing.T) {
		t.Parallel()

		_, shared, err := newOptionsFromObject(map[string]any{
			"socket": map[string]any{"host": "localhost", "port": int64(6379)},
			"shared": false,
		})
		require.NoError(t, err)
		assert.False(t, shared)
	})

	t.Run("shared_omitted_defaults_false", func(t *testing.T) {
		t.Parallel()

		_, shared, err := newOptionsFromObject(map[string]any{
			"socket": map[string]any{"host": "localhost", "port": int64(6379)},
		})
		require.NoError(t, err)
		assert.False(t, shared)
	})

	t.Run("shared_not_supported_for_url", func(t *testing.T) {
		t.Parallel()

		_, shared, err := readOptions("redis://localhost:6379")
		require.NoError(t, err)
		assert.False(t, shared)
	})
}

func TestOptionsURLDefaultsToRESP2(t *testing.T) {
	t.Parallel()

	opts, err := newOptionsFromString("redis://localhost:6379")
	require.NoError(t, err)
	assert.True(t, opts.AlwaysRESP2)
	assert.True(t, opts.DisableCache)
}

func TestOptionsRemovedFieldsRejected(t *testing.T) {
	t.Parallel()

	rejectedFields := []struct {
		name  string
		field string
		value any
	}{
		{"dialTimeout", "dialTimeout", int64(5000)},
		{"readTimeout", "readTimeout", int64(5000)},
		{"writeTimeout", "writeTimeout", int64(5000)},
		{"poolSize", "poolSize", int64(10)},
		{"minIdleConns", "minIdleConns", int64(5)},
		{"maxConnAge", "maxConnAge", int64(3600000)},
		{"poolTimeout", "poolTimeout", int64(4000)},
		{"idleTimeout", "idleTimeout", int64(300000)},
		{"idleCheckFrequency", "idleCheckFrequency", int64(60000)},
	}

	for _, tc := range rejectedFields {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := newOptionsFromObject(map[string]any{
				"socket": map[string]any{
					"host":   "localhost",
					"port":   int64(6379),
					tc.field: tc.value,
				},
			})
			assert.Error(t, err, "removed option %s should be rejected", tc.field)
		})
	}

	clusterRejected := []struct {
		name  string
		field string
		value any
	}{
		{"maxRedirects", "maxRedirects", int64(3)},
		{"routeByLatency", "routeByLatency", true},
		{"routeRandomly", "routeRandomly", true},
	}

	for _, tc := range clusterRejected {
		t.Run("cluster/"+tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := newOptionsFromObject(map[string]any{
				"cluster": map[string]any{
					"nodes":  []any{"redis://host1:6379"},
					tc.field: tc.value,
				},
			})
			assert.Error(t, err, "removed cluster option %s should be rejected", tc.field)
		})
	}
}

package valkey

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	valkey "github.com/valkey-io/valkey-go"
	"go.k6.io/k6/lib"
)

// sharedClientEntry holds a shared valkey.Client along with a reference count.
type sharedClientEntry struct {
	client   valkey.Client
	refCount atomic.Int64
}

// sharedClientRegistry manages shared valkey.Client instances across VUs.
// When multiple VUs use `shared: true` with the same connection options,
// they share a single underlying valkey.Client, enabling auto-pipelining
// and realistic backend-like connection patterns.
type sharedClientRegistry struct {
	mu      sync.Mutex
	clients map[string]*sharedClientEntry
}

// newSharedClientRegistry creates a new registry.
func newSharedClientRegistry() *sharedClientRegistry {
	return &sharedClientRegistry{
		clients: make(map[string]*sharedClientEntry),
	}
}

// optionsKey generates a deterministic string key from the connection-identity
// fields of a valkey.ClientOption. Two options with the same key represent the
// same logical connection and can safely share a client.
func optionsKey(opts valkey.ClientOption) string {
	return fmt.Sprintf("addrs=%v;user=%s;pass=%s;db=%d;master=%s;tls=%t;resp2=%t;cache=%t",
		opts.InitAddress,
		opts.Username,
		opts.Password,
		opts.SelectDB,
		opts.Sentinel.MasterSet,
		opts.TLSConfig != nil,
		opts.AlwaysRESP2,
		opts.DisableCache,
	)
}

// getOrCreate returns an existing shared client for the given key, or creates
// a new one using the provided options and VU state. The caller must eventually
// call release with the same key.
func (r *sharedClientRegistry) getOrCreate(key string, opts valkey.ClientOption, vuState *lib.State) (valkey.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.clients[key]; ok {
		entry.refCount.Add(1)
		return entry.client, nil
	}

	applyDialer(&opts, vuState)

	client, err := valkey.NewClient(opts)
	if err != nil {
		return nil, err
	}

	entry := &sharedClientEntry{client: client}
	entry.refCount.Store(1)
	r.clients[key] = entry

	return client, nil
}

// release decrements the reference count for the given key. When it reaches
// zero the underlying client is closed and removed from the registry.
func (r *sharedClientRegistry) release(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.clients[key]
	if !ok {
		return
	}

	if entry.refCount.Add(-1) <= 0 {
		entry.client.Close()
		delete(r.clients, key)
	}
}

// applyDialer configures DialCtxFn on opts using the VU's netext.Dialer,
// including TLS upgrade when TLS is configured. This logic is shared between
// the per-VU connect path and the shared client registry.
func applyDialer(opts *valkey.ClientOption, vuState *lib.State) {
	tlsCfg := opts.TLSConfig
	if tlsCfg != nil && vuState.TLSConfig != nil {
		tlsCfg.InsecureSkipVerify = vuState.TLSConfig.InsecureSkipVerify
		tlsCfg.CipherSuites = vuState.TLSConfig.CipherSuites
		tlsCfg.MinVersion = vuState.TLSConfig.MinVersion
		tlsCfg.MaxVersion = vuState.TLSConfig.MaxVersion
		tlsCfg.Renegotiation = vuState.TLSConfig.Renegotiation
		tlsCfg.KeyLogWriter = vuState.TLSConfig.KeyLogWriter
		tlsCfg.Certificates = append(tlsCfg.Certificates, vuState.TLSConfig.Certificates...)

		if vuState.TLSConfig.RootCAs != nil {
			merged := vuState.TLSConfig.RootCAs.Clone()
			if tlsCfg.RootCAs != nil {
				for _, cert := range tlsCfg.RootCAs.Subjects() { //nolint:staticcheck
					merged.AppendCertsFromPEM(cert)
				}
			}
			tlsCfg.RootCAs = merged
		}

		opts.DialCtxFn = func(ctx context.Context, addr string, dialer *net.Dialer, config *tls.Config) (net.Conn, error) {
			rawConn, err := vuState.Dialer.DialContext(ctx, "tcp", addr)
			if err != nil {
				return nil, err
			}
			tlsConn := tls.Client(rawConn, config)
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				rawConn.Close()
				return nil, err
			}
			return tlsConn, nil
		}
	} else {
		opts.DialCtxFn = func(ctx context.Context, addr string, dialer *net.Dialer, config *tls.Config) (net.Conn, error) {
			return vuState.Dialer.DialContext(ctx, "tcp", addr)
		}
	}
}

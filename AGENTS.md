# xk6-valkey

k6 extension providing a Valkey client for load tests. Wraps valkey-go and supports single-node, cluster, and sentinel deployments. All commands return JS promises.

## Architecture

The extension follows k6's standard module pattern: a root module creates per-VU instances, each owning a Valkey client.

**Lazy connection**: The client stores options at construction time but does not connect until the first command executes inside a VU context. This satisfies k6's convention of no IO during init. Every command method checks `connect()` before proceeding.

**Promise pattern**: Every Valkey command creates a JS promise, validates arguments, spawns a goroutine for the actual Valkey call, and resolves or rejects the promise from that goroutine. This pattern is uniform across all 40+ command methods.

**Options parsing**: The constructor accepts either a URL string (single-node) or a JS object. The object form determines the client type: if `masterName` is present, sentinel is used; if `cluster` is present, a cluster client is used; otherwise, single-node. In cluster mode, all nodes must share consistent options (credentials, database) or construction fails.

**TLS merging**: When TLS is configured, the extension merges the user-provided TLS config with k6's VU-level TLS settings (cipher suites, min/max version, client certificates, renegotiation, key logging). It uses `DialCtxFn` to wrap k6's network dialer, manually upgrading the connection to TLS. Root CA merging is explicitly marked as a TODO and not yet implemented.

**Type validation**: Before sending values to Valkey, the client validates that arguments are of supported Go types (string, numeric, bool). Validation errors report the argument's position in the outer command call using a position offset.

## Gotchas

- Cluster node option consistency is enforced for most fields, but TLS config is only taken from the first node that provides it. If different nodes specify different TLS configs, only the first one wins silently.

- Tests use a custom in-process RESP protocol stub server over TCP, not a real Valkey. The stub auto-ignores CLIENT, HELLO, and CLUSTER commands. If your test relies on these command behaviors, you must register a handler explicitly.

- The linter config is not committed. It is downloaded from k6 core's master branch on first lint run. Do not commit it.

- The underlying valkey-go client is instantiated once per VU on first use. valkey-go uses auto-pipelining for high throughput. Pool-related JS options (poolSize, minIdleConns, etc.) are accepted but ignored, as valkey-go manages connections internally.

- RESP2 protocol is used by default (`AlwaysRESP2: true`) for compatibility. Client-side caching is disabled (`DisableCache: true`).

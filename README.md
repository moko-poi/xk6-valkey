# xk6-valkey

This is a [k6](https://github.com/grafana/k6) extension, it's a client library for [Valkey](https://valkey.io) (and Redis-compatible) databases, powered by [valkey-go](https://github.com/valkey-io/valkey-go).

## Get started

Add the extension's import
```js
import valkey from "k6/x/valkey";
```

Define the client
```js
import valkey from "k6/x/valkey";

// Instantiate a new Valkey client using a URL.
// The connection will be established on the first command call.
const client = new valkey.Client('redis://localhost:6379');
```

Call the API from the VU context

```js
export default function () {
  client.sadd("crocodile_ids", ...crocodileIDs);
...
}
```

### Arbitrary commands

`sendCommand` sends any command, but it passes every argument as a plain
argument. In cluster mode that means no hash slot is computed client-side, so
the command lands on an arbitrary node and only reaches the right one after a
`MOVED` redirect. For single-key commands, use:

```js
// Computes the slot from the key and goes straight to the owning node.
await client.sendKeyCommand("set", "mykey", "myvalue");

// Same, plus marks the command readonly so it can be served by a replica
// when the cluster client was created with `readOnly: true`.
await client.sendReadCommand("zrange", "myzset", 0, -1, "WITHSCORES");
```

Both take `(command, key, ...args)`. `sendCommand` remains the right choice for
commands with no key, or with several keys.

### Shared client

By default each VU gets its own client. Set `shared` to have all VUs of a k6
instance share a single underlying connection pool, which is closer to how a
real backend talks to Valkey:

```js
// URL form
const client = new valkey.Client("redis://localhost:6379?shared=true");

// Object form
const client = new valkey.Client({
  socket: { host: "localhost", port: 6379 },
  shared: true,
});
```

### TLS

`socket.tls.skipVerify` disables certificate verification for this client only.
It is OR-ed with k6's global `insecureSkipTLSVerify`, so enabling either one is
enough:

```js
const client = new valkey.Client({
  socket: {
    host: "localhost",
    port: 6379,
    tls: { skipVerify: true },
  },
});
```

## Build

The most common and simple case is to use k6 with automatic extension resolution. Simply add the extension's import and k6 will resolve the dependency automatically.
However, if you prefer to build it from source using xk6, first ensure you have the prerequisites:

- [Go toolchain](https://go101.org/article/go-toolchain.html)
- Git

Then:

1. Install `xk6`:
  ```shell
  go install go.k6.io/xk6/cmd/xk6@latest
  ```

2. Build the binary:
  ```shell
  xk6 build --with github.com/moko-poi/xk6-valkey
  ```

## Compatibility

| xk6-valkey | k6     | Go   |
|------------|--------|------|
| v0.2.0     | v2.1.0 | 1.25 |
| v0.1.0     | v1.5.0 | 1.25 |

The extension imports `go.k6.io/k6/v2`, so it must be built against k6 v2.x.
Building it into a k6 v1 binary fails with a "conflicting k6 versions" error;
use xk6-valkey v0.1.0 for k6 v1.

## Notice

This project is a derivative work of [xk6-redis](https://github.com/grafana/xk6-redis) by Grafana Labs, licensed under the Apache License 2.0. The Redis client library (go-redis) has been replaced with [valkey-go](https://github.com/valkey-io/valkey-go) to provide native Valkey support.

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.

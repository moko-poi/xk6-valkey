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

| xk6-valkey | k6    | Go   |
|------------|-------|------|
| v0.1.0     | v1.5.0 | 1.25 |

## Notice

This project is a derivative work of [xk6-redis](https://github.com/grafana/xk6-redis) by Grafana Labs, licensed under the Apache License 2.0. The Redis client library (go-redis) has been replaced with [valkey-go](https://github.com/valkey-io/valkey-go) to provide native Valkey support.

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.

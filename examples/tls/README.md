# How to run a k6 test against a Valkey test server with TLS

1. Move into the docker folder: `cd docker`
2. Run `sh gen-test-certs.sh` to generate the TLS certificates the container will use.
3. Run `docker compose up -d` to start the Valkey server with TLS enabled.
4. Verify the server is set up correctly:
   ```shell
   docker compose exec valkey valkey-cli --tls \
     --cert /tls/valkey.crt --key /tls/valkey.key --cacert /tls/ca.crt \
     -a tjkbZ8jrwz3pGiku ping
   ```
   It should answer `PONG`. The plaintext port is disabled, so a `valkey-cli` without `--tls` will fail.
5. Build the k6 binary: `xk6 build --with github.com/moko-poi/xk6-valkey=.` (from the repository root)
6. Run `./k6 run examples/tls/loadtest-tls.js` to run the k6 load test with TLS enabled.

The server runs the official [`valkey/valkey`](https://hub.docker.com/r/valkey/valkey) image. It has no
env-var configuration API, so TLS is configured through `valkey-server` flags in
`docker/docker-compose.yml`.

`gen-test-certs.sh` widens the generated keys to `0644`: the image drops
privileges to its `valkey` user, which otherwise cannot read the mounted key.
These are throwaway test certificates — do not reuse the script's permissions
for real ones.

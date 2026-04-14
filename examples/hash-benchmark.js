import valkey from "k6/x/valkey";
import exec from "k6/execution";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

export const options = {
  scenarios: {
    writeProfiles: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "writeProfiles",
    },
    readProfiles: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "readProfiles",
      startTime: "3s",
    },
  },
};

const client = new valkey.Client(
  __ENV.REDIS_URL || "redis://localhost:6379",
);

// Create and update user profiles
export function writeProfiles() {
  const userKey = `user:${exec.vu.idInTest}`;

  client
    .hset(userKey, "name", `user-${exec.vu.idInTest}`)
    .then(() => client.hset(userKey, "email", `user-${exec.vu.idInTest}@example.com`))
    .then(() => client.hset(userKey, "status", "active"))
    .then(() => client.hsetnx(userKey, "created_at", `${Date.now()}`))
    .then(() => client.hincrby(userKey, "login_count", 1))
    .then(() => client.hlen(userKey))
    .then((len) => {
      if (len < 4) {
        throw new Error(`expected at least 4 fields, got ${len}`);
      }
    });
}

// Read user profile data
export function readProfiles() {
  const userKey = `user:${exec.vu.idInTest}`;

  client
    .hgetall(userKey)
    .then((profile) => {
      if (!profile) return;

      return client.hget(userKey, "name");
    })
    .then(() => client.hkeys(userKey))
    .then(() => client.hvals(userKey))
    .then(() => {
      // Clean up a field
      return client.hdel(userKey, "status");
    });
}

export function teardown() {
  for (let i = 1; i <= 10; i++) {
    client.del(`user:${i}`);
  }
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: "  ", enableColors: true }),
  };
}

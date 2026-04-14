import valkey from "k6/x/valkey";
import exec from "k6/execution";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

export const options = {
  scenarios: {
    normalReads: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "normalReads",
    },
    cachedReads: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "cachedReads",
    },
  },
};

// Note: Client-side caching requires disableCache to NOT be set (default enables it).
// The DoCache API uses server-assisted client-side caching for read-heavy workloads.
const client = new valkey.Client(
  __ENV.REDIS_URL || "redis://localhost:6379",
);

const cacheTTL = 10; // seconds

export function setup() {
  // Seed data for read benchmarks
  for (let i = 0; i < 10; i++) {
    client.set(`cache-key-${i}`, `value-${i}`, 0);
    client.hset(`cache-hash-${i}`, "field1", `value-${i}`);
    client.lpush(`cache-list-${i}`, `item-a-${i}`, `item-b-${i}`, `item-c-${i}`);
  }
}

// Normal reads without client-side caching
export function normalReads() {
  const idx = exec.vu.idInTest % 10;

  client
    .get(`cache-key-${idx}`)
    .then(() => client.hgetall(`cache-hash-${idx}`))
    .then(() => client.lrange(`cache-list-${idx}`, 0, -1))
    .then(() => client.llen(`cache-list-${idx}`))
    .then(() => client.hget(`cache-hash-${idx}`, "field1"));
}

// Cached reads using DoCache API (server-assisted client-side caching)
export function cachedReads() {
  const idx = exec.vu.idInTest % 10;

  client
    .getCache(`cache-key-${idx}`, cacheTTL)
    .then(() => client.hgetallCache(`cache-hash-${idx}`, cacheTTL))
    .then(() => client.lrangeCache(`cache-list-${idx}`, 0, -1, cacheTTL))
    .then(() => client.llenCache(`cache-list-${idx}`, cacheTTL))
    .then(() => client.hgetCache(`cache-hash-${idx}`, "field1", cacheTTL));
}

export function teardown() {
  for (let i = 0; i < 10; i++) {
    client.del(`cache-key-${i}`);
    client.del(`cache-hash-${i}`);
    client.del(`cache-list-${i}`);
  }
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: "  ", enableColors: true }),
  };
}

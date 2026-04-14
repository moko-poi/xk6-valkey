import valkey from "k6/x/valkey";
import exec from "k6/execution";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

export const options = {
  scenarios: {
    producer: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "producer",
    },
    consumer: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "consumer",
      startTime: "5s",
    },
  },
};

const client = new valkey.Client(
  __ENV.REDIS_URL || "redis://localhost:6379",
);

// Producer: push messages to a queue
export function producer() {
  const queue = `queue-${exec.vu.idInTest}`;
  const message = `msg-${Date.now()}`;

  client
    .lpush(queue, message)
    .then(() => client.rpush(queue, `${message}-low-priority`))
    .then(() => client.llen(queue))
    .then((len) => {
      if (len < 2) {
        throw new Error(`expected at least 2 items, got ${len}`);
      }
    });
}

// Consumer: pop and inspect messages from the queue
export function consumer() {
  const queue = `queue-${exec.vu.idInTest}`;

  client
    .llen(queue)
    .then((len) => {
      if (len === 0) return;

      return client
        .lrange(queue, 0, -1)
        .then((items) => {
          if (items.length === 0) return;

          return client.lindex(queue, 0);
        })
        .then(() => client.rpop(queue))
        .then(() => client.lpop(queue));
    });
}

export function teardown() {
  for (let i = 1; i <= 10; i++) {
    client.del(`queue-${i}`);
  }
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: "  ", enableColors: true }),
  };
}

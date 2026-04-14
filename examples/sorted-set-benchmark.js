import valkey from "k6/x/valkey";
import exec from "k6/execution";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

export const options = {
  scenarios: {
    updateScores: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "updateScores",
    },
    readLeaderboard: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "readLeaderboard",
      startTime: "3s",
    },
  },
};

const client = new valkey.Client(
  __ENV.REDIS_URL || "redis://localhost:6379",
);

const leaderboard = "leaderboard";

export function setup() {
  // Seed initial leaderboard data
  for (let i = 0; i < 20; i++) {
    client.zadd(leaderboard, Math.random() * 1000, `player-${i}`);
  }
}

// Update player scores and manage the leaderboard
export function updateScores() {
  const player = `player-${exec.vu.idInTest}`;
  const score = Math.random() * 1000;

  client
    .zadd(leaderboard, score, player)
    .then(() => client.zscore(leaderboard, player))
    .then((currentScore) => {
      if (currentScore !== score) {
        throw new Error(`score mismatch: expected ${score}, got ${currentScore}`);
      }
    })
    .then(() => client.zrank(leaderboard, player))
    .then(() => client.zcard(leaderboard));
}

// Read leaderboard rankings
export function readLeaderboard() {
  client
    .zcard(leaderboard)
    .then((total) => {
      if (total === 0) return;

      // Get top 10 players
      return client.zrange(leaderboard, "0", "9");
    })
    .then(() => {
      // Clean up old low-score entries
      return client.zremrangebyscore(leaderboard, "-inf", "10");
    });
}

export function teardown() {
  client.del(leaderboard);
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: "  ", enableColors: true }),
  };
}

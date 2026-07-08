import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 100 }, // Ramp up to 100 users
    { duration: '30s', target: 500 }, // Ramp up to 500 users
    { duration: '20s', target: 0 },   // Scale down
  ],
  thresholds: {
    // Gateway added overhead should be small. Total latency depends on LLM mock.
    // For local mock, we target p99 < 30ms for gateway routing.
    http_req_duration: ['p(95)<20', 'p(99)<30'],
  },
};

export default function () {
  const payload = JSON.stringify({
    model: 'gemini-1.5-pro',
    messages: [
      { role: 'user', content: 'Hello, how are you?' },
    ],
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'k6-test-key',
    },
  };

  // Assuming Gateway is running on :8080
  const res = http.post('http://localhost:8080/v1/chat/completions', payload, params);

  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });

  sleep(0.1);
}

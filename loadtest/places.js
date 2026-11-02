import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    public_reads: {
      executor: "constant-vus",
      vus: Number(__ENV.VUS || 5),
      duration: __ENV.DURATION || "30s",
    },
  },
  // Candidate acceptance targets only; they are not observed production metrics.
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<500"],
  },
};

const baseURL = __ENV.BASE_URL || "http://localhost:8080";

export default function () {
  const responses = http.batch([
    ["GET", `${baseURL}/v1/place-collections`],
    ["GET", `${baseURL}/v1/places`],
    ["GET", `${baseURL}/v1/places/sulakskiy-kanon`],
  ]);

  for (const response of responses) {
    check(response, {
      "status is 200": (r) => r.status === 200,
      "etag is present": (r) => Boolean(r.headers.Etag),
    });
  }
  sleep(1);
}


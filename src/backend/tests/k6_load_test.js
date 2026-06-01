// ============================================================================
// UniHub Workshop — k6 Load Test Script (v2 — Pre-cached JWT Tokens)
// ============================================================================
// Dùng tokens.json được tạo bởi pre_login.py, KHÔNG login trong k6.
//
// WORKFLOW:
//   1. python3 tests/pre_login.py                          # Pre-login → tokens.json
//   2. k6 run src/backend/tests/k6_load_test.js --env PHASE=burst  # Chạy test
//
// PHASES:
//   smoke   — 10 VUs, 35s (kiểm tra nhanh)
//   stress  — 0→500 VUs, 3.5 phút
//   spike   — đột ngột 1000 VUs
//   burst   — 10,000 VUs ngay lập tức!
//   soak    — 200 VUs liên tục 5 phút
//
// OPTIONS:
//   --env BASE_URL=http://localhost:8080
//   --env PHASE=burst
//   --env TOKENS_FILE=data/tokens.json
// ============================================================================

import http from "k6/http";
import { check, sleep, group } from "k6";
import { Rate, Trend, Counter } from "k6/metrics";
import { SharedArray } from "k6/data";

// ── Custom Metrics ──
const registrationSuccess = new Rate("registration_success");
const registrationLatency = new Trend("registration_latency", true);
const http429Count = new Counter("rate_limited_429");
const http400Count = new Counter("seat_full_400");
const httpErrorCount = new Counter("server_errors");

// ── Configuration ──
const BASE_URL = __ENV.BASE_URL || "http://8.233.213.55";
const PHASE = __ENV.PHASE || "stress";
const TOKENS_FILE = __ENV.TOKENS_FILE || "../data/tokens.json";

// ── Load pre-cached tokens (chạy 1 lần lúc init, chia sẻ giữa tất cả VUs) ──
const tokens = new SharedArray("jwt_tokens", function () {
  const data = JSON.parse(open(TOKENS_FILE));
  console.log(`📦 Loaded ${data.length} cached tokens from ${TOKENS_FILE}`);
  return data;
});

// ── Scenarios ──
const scenarios = {
  smoke: {
    executor: "ramping-vus",
    stages: [
      { duration: "10s", target: 10 },
      { duration: "20s", target: 10 },
      { duration: "5s", target: 0 },
    ],
  },
  stress: {
    executor: "ramping-vus",
    stages: [
      { duration: "30s", target: 100 },
      { duration: "1m", target: 300 },
      { duration: "1m", target: 500 },
      { duration: "30s", target: 500 },
      { duration: "30s", target: 0 },
    ],
  },
  spike: {
    executor: "ramping-vus",
    stages: [
      { duration: "10s", target: 50 },
      { duration: "5s", target: 1000 },
      { duration: "1m", target: 1000 },
      { duration: "10s", target: 50 },
      { duration: "20s", target: 0 },
    ],
  },
  // 🔥 Burst: 10,000 VUs spike rồi giảm dần
  burst: {
    executor: "ramping-vus",
    stages: [
      { duration: "5s", target: 2000 },
      { duration: "5s", target: 5000 },
      { duration: "5s", target: 10000 },
      { duration: "1m", target: 10000 },
      { duration: "30s", target: 5000 },
      { duration: "20s", target: 1000 },
      { duration: "10s", target: 0 },
    ],
  },
  soak: {
    executor: "constant-vus",
    vus: 200,
    duration: "5m",
  },
};

export const options = {
  setupTimeout: "30s",
  teardownTimeout: "30s",
  scenarios: {
    default: scenarios[PHASE] || scenarios.stress,
  },
  thresholds: {
    http_req_duration: ["p(95)<5000"],
    registration_success: ["rate>0.01"],
    http_req_failed: ["rate<0.99"],
  },
  batch: 500,
  batchPerHost: 200,
  noConnectionReuse: false,
};

// ── Setup: chỉ cần lấy workshop ID, KHÔNG login ──
export function setup() {
  console.log(`\n🚀 UniHub k6 Load Test v2 — Phase: ${PHASE}`);
  console.log(`📍 Target: ${BASE_URL}`);
  console.log(`🔑 Tokens: ${tokens.length} (pre-cached, JWT valid 1 hour)`);
  console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`);

  // Health check
  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status !== 200) {
    console.error(`❌ Health check failed! Status: ${healthRes.status}`);
    return { error: true };
  }
  console.log("✅ Health check passed");

  // Get workshops
  const workshopsRes = http.get(`${BASE_URL}/api/v1/workshops`);
  let workshopId = "";
  let workshopTitle = "";

  if (workshopsRes.status === 200) {
    const body = workshopsRes.json();
    const workshops = body.data || body;
    if (Array.isArray(workshops) && workshops.length > 0) {
      let best = workshops[0];
      for (const w of workshops) {
        if ((w.available_seats || 0) > (best.available_seats || 0)) {
          best = w;
        }
      }
      workshopId = best.id;
      workshopTitle = `${best.title} (${best.available_seats}/${best.capacity} seats)`;
    }
  }

  if (!workshopId) {
    console.error("❌ No workshops found");
    return { error: true };
  }

  console.log(`✅ Target workshop: ${workshopTitle}`);
  console.log(`   Workshop ID: ${workshopId}\n`);

  return { workshopId, workshopTitle };
}

// ── Main Test: Mỗi VU lấy token từ SharedArray, KHÔNG login ──
export default function (data) {
  if (data.error) return;

  // Round-robin: mỗi VU lấy 1 token riêng
  const tokenIndex = (__VU - 1) % tokens.length;
  const token = tokens[tokenIndex];

  // ── Step 1: Get Workshops ──
  group("01_get_workshops", function () {
    const res = http.get(`${BASE_URL}/api/v1/workshops`, {
      headers: { Authorization: `Bearer ${token}` },
      tags: { name: "get_workshops" },
    });

    check(res, {
      "workshops status 200": (r) => r.status === 200,
    });
  });

  sleep(Math.random() * 0.3);

  // ── Step 2: Register for Workshop (Rate Limiter hoạt động ở đây) ──
  group("02_register", function () {
    const start = Date.now();
    const res = http.post(
      `${BASE_URL}/api/v1/registrations`,
      JSON.stringify({ workshop_id: data.workshopId }),
      {
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        tags: { name: "register" },
      }
    );
    const latency = Date.now() - start;
    registrationLatency.add(latency);

    if (res.status === 200 || res.status === 202) {
      registrationSuccess.add(1);
    } else if (res.status === 429) {
      http429Count.add(1);
      registrationSuccess.add(0);
    } else if (res.status === 400 || res.status === 409) {
      http400Count.add(1);
      registrationSuccess.add(0);
    } else {
      httpErrorCount.add(1);
      registrationSuccess.add(0);
    }

    check(res, {
      "register accepted (200/202)": (r) => r.status === 200 || r.status === 202,
      "register not server error": (r) => r.status < 500,
    });
  });

  // ── Step 3: Check Registration Status ──
  group("03_check_status", function () {
    const res = http.get(`${BASE_URL}/api/v1/registrations/my`, {
      headers: { Authorization: `Bearer ${token}` },
      tags: { name: "check_status" },
    });

    check(res, {
      "my registrations status 200": (r) => r.status === 200,
    });
  });

  sleep(Math.random() * 0.5);
}

// ── Teardown ──
export function teardown(data) {
  if (data.error) return;
  console.log(`\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
  console.log(`📊 Load test completed!`);
  console.log(`   Target: ${BASE_URL}`);
  console.log(`   Workshop: ${data.workshopTitle}`);
  console.log(`   Tokens used: ${tokens.length}`);
  console.log(`   Phase: ${PHASE}`);
  console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`);
}

import http from 'k6/http';
import { check, fail, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { Counter } from 'k6/metrics';
import exec from 'k6/execution';

const fixture = JSON.parse(open(__ENV.SESSIONS_FILE || '/fixtures/sessions.json'));
const sessions = new SharedArray('sessions', () => fixture.sessions);
const base = (__ENV.BASE_URL || 'http://unihub-api-service').replace(/\/$/, '');
const functionalErrors = new Counter('functional_errors');
const vus = Number(__ENV.VUS || 20);
export const options = {
  scenarios: {
    browse: {executor: 'constant-vus', vus, duration: __ENV.DURATION || '30s', exec: 'browse'},
    registration: {executor: 'shared-iterations', vus: Math.min(vus, sessions.length), iterations: sessions.length, maxDuration: '60s', startTime: '35s', exec: 'register'},
  },
  thresholds: {
    'http_req_duration{gate:latency}': ['p(95)<200'],
    http_req_failed: ['rate<0.005'],
    checks: ['rate==1'],
    functional_errors: ['count==0'],
  },
};
function params(session, latency = true) {
  return {headers: {'Authorization': `Bearer ${session.token}`, 'Content-Type': 'application/json', 'X-Device-ID': session.student_id}, tags: {gate: latency ? 'latency' : 'poll'}};
}
function valid(response, expected) {
  let success = false;
  try { success = response.status === expected && response.json('success') === true; } catch (_) {}
  if (!check(response, {'authenticated request succeeds': () => success})) functionalErrors.add(1);
  return success;
}
export function setup() {
  if (!sessions.length || vus > sessions.length || !fixture.workshop_id) fail('Fixtures must include workshop_id and at least VUS authenticated users');
  const probe = http.get(`${base}/api/v1/auth/me`, params(sessions[0]));
  if (!valid(probe, 200)) fail('Authentication preflight failed');
  functionalErrors.add(0);
}
export function browse() {
  const session = sessions[(__VU - 1) % sessions.length];
  valid(http.get(`${base}/api/v1/workshops`, params(session)), 200);
  valid(http.get(`${base}/api/v1/workshops/${fixture.workshop_id}`, params(session)), 200);
  valid(http.get(`${base}/api/v1/workshops/${fixture.workshop_id}/presence`, params(session)), 200);
  sleep(0.2);
}
export function register() {
  const session = sessions[exec.scenario.iterationInTest];
  const response = http.post(`${base}/api/v1/registrations`, JSON.stringify({workshop_id: fixture.workshop_id}), params(session));
  if (!valid(response, 202)) return;
  const id = response.json('data.correlation_id');
  // Verify completion by a real worker; a fast 202 alone is insufficient.
  for (let attempt = 0; attempt < 100; attempt++) {
    const status = http.get(`${base}/api/v1/registrations/status/${id}`, params(session, false));
    if (!valid(status, 200)) return;
    const state = status.json('data.status');
    if (state === 'SUCCESS') return;
    if (state === 'FAILED') break;
    sleep(0.2);
  }
  functionalErrors.add(1);
  check(false, {'registration reaches SUCCESS': () => false});
}
export function handleSummary(data) {
  const report = {passed: Object.values(data.metrics).every(m => !m.thresholds || Object.values(m.thresholds).every(t => t.ok)), metrics: {latency: data.metrics['http_req_duration{gate:latency}'], failed: data.metrics.http_req_failed, checks: data.metrics.checks, functional_errors: data.metrics.functional_errors}};
  return {stdout: JSON.stringify(report, null, 2) + '\n', [__ENV.SUMMARY_FILE || '/tmp/loadtest-summary.json']: JSON.stringify(report)};
}

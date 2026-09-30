import http from 'k6/http';
import { check, sleep } from 'k6';

// =============================================================================
// UNIHUB WORKSHOP — 2000 CONCURRENT USERS HEARTBEAT & BROWSING SIMULATION
// =============================================================================
// Simulates 2,000 students visiting the workshop portal simultaneously.
// Each Virtual User (VU) sends realistic presence heartbeats via X-Device-ID
// and browses workshop details.

export const options = {
    stages: [
        { duration: '15s', target: 500 },   // Ramp up to 500 students
        { duration: '20s', target: 2000 },  // Surge up to 2,000 students
        { duration: '45s', target: 2000 },  // Sustained 2,000 students online
        { duration: '15s', target: 0 },     // Students leave after session
    ],
    thresholds: {
        http_req_failed: ['rate<0.05'],    // Less than 5% errors even under surge
        http_req_duration: ['p(95)<1500'], // 95% of requests under 1.5s on 2 CPUs
    },
};

const BASE_URL = __ENV.TARGET_URL || 'http://192.168.58.2:30080';
const WORKSHOP_ID = __ENV.WORKSHOP_ID || 'dd77cc68-e69e-4b2e-b78f-f157a35952ef';

export default function () {
    // Unique device fingerprint per Virtual User
    const deviceId = `student-device-${__VU.toString().padStart(4, '0')}`;

    const headers = {
        'Content-Type': 'application/json',
        'X-Device-ID': deviceId,
        'X-Forwarded-For': `10.0.${Math.floor(__VU / 256)}.${__VU % 256}`,
        'User-Agent': 'Mozilla/5.0 (UniHub-K6-LoadTest/1.0)',
    };

    // 1. Send Presence Heartbeat (as Next.js usePresenceHeartbeat hook does)
    const presenceRes = http.get(`${BASE_URL}/api/v1/workshops/${WORKSHOP_ID}/presence`, {
        headers: headers,
        tags: { name: 'HeartbeatPresence' },
    });

    check(presenceRes, {
        'presence status is 200': (r) => r.status === 200,
    });

    // 2. Realistic user interaction: 50% browse workshops list, 50% view detail
    if (Math.random() < 0.5) {
        const listRes = http.get(`${BASE_URL}/api/v1/workshops`, {
            headers: headers,
            tags: { name: 'GetWorkshops' },
        });
        check(listRes, {
            'workshops list status is 200': (r) => r.status === 200,
        });
    }

    // Realistic think time between heartbeats (simulate 2-4 seconds between user actions)
    sleep(Math.random() * 2 + 2);
}

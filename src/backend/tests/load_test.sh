#!/bin/bash
# shellcheck disable=SC2059
# ============================================================================
# UniHub Workshop — Load Test Script
# Mô phỏng 12,000 sinh viên đăng ký workshop đồng thời
# ============================================================================
#
# USAGE:
#   ./tests/load_test.sh [phase]
#
# PHASES:
#   all       - Chạy tất cả (default)
#   generate  - Tạo file CSV 12k sinh viên
#   import    - Import CSV vào DB
#   loadtest  - Chạy load test đăng ký workshop
#   report    - Xem báo cáo kết quả
#
# EXAMPLES:
#   ./tests/load_test.sh              # Chạy tất cả
#   ./tests/load_test.sh generate     # Chỉ tạo CSV
#   ./tests/load_test.sh import       # Chỉ import
#   ./tests/load_test.sh loadtest     # Chỉ load test
# ============================================================================

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
CONCURRENCY="${CONCURRENCY:-5000}"       # Số request song song
TOTAL_STUDENTS="${TOTAL_STUDENTS:-12000}"
RESULTS_DIR="./data/loadtest_results"
CSV_FILE="./data/students_12k.csv"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()  { echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; echo -e "${BLUE}  $*${NC}"; echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

# ============================================================================
# Phase 0: Health Check
# ============================================================================
health_check() {
    log_step "🏥 Phase 0: Health Check"
    
    if ! curl -sf "${BASE_URL}/health" > /dev/null 2>&1; then
        log_error "Server is not running at ${BASE_URL}"
        log_info  "Please start the server first: go run cmd/server/main.go"
        exit 1
    fi
    log_ok "Server is healthy at ${BASE_URL}"
}

# ============================================================================
# Phase 1: Generate CSV
# ============================================================================
phase_generate() {
    log_step "📝 Phase 1: Generate 12,000 Student CSV"
    
    if [ -f "$CSV_FILE" ]; then
        EXISTING_COUNT=$(wc -l < "$CSV_FILE")
        EXISTING_COUNT=$((EXISTING_COUNT - 1))  # Minus header
        log_ok "CSV already exists with ${EXISTING_COUNT} records, skipping generation"
        return
    fi
    
    python3 tests/generate_12k_students.py
    log_ok "CSV generated: $(wc -l < "$CSV_FILE") lines"
}

# ============================================================================
# Phase 2: Import Students via CSV API
# ============================================================================
phase_import() {
    log_step "📦 Phase 2: Import Students to Database"
    
    if [ ! -f "$CSV_FILE" ]; then
        log_error "CSV file not found: $CSV_FILE"
        log_info  "Run: ./tests/load_test.sh generate"
        exit 1
    fi
    
    # Login as admin
    log_info "Logging in as admin..."
    LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"student_id":"admin","password":"123456"}')
    
    TOKEN=$(echo "$LOGIN_RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])" 2>/dev/null || true)
    
    if [ -z "$TOKEN" ]; then
        log_error "Failed to login as admin"
        log_info  "Response: $LOGIN_RESPONSE"
        exit 1
    fi
    log_ok "Admin login successful"
    
    # Upload CSV
    log_info "Uploading CSV ($(du -h "$CSV_FILE" | cut -f1))..."
    IMPORT_START=$(date +%s)
    
    IMPORT_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/admin/import/csv" \
        -H "Authorization: Bearer $TOKEN" \
        -F "file=@${CSV_FILE}" \
        --max-time 600)
    
    IMPORT_END=$(date +%s)
    IMPORT_DURATION=$((IMPORT_END - IMPORT_START))
    
    log_ok "Import completed in ${IMPORT_DURATION}s"
    echo "$IMPORT_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$IMPORT_RESPONSE"
    
    # Save admin token for later use
    echo "$TOKEN" > "${RESULTS_DIR}/admin_token.txt"
}

# ============================================================================
# Phase 3: Load Test - Concurrent Workshop Registration
# ============================================================================
phase_loadtest() {
    log_step "🔥 Phase 3: Load Test — ${TOTAL_STUDENTS} Students × ${CONCURRENCY} Concurrent"
    
    mkdir -p "$RESULTS_DIR"
    
    # Auto-reset: clean Redis + DB before each test
    log_info "Resetting Redis & DB for clean test..."
    docker exec unihub-redis redis-cli FLUSHALL > /dev/null 2>&1 || true
    PGPASSWORD=unihub_secret psql -h localhost -p 5433 -U unihub -d unihub_workshop \
        -c "DELETE FROM notifications; DELETE FROM payments; DELETE FROM registrations; UPDATE workshops SET available_seats = capacity;" > /dev/null 2>&1 || true
    log_ok "Clean state ready"
    
    # Get available workshops
    log_info "Fetching available workshops..."
    WORKSHOPS=$(curl -sf "${BASE_URL}/api/v1/workshops")
    WORKSHOP_ID=$(echo "$WORKSHOPS" | python3 -c "
import sys, json
data = json.load(sys.stdin)
workshops = data.get('data', data) if isinstance(data, dict) else data
if isinstance(workshops, list) and len(workshops) > 0:
    # Pick the workshop with most available seats
    best = max(workshops, key=lambda w: w.get('available_seats', 0))
    print(best['id'])
else:
    print('')
" 2>/dev/null || true)
    
    if [ -z "$WORKSHOP_ID" ]; then
        log_error "No workshops found. Please check the database."
        exit 1
    fi
    
    WORKSHOP_TITLE=$(echo "$WORKSHOPS" | python3 -c "
import sys, json
data = json.load(sys.stdin)
workshops = data.get('data', data) if isinstance(data, dict) else data
for w in workshops:
    if w['id'] == '$WORKSHOP_ID':
        print(f\"{w['title']} (seats: {w.get('available_seats', '?')}/{w.get('capacity', '?')})\")
        break
" 2>/dev/null || true)
    log_ok "Target workshop: $WORKSHOP_TITLE"
    log_ok "Workshop ID: $WORKSHOP_ID"
    
    # ── Sub-phase 3a: Mass Login (get JWT tokens) ──
    log_info ""
    log_info "── Phase 3a: Mass Login (batch JWT tokens) ──"
    
    TOKEN_FILE="${RESULTS_DIR}/tokens.txt"
    > "$TOKEN_FILE"
    
    START_ID=21127100
    END_ID=$((START_ID + TOTAL_STUDENTS - 1))
    BATCH_SIZE=500
    LOGIN_SUCCESS=0
    LOGIN_FAIL=0
    
    LOGIN_START=$(date +%s%N)
    
    for ((batch_start=START_ID; batch_start<=END_ID; batch_start+=BATCH_SIZE)); do
        batch_end=$((batch_start + BATCH_SIZE - 1))
        if [ $batch_end -gt $END_ID ]; then
            batch_end=$END_ID
        fi
        
        PROGRESS=$(( (batch_start - START_ID) * 100 / TOTAL_STUDENTS ))
        printf "\r  🔑 Login progress: %d%% [%d/%d]" "$PROGRESS" "$((batch_start - START_ID))" "$TOTAL_STUDENTS"
        
        # Parallel login within batch
        for ((sid=batch_start; sid<=batch_end; sid++)); do
            (
                RESP=$(curl -s -X POST "${BASE_URL}/api/v1/auth/login" \
                    -H "Content-Type: application/json" \
                    -d "{\"student_id\":\"${sid}\",\"password\":\"123456\"}" \
                    --max-time 10 2>/dev/null || echo "FAIL")
                
                TK=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])" 2>/dev/null || true)
                
                if [ -n "$TK" ]; then
                    echo "$TK" >> "$TOKEN_FILE"
                fi
            ) &
            
            # Throttle: max CONCURRENCY parallel jobs
            if (( (sid - batch_start + 1) % CONCURRENCY == 0 )); then
                wait
            fi
        done
        wait
    done
    
    LOGIN_END=$(date +%s%N)
    LOGIN_DURATION=$(( (LOGIN_END - LOGIN_START) / 1000000 ))
    LOGIN_SUCCESS=$(wc -l < "$TOKEN_FILE")
    LOGIN_FAIL=$((TOTAL_STUDENTS - LOGIN_SUCCESS))
    
    printf "\r  🔑 Login complete!                                    \n"
    log_ok "Tokens acquired: ${LOGIN_SUCCESS}/${TOTAL_STUDENTS} (${LOGIN_FAIL} failed) in ${LOGIN_DURATION}ms"
    
    if [ "$LOGIN_SUCCESS" -lt 10 ]; then
        log_error "Too few successful logins. Did you import students first?"
        log_info  "Run: ./tests/load_test.sh import"
        exit 1
    fi
    
    # ── Sub-phase 3b: Concurrent Registration with Waiting Room Retry ──
    log_info ""
    log_info "── Phase 3b: Registration Storm (with Waiting Room retry) ──"
    log_info "Each student: POST → if 429 → poll waiting room → retry when GRANTED"
    log_info "Concurrency: ${CONCURRENCY}, Timeout per student: 60s"
    
    REG_RESULTS="${RESULTS_DIR}/registration_results.txt"
    REG_LATENCIES="${RESULTS_DIR}/latencies.txt"
    > "$REG_RESULTS"
    > "$REG_LATENCIES"
    
    # Create standalone worker script (avoids set -e inheritance issues)
    WORKER_SCRIPT="${RESULTS_DIR}/_worker.sh"
    cat > "$WORKER_SCRIPT" << 'WORKER_EOF'
#!/bin/bash
# Worker: handles full registration lifecycle for one student
# Args: $1=token $2=base_url $3=workshop_id $4=results_file $5=latencies_file
TOKEN="$1"; BASE_URL="$2"; WORKSHOP_ID="$3"; RESULTS="$4"; LATENCIES="$5"
MAX_ATTEMPTS=20; POLL_INTERVAL=3; ATTEMPT=0
START_NS=$(date +%s%N)

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    
    RESP=$(curl -s -w "\n%{http_code}" \
        -X POST "${BASE_URL}/api/v1/registrations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"workshop_id\":\"${WORKSHOP_ID}\"}" \
        --max-time 15 2>/dev/null || echo -e "\n000")
    
    CODE=$(echo "$RESP" | tail -1)
    
    case "$CODE" in
        202|200)
            END_NS=$(date +%s%N)
            MS=$(( (END_NS - START_NS) / 1000000 ))
            echo "${CODE},${MS}" >> "$RESULTS"
            echo "$MS" >> "$LATENCIES"
            exit 0
            ;;
        400|409)
            END_NS=$(date +%s%N)
            MS=$(( (END_NS - START_NS) / 1000000 ))
            echo "400,${MS}" >> "$RESULTS"
            echo "$MS" >> "$LATENCIES"
            exit 0
            ;;
        429)
            # Poll waiting room until GRANTED
            while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
                sleep $POLL_INTERVAL
                ATTEMPT=$((ATTEMPT + 1))
                
                POLL=$(curl -s \
                    "${BASE_URL}/api/v1/registrations/waiting-room/${WORKSHOP_ID}" \
                    -H "Authorization: Bearer $TOKEN" \
                    --max-time 10 2>/dev/null || echo "")
                
                STATUS=$(echo "$POLL" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print(d.get('data',{}).get('status_text','UNKNOWN'))
except:
    print('ERROR')
" 2>/dev/null || echo "ERROR")
                
                if [ "$STATUS" = "GRANTED" ] || [ "$STATUS" = "ALREADY_ACTIVE" ]; then
                    break  # Retry registration
                fi
            done
            ;;
        *)
            sleep 1
            ;;
    esac
done

# Timeout
END_NS=$(date +%s%N)
MS=$(( (END_NS - START_NS) / 1000000 ))
echo "TIMEOUT,${MS}" >> "$RESULTS"
echo "$MS" >> "$LATENCIES"
exit 0
WORKER_EOF
    chmod +x "$WORKER_SCRIPT"
    
    REG_START=$(date +%s%N)
    REG_TOTAL=$(wc -l < "$TOKEN_FILE")
    
    # Progress tracking in background
    (
        while true; do
            sleep 2
            DONE=$(wc -l < "$REG_RESULTS" 2>/dev/null || echo 0)
            if [ "$DONE" -ge "$REG_TOTAL" ] 2>/dev/null; then break; fi
            PCT=$((DONE * 100 / REG_TOTAL))
            printf "\r  🚀 Registration progress: %d%% [%d/%d]" "$PCT" "$DONE" "$REG_TOTAL"
        done
    ) &
    PROGRESS_PID=$!
    
    # Run workers using xargs for clean parallel execution
    cat "$TOKEN_FILE" | xargs -P "$CONCURRENCY" -I {} \
        bash "$WORKER_SCRIPT" {} "$BASE_URL" "$WORKSHOP_ID" "$REG_RESULTS" "$REG_LATENCIES"
    
    # Stop progress tracker
    kill $PROGRESS_PID 2>/dev/null || true
    wait $PROGRESS_PID 2>/dev/null || true
    
    # Cleanup worker script
    rm -f "$WORKER_SCRIPT"
    
    REG_END=$(date +%s%N)
    REG_DURATION=$(( (REG_END - REG_START) / 1000000 ))
    
    printf "\r  🚀 Registration complete!                              \n"
    log_ok "All ${REG_TOTAL} requests processed in ${REG_DURATION}ms"
    
    # ── Generate report ──
    phase_report
}

# ============================================================================
# Phase 4: Report
# ============================================================================
phase_report() {
    log_step "📊 Phase 4: Load Test Report"
    
    REG_RESULTS="${RESULTS_DIR}/registration_results.txt"
    REG_LATENCIES="${RESULTS_DIR}/latencies.txt"
    
    if [ ! -f "$REG_RESULTS" ]; then
        log_error "No results found. Run load test first."
        exit 1
    fi
    
    python3 -c "
import sys
results_file = '${REG_RESULTS}'
latencies_file = '${REG_LATENCIES}'

status_counts = {}
latencies = []

with open(results_file) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        parts = line.split(',')
        code = parts[0]
        lat = int(parts[1]) if len(parts) > 1 else 0
        status_counts[code] = status_counts.get(code, 0) + 1
        latencies.append(lat)

total = sum(status_counts.values())
latencies.sort()

print()
print('┌─────────────────────────────────────────────────────┐')
print('│         📊 LOAD TEST RESULTS SUMMARY                │')
print('├─────────────────────────────────────────────────────┤')
print(f'│  Total Requests:     {total:>10}                     │')
print(f'│  HTTP 202 (Queued):  {status_counts.get(\"202\", 0):>10}                     │')
print(f'│  HTTP 200 (OK):      {status_counts.get(\"200\", 0):>10}                     │')
print(f'│  HTTP 400 (Reject):  {status_counts.get(\"400\", 0):>10}                     │')
print(f'│  HTTP 429 (Rate):    {status_counts.get(\"429\", 0):>10}                     │')
print(f'│  HTTP 500 (Error):   {status_counts.get(\"500\", 0):>10}                     │')
print(f'│  Timeout/Fail:       {status_counts.get(\"000\", 0) + status_counts.get(\"TIMEOUT\", 0):>10}                     │')
print('├─────────────────────────────────────────────────────┤')

if latencies:
    avg = sum(latencies) / len(latencies)
    p50 = latencies[len(latencies) // 2]
    p95 = latencies[int(len(latencies) * 0.95)]
    p99 = latencies[int(len(latencies) * 0.99)]
    max_lat = latencies[-1]
    min_lat = latencies[0]
    
    print(f'│  Latency (avg):      {avg:>8.0f}ms                     │')
    print(f'│  Latency (p50):      {p50:>8}ms                     │')
    print(f'│  Latency (p95):      {p95:>8}ms                     │')
    print(f'│  Latency (p99):      {p99:>8}ms                     │')
    print(f'│  Latency (min):      {min_lat:>8}ms                     │')
    print(f'│  Latency (max):      {max_lat:>8}ms                     │')
    
    total_sec = max_lat / 1000 if max_lat > 0 else 1
    rps = total / total_sec if total_sec > 0 else 0
    print(f'│  Throughput:         {rps:>8.0f} req/s                  │')

print('├─────────────────────────────────────────────────────┤')
print('│  HTTP Status Breakdown:                             │')
for code, count in sorted(status_counts.items()):
    pct = count * 100 / total if total > 0 else 0
    bar = '█' * int(pct / 2)
    label = {
        '200': 'OK',
        '202': 'Accepted (queued)',
        '400': 'Bad Request',
        '401': 'Unauthorized',
        '429': 'Rate Limited',
        '500': 'Server Error',
        '000': 'Timeout/Conn Fail'
    }.get(code, f'HTTP {code}')
    print(f'│  {code} {label:<20s} {count:>6} ({pct:5.1f}%) {bar:<10}│')
print('└─────────────────────────────────────────────────────┘')
print()
print('📌 Key observations:')
print('   • HTTP 202 = Registration queued in RabbitMQ (success)')
print('   • HTTP 400 = Workshop full / already registered / seat lock failed')
print('   • HTTP 429 = Rate limiter (Token Bucket) kicked in')
print('   • HTTP 000 = Connection refused/timeout')
print()
"
    
    log_ok "Results saved to: ${RESULTS_DIR}/"
}

# ============================================================================
# Main
# ============================================================================
main() {
    PHASE="${1:-all}"
    
    echo -e "${CYAN}"
    echo "  ╔═══════════════════════════════════════════════════╗"
    echo "  ║   UniHub Workshop — Load Test (12,000 Students)   ║"
    echo "  ╚═══════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    mkdir -p "$RESULTS_DIR"
    
    case "$PHASE" in
        generate)
            phase_generate
            ;;
        import)
            health_check
            phase_import
            ;;
        loadtest)
            health_check
            phase_loadtest
            ;;
        report)
            phase_report
            ;;
        all)
            health_check
            phase_generate
            phase_import
            phase_loadtest
            ;;
        *)
            echo "Usage: $0 {all|generate|import|loadtest|report}"
            exit 1
            ;;
    esac
    
    echo ""
    log_ok "Done! 🎉"
}

main "$@"

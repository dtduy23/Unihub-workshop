#!/bin/bash
# ============================================================================
# UniHub Workshop — One-click Setup
# Khởi tạo Docker + DB + Seed + Import 12k students
# Usage: ./tests/setup.sh
# ============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

# DB config (matching docker-compose.yml)
DB_HOST=localhost
DB_PORT=5433
DB_USER=unihub
DB_PASSWORD=unihub_secret
DB_NAME=unihub_workshop

# Bcrypt hash of "123456" (pre-computed)
BCRYPT_HASH='$2a$10$x6.ouOf.LBM3OogQNHcKZe/De3W9uivpBZ2Ty1mt3SQxBXq9Jb2JG'

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

run_sql() {
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -q "$@" 2>/dev/null
}

echo -e "${CYAN}"
echo "  ╔════════════════════════════════════════════════╗"
echo "  ║   UniHub Workshop — One-click Setup            ║"
echo "  ╚════════════════════════════════════════════════╝"
echo -e "${NC}"

# ── Step 1: Docker Compose ──
echo -e "\n${CYAN}━━━ Step 1: Docker Compose ━━━${NC}"

if ! command -v docker &>/dev/null; then
    log_error "Docker not installed"
    exit 1
fi

docker compose up -d
log_ok "Containers started"

# ── Step 2: Wait for PostgreSQL ──
echo -e "\n${CYAN}━━━ Step 2: Waiting for PostgreSQL ━━━${NC}"

for i in $(seq 1 30); do
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" &>/dev/null; then
        log_ok "PostgreSQL ready (${i}s)"
        break
    fi
    printf "\r  ⏳ Waiting... (%ds)" "$i"
    sleep 1
done

if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" &>/dev/null; then
    log_error "PostgreSQL not ready after 30s"
    exit 1
fi

# ── Step 3: Schema migration ──
echo -e "\n${CYAN}━━━ Step 3: Schema Check ━━━${NC}"

# Check if room_layout_url column exists, add if missing
HAS_LAYOUT=$(run_sql -t -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='workshops' AND column_name='room_layout_url';")
if [ "$(echo "$HAS_LAYOUT" | tr -d ' ')" = "0" ]; then
    run_sql -c "ALTER TABLE workshops ADD COLUMN room_layout_url character varying;"
    log_ok "Added missing column: room_layout_url"
else
    log_ok "Schema is up to date"
fi

# ── Step 4: Seed base data ──
echo -e "\n${CYAN}━━━ Step 4: Seed Data ━━━${NC}"

USER_COUNT=$(run_sql -t -c "SELECT COUNT(*) FROM users;" | tr -d ' ')
WORKSHOP_COUNT=$(run_sql -t -c "SELECT COUNT(*) FROM workshops;" | tr -d ' ')

if [ "$USER_COUNT" -gt "0" ] && [ "$WORKSHOP_COUNT" -gt "0" ]; then
    log_ok "Already seeded: ${USER_COUNT} users, ${WORKSHOP_COUNT} workshops"
else
    run_sql -c "
    INSERT INTO users (user_id, password_hash, full_name, email, phone, role) VALUES
    ('admin',    '${BCRYPT_HASH}', 'System Administrator', 'admin@unihub.vn',          '0909999001', 'ADMIN'),
    ('staff01',  '${BCRYPT_HASH}', 'Trần Staff',           'staff01@unihub.vn',         '0909999002', 'STAFF'),
    ('staff02',  '${BCRYPT_HASH}', 'Lê Staff',             'staff02@unihub.vn',         '0909999003', 'STAFF'),
    ('21127001', '${BCRYPT_HASH}', 'Nguyễn Văn An',        'an.nguyen@student.edu.vn',  '0901234001', 'STUDENT'),
    ('21127002', '${BCRYPT_HASH}', 'Trần Thị Bình',        'binh.tran@student.edu.vn',  '0901234002', 'STUDENT'),
    ('21127003', '${BCRYPT_HASH}', 'Lê Hoàng Cường',       'cuong.le@student.edu.vn',   '0901234003', 'STUDENT'),
    ('21127004', '${BCRYPT_HASH}', 'Phạm Minh Duy',        'duy.pham@student.edu.vn',   '0901234004', 'STUDENT'),
    ('21127005', '${BCRYPT_HASH}', 'Hoàng Thị Em',         'em.hoang@student.edu.vn',   '0901234005', 'STUDENT')
    ON CONFLICT (user_id) DO NOTHING;

    INSERT INTO workshops (title, description, speaker, room, start_time, end_time, capacity, available_seats, price, status) VALUES
    ('Kỹ năng CV và Phỏng vấn',        'Workshop hướng dẫn viết CV chuyên nghiệp',       'ThS. Nguyễn Minh Tuấn',   'A101', NOW() + interval '7 day',  NOW() + interval '7 day 2 hour',  200,   200,   0,      'PUBLISHED'),
    ('Lập trình Python cho người mới',  'Khóa học cơ bản về Python',                      'TS. Trần Văn Hùng',       'B201', NOW() + interval '8 day',  NOW() + interval '8 day 3 hour',  5000,  5000,  50000,  'PUBLISHED'),
    ('Design Thinking Workshop',        'Phương pháp tư duy thiết kế',                    'MBA. Lê Thu Hà',          'C301', NOW() + interval '9 day',  NOW() + interval '9 day 3 hour',  50,    50,    0,      'PUBLISHED'),
    ('Cloud Computing with AWS',        'Thực hành triển khai trên AWS',                   'AWS SA Trần Minh',        'B201', NOW() + interval '10 day', NOW() + interval '10 day 4 hour', 10000, 10000, 0,      'PUBLISHED')
    ON CONFLICT DO NOTHING;
    "
    USER_COUNT=$(run_sql -t -c "SELECT COUNT(*) FROM users;" | tr -d ' ')
    WORKSHOP_COUNT=$(run_sql -t -c "SELECT COUNT(*) FROM workshops;" | tr -d ' ')
    log_ok "Seeded: ${USER_COUNT} users, ${WORKSHOP_COUNT} workshops"
fi

# ── Step 5: Import 12k students ──
echo -e "\n${CYAN}━━━ Step 5: Import 12,000 Students ━━━${NC}"

STUDENT_COUNT=$(run_sql -t -c "SELECT COUNT(*) FROM users WHERE role='STUDENT';" | tr -d ' ')

if [ "$STUDENT_COUNT" -ge "12000" ]; then
    log_ok "Already imported: ${STUDENT_COUNT} students"
else
    # Generate CSV if needed
    if [ ! -f "data/students_12k.csv" ]; then
        log_info "Generating CSV..."
        python3 tests/generate_12k_students.py
    fi

    # Import via psql (fast)
    log_info "Importing via direct SQL..."
    python3 tests/direct_import_12k.py

    STUDENT_COUNT=$(run_sql -t -c "SELECT COUNT(*) FROM users WHERE role='STUDENT';" | tr -d ' ')
    log_ok "Total students: ${STUDENT_COUNT}"
fi

# ── Step 6: Summary ──
echo -e "\n${CYAN}━━━ Setup Complete ━━━${NC}"

TOTAL_USERS=$(run_sql -t -c "SELECT COUNT(*) FROM users;" | tr -d ' ')
TOTAL_WORKSHOPS=$(run_sql -t -c "SELECT COUNT(*) FROM workshops;" | tr -d ' ')

echo ""
echo "  📊 Database Summary:"
echo "     Users:     ${TOTAL_USERS} (${STUDENT_COUNT} students)"
echo "     Workshops: ${TOTAL_WORKSHOPS}"
echo "     Password:  123456 (bcrypt hashed)"
echo ""
echo "  🔑 Accounts:"
echo "     Admin:   admin / 123456"
echo "     Staff:   staff01 / 123456"
echo "     Student: 21127001-21139099 / 123456"
echo ""
echo "  🚀 Next steps:"
echo "     go run cmd/server/main.go"
echo "     ./tests/load_test.sh loadtest"
echo ""

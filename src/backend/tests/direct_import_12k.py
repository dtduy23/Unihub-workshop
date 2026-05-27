#!/usr/bin/env python3
"""
Generate SQL INSERT statements for 12,000 students and execute via psql.
Much faster than HTTP API (no bcrypt overhead, bulk INSERT).

Usage: python3 scripts/direct_import_12k.py
"""

import csv
import os
import subprocess
import sys
import time


def load_env():
    """Load .env file."""
    env = {}
    env_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '.env')
    with open(env_path) as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith('#') or '=' not in line:
                continue
            key, _, value = line.partition('=')
            value = value.strip()
            # Handle multi-line values (like RSA keys)
            if value.startswith('"') and not value.endswith('"'):
                continue
            value = value.strip('"').strip("'")
            env[key.strip()] = value
    return env


def escape_sql(s):
    """Escape single quotes for SQL."""
    return s.replace("'", "''")


def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_dir = os.path.dirname(script_dir)
    csv_path = os.path.join(project_dir, "data", "students_12k.csv")
    sql_path = os.path.join(project_dir, "data", "import_12k.sql")

    if not os.path.exists(csv_path):
        print("❌ CSV file not found. Run: python3 scripts/generate_12k_students.py")
        sys.exit(1)

    # Load DB config from .env
    env = load_env()

    # Read CSV
    print(f"📖 Reading {csv_path}...")
    students = []
    with open(csv_path, newline='', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        for row in reader:
            students.append(row)

    total = len(students)
    print(f"📦 Loaded {total} student records")

    # Pre-computed bcrypt hash of "123456" (same for all students)
    BCRYPT_HASH = "$2a$10$x6.ouOf.LBM3OogQNHcKZe/De3W9uivpBZ2Ty1mt3SQxBXq9Jb2JG"
    chunk_size = 500
    print(f"📝 Generating SQL (bcrypt hashed)...")

    with open(sql_path, 'w', encoding='utf-8') as sql_file:
        sql_file.write("-- Auto-generated: Import 12,000 students\n")
        sql_file.write("-- Password: 123456 (bcrypt hashed)\n")
        sql_file.write("BEGIN;\n\n")

        for i in range(0, total, chunk_size):
            chunk = students[i:i + chunk_size]
            values = []
            for s in chunk:
                sid = escape_sql(s['student_id'])
                name = escape_sql(s['full_name'])
                email = escape_sql(s['email'])
                phone = escape_sql(s['phone'])
                role = escape_sql(s['role'])
                values.append(f"  ('{sid}', '{BCRYPT_HASH}', '{name}', '{email}', '{phone}', '{role}')")

            sql_file.write(f"-- Chunk {i//chunk_size + 1} ({i+1}-{i+len(chunk)})\n")
            sql_file.write("INSERT INTO users (user_id, password_hash, full_name, email, phone, role) VALUES\n")
            sql_file.write(",\n".join(values))
            sql_file.write("\nON CONFLICT (user_id) DO UPDATE SET\n")
            sql_file.write("  password_hash = EXCLUDED.password_hash,\n")
            sql_file.write("  full_name = EXCLUDED.full_name,\n")
            sql_file.write("  email = EXCLUDED.email,\n")
            sql_file.write("  phone = EXCLUDED.phone,\n")
            sql_file.write("  role = EXCLUDED.role,\n")
            sql_file.write("  updated_at = CURRENT_TIMESTAMP;\n\n")

        sql_file.write("COMMIT;\n")

    sql_size = os.path.getsize(sql_path) / (1024 * 1024)
    print(f"✅ SQL generated: {sql_path} ({sql_size:.2f} MB)")

    # Execute via psql
    print(f"\n🚀 Importing into PostgreSQL...")
    start = time.time()

    psql_env = os.environ.copy()
    psql_env['PGPASSWORD'] = env.get('DB_PASSWORD', '')

    result = subprocess.run(
        [
            'psql',
            '-h', env.get('DB_HOST', 'localhost'),
            '-p', env.get('DB_PORT', '5432'),
            '-U', env.get('DB_USER', 'postgres'),
            '-d', env.get('DB_NAME', 'postgres'),
            '-f', sql_path,
            '-q',  # Quiet mode
        ],
        env=psql_env,
        capture_output=True,
        text=True,
    )

    elapsed = time.time() - start

    if result.returncode != 0:
        print(f"❌ Import failed!")
        print(f"stderr: {result.stderr[:500]}")
        sys.exit(1)

    print(f"\n{'='*60}")
    print(f"✅ Import completed in {elapsed:.1f}s ({total/elapsed:.0f} records/s)")
    print(f"{'='*60}")

    # Verify count
    verify = subprocess.run(
        [
            'psql',
            '-h', env.get('DB_HOST', 'localhost'),
            '-p', env.get('DB_PORT', '5432'),
            '-U', env.get('DB_USER', 'postgres'),
            '-d', env.get('DB_NAME', 'postgres'),
            '-t', '-c', "SELECT COUNT(*) FROM users WHERE role = 'STUDENT';",
        ],
        env=psql_env,
        capture_output=True,
        text=True,
    )

    if verify.returncode == 0:
        count = verify.stdout.strip()
        print(f"📊 Total students in DB: {count}")

    # Cleanup SQL file
    os.remove(sql_path)
    print(f"🧹 Cleaned up temporary SQL file")


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# dependencies = ["aiohttp"]
# ///
"""
Pre-login 12,000 sinh viên và lưu JWT tokens ra file JSON.
Dùng asyncio + aiohttp để login song song nhanh chóng.

USAGE:
    uv run tests/pre_login.py
    uv run tests/pre_login.py --base-url http://localhost:8080
    uv run tests/pre_login.py --base-url http://8.233.213.55 --students 12000
    uv run tests/pre_login.py --concurrency 50   # Giảm nếu server yếu

OUTPUT:
    data/tokens.json  — Mảng JWT tokens, k6 đọc trực tiếp
"""

import argparse
import asyncio
import json
import time

import aiohttp


async def login_student(session: aiohttp.ClientSession, base_url: str, student_id: str, semaphore: asyncio.Semaphore):
    """Login 1 sinh viên, trả về token hoặc None."""
    url = f"{base_url}/api/v1/auth/login"
    payload = {"student_id": student_id, "password": "123456"}

    async with semaphore:
        try:
            async with session.post(url, json=payload, timeout=aiohttp.ClientTimeout(total=30)) as resp:
                if resp.status == 200:
                    data = await resp.json()
                    token = data.get("data", {}).get("token")
                    return token
                return None
        except Exception:
            return None


async def main():
    parser = argparse.ArgumentParser(description="Pre-login students and cache JWT tokens")
    parser.add_argument("--base-url", default="http://8.233.213.55", help="Server URL")
    parser.add_argument("--students", type=int, default=12000, help="Number of students")
    parser.add_argument("--start-id", type=int, default=31127100, help="Starting student ID")
    parser.add_argument("--concurrency", type=int, default=100, help="Max concurrent logins")
    parser.add_argument("--output", default="data/tokens.json", help="Output file path")
    args = parser.parse_args()

    print(f"\n🚀 UniHub Pre-Login Tool")
    print(f"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    print(f"📍 Server:      {args.base_url}")
    print(f"👥 Students:    {args.students} (ID: {args.start_id} → {args.start_id + args.students - 1})")
    print(f"⚡ Concurrency: {args.concurrency}")
    print(f"📁 Output:      {args.output}")
    print(f"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

    # Health check
    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(f"{args.base_url}/health", timeout=aiohttp.ClientTimeout(total=5)) as resp:
                if resp.status != 200:
                    print(f"❌ Health check failed: HTTP {resp.status}")
                    return
            print("✅ Health check passed\n")
        except Exception as e:
            print(f"❌ Cannot connect to server: {e}")
            return

    semaphore = asyncio.Semaphore(args.concurrency)
    tokens = []
    success = 0
    fail = 0
    start_time = time.time()

    connector = aiohttp.TCPConnector(limit=args.concurrency, limit_per_host=args.concurrency)
    async with aiohttp.ClientSession(connector=connector) as session:
        # Chia thành batch 1000 để hiển thị progress
        batch_size = 1000
        for batch_start in range(0, args.students, batch_size):
            batch_end = min(batch_start + batch_size, args.students)
            student_ids = [str(args.start_id + i) for i in range(batch_start, batch_end)]

            tasks = [login_student(session, args.base_url, sid, semaphore) for sid in student_ids]
            results = await asyncio.gather(*tasks)

            for token in results:
                if token:
                    tokens.append(token)
                    success += 1
                else:
                    fail += 1

            elapsed = time.time() - start_time
            pct = batch_end * 100 // args.students
            rps = success / elapsed if elapsed > 0 else 0
            print(f"   🔑 {pct:3d}% | {success:>6} OK | {fail:>5} fail | {rps:.0f} req/s | {elapsed:.1f}s")

    elapsed = time.time() - start_time

    # Lưu tokens ra JSON
    import os
    os.makedirs(os.path.dirname(args.output) or ".", exist_ok=True)
    with open(args.output, "w") as f:
        json.dump(tokens, f)

    file_size = os.path.getsize(args.output) / 1024 / 1024  # MB

    print(f"\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    print(f"✅ Pre-login complete!")
    print(f"   Tokens saved: {len(tokens)}/{args.students}")
    print(f"   Failed:       {fail}")
    print(f"   Duration:     {elapsed:.1f}s ({success / elapsed:.0f} logins/s)")
    print(f"   File:         {args.output} ({file_size:.1f} MB)")
    print(f"   JWT valid:    1 hour from now")
    print(f"\n💡 Chạy k6 load test:")
    print(f"   k6 run src/backend/tests/k6_load_test.js --env PHASE=burst")
    print(f"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")


if __name__ == "__main__":
    asyncio.run(main())

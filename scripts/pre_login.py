#!/usr/bin/env python3
"""Login generated students once; save sessions without printing JWTs."""
import argparse
import csv
from concurrent.futures import ThreadPoolExecutor
import json
import os
from pathlib import Path
from urllib.request import Request, urlopen

def login(base_url, row, opener=urlopen):
    payload = json.dumps({"student_id": row["student_id"], "password": row["password"]}).encode()
    request = Request(base_url.rstrip("/") + "/api/v1/auth/login", data=payload, headers={"Content-Type": "application/json"})
    with opener(request, timeout=20) as response:
        result = json.load(response)
    token = result.get("data", {}).get("token")
    if not result.get("success") or not isinstance(token, str) or not token:
        raise ValueError(f"Login failed for {row['student_id']}")
    return {"student_id": row["student_id"], "token": token}

def prepare(base_url, csv_path, output, workers=4):
    with Path(csv_path).open(encoding="utf-8", newline="") as stream:
        users = list(csv.DictReader(stream))
    if not users:
        raise ValueError("CSV has no students")
    with ThreadPoolExecutor(max_workers=workers) as executor:
        sessions = list(executor.map(lambda row: login(base_url, row), users))
    output = Path(output)
    output.parent.mkdir(parents=True, exist_ok=True)
    with output.open("w", encoding="utf-8") as stream:
        os.chmod(output, 0o600)
        json.dump({"sessions": sessions}, stream)
    return len(sessions)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--csv", required=True)
    parser.add_argument("--output", default="/tmp/unihub-sessions.json")
    parser.add_argument("--workers", type=int, default=4)
    args = parser.parse_args()
    if not 1 <= args.workers <= 32:
        parser.error("workers must be 1..32")
    print(f"Saved {prepare(args.base_url, args.csv, args.output, args.workers)} sessions")
if __name__ == "__main__":
    main()

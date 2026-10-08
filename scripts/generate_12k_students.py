#!/usr/bin/env python3
"""Generate deterministic demo CSV matching UniHub's student importer."""
import argparse
import csv
import os
from pathlib import Path
FIELDS = ["student_id", "password", "full_name", "email", "phone", "role"]
def rows(count=12000, password="Demo123456!"):
    if not 1 <= count <= 12000:
        raise ValueError("count must be between 1 and 12000")
    for number in range(1, count + 1):
        uid = f"demo{number:05d}"
        yield [uid, password, f"Demo Student {number}", f"{uid}@example.test", f"090{number:07d}", "STUDENT"]
def generate(path, count=12000, password="Demo123456!"):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as stream:
        os.chmod(path, 0o600)
        writer = csv.writer(stream)
        writer.writerow(FIELDS)
        writer.writerows(rows(count, password))
def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", default="/tmp/unihub-students.csv")
    parser.add_argument("--count", type=int, default=12000)
    parser.add_argument("--password", default=os.getenv("DEMO_PASSWORD", "Demo123456!"))
    args = parser.parse_args()
    generate(args.output, args.count, args.password)
    print(f"Generated {args.count} students in {args.output}")
if __name__ == "__main__":
    main()

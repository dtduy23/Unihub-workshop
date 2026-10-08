import csv
import io
import json
from pathlib import Path
import sys
import tempfile
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import generate_12k_students as generator
import pre_login

class DemoToolsTest(unittest.TestCase):
    def test_12k_csv_has_unique_importable_records(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "students.csv"
            generator.generate(path)
            with path.open(newline="", encoding="utf-8") as stream:
                records = list(csv.DictReader(stream))
            self.assertEqual(len(records), 12000)
            self.assertEqual(len({r["student_id"] for r in records}), 12000)
            self.assertEqual(len({r["email"] for r in records}), 12000)
            self.assertEqual(list(records[0]), generator.FIELDS)
            self.assertTrue(all(r["role"] == "STUDENT" and len(r["phone"]) == 10 for r in records))
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)
    def test_rejects_invalid_counts(self):
        for count in (0, -1, 12001):
            with self.assertRaises(ValueError):
                list(generator.rows(count))
    def test_login_uses_contract_and_rejects_missing_token(self):
        row = {"student_id": "demo00001", "password": "Demo123456!"}
        def opener(request, timeout):
            self.assertEqual(request.full_url, "http://api/api/v1/auth/login")
            self.assertEqual(json.loads(request.data), row)
            return io.StringIO('{"success":true,"data":{"token":"test-jwt"}}')
        self.assertEqual(pre_login.login("http://api/", row, opener)["token"], "test-jwt")
        with self.assertRaises(ValueError):
            pre_login.login("http://api", row, lambda *a, **k: io.StringIO('{"success":false}'))
if __name__ == "__main__":
    unittest.main()

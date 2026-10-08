from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "devops"))
from promote_images import image_values
class PromotionTest(unittest.TestCase):
    def test_shared_immutable_backend_tag(self):
        sha = "a" * 40
        values = image_values("registry.example/team", sha)
        self.assertEqual(values.count(sha), 3)
        self.assertEqual(values.count("registry.example/team/unihub-backend"), 2)
    def test_rejects_mutable_refs_and_yaml_injection(self):
        for sha in ["main", "latest", "a" * 39, "a" * 40 + "\n"]:
            with self.assertRaises(ValueError):
                image_values("registry.example/team", sha)
        with self.assertRaises(ValueError):
            image_values("repo\nmalicious: true", "b" * 40)

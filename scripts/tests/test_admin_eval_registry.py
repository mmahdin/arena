"""BO-08: the embedded unreviewed golden manifest must match source bytes.
This is a source fingerprint check, NOT proof that any evaluator executed them.
"""
import hashlib
import json
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]

class GoldenRegistryTest(unittest.TestCase):
    def test_manifest_matches_all_28_unreviewed_fixture_files(self):
        registry = json.loads((ROOT / "apps/api/internal/modules/admin/domain/golden_registry.json").read_text())
        files = sorted((ROOT / "knowledge/evals/golden").glob("*.yaml"))
        self.assertEqual(len(files), 28)
        self.assertEqual([x["id"] for x in registry], [f.stem for f in files])
        for row, file in zip(registry, files):
            data = file.read_bytes()
            self.assertEqual(row["sha256"], hashlib.sha256(data).hexdigest())
            self.assertRegex(data.decode(), r"(?m)^synthetic_only: true$")
            self.assertRegex(data.decode(), rf"(?m)^id: {re.escape(row['id'])}$")
            languages = re.findall(r"(?m)^- language_([a-z]{2})\s*$", data.decode())
            self.assertEqual(row["language"], languages[0] if len(languages) == 1 else "unmarked")

if __name__ == "__main__":
    unittest.main()

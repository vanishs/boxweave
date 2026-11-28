"""Synthetic identity calibration only; not native acceptance."""
import hashlib
import unittest
from pr191_n2_evidence_identity import evaluate

SOURCE = "a" * 40
BINARY = "b" * 64
LINES = [
    "N2_INSTALL_SOURCE=" + SOURCE,
    "N2_INSTALL_CANDIDATE_SHA256=" + BINARY,
    "N2_INSTALLED_CANDIDATE_SHA256=" + BINARY,
]


class Identity(unittest.TestCase):
    def check(self, lines, source=SOURCE):
        return evaluate(("\n".join(lines)+"\n").encode(), source)

    def test_exact_original(self):
        raw = ("\n".join(LINES)+"\n").encode()
        self.assertEqual(evaluate(raw, SOURCE), {
            "source": SOURCE, "candidate_sha256": BINARY,
            "transcript_sha256": hashlib.sha256(raw).hexdigest(),
        })

    def test_missing_duplicate_and_echo(self):
        for i in range(len(LINES)):
            with self.subTest(field=i):
                omitted = LINES[:i] + LINES[i+1:]
                for lines in (omitted, omitted + ["echo " + LINES[i]],
                              LINES + [LINES[i]]):
                    with self.assertRaises(ValueError):
                        self.check(lines)

    def test_malformed_and_mismatched(self):
        for lines in ([LINES[0].replace(SOURCE,"c"*40), *LINES[1:]],
                      [*LINES[:2], LINES[2].replace(BINARY,"d"*64)],
                      [LINES[0], "N2_INSTALL_CANDIDATE_SHA256=invalid", LINES[2]]):
            with self.assertRaises(ValueError):
                self.check(lines)
        with self.assertRaises(ValueError):
            self.check(LINES, "invalid")

    def test_no_acceptance_claim(self):
        result = self.check(LINES)
        self.assertEqual(set(result), {"source", "candidate_sha256", "transcript_sha256"})


if __name__ == "__main__":
    unittest.main()

"""Private oracle tests; no native fixture is executed by this suite."""
import unittest
from pr191_scoped_update_ack_wal import ack_proof


class AckOracle(unittest.TestCase):
    def check(self, phase, state, receipt, mark, events=None, tracking="tracking"):
        cp = {"update_id": "original", "tracking_id": "tracking", "box_id": "uuid"}
        def query(sql, args=()):
            if sql.startswith("SELECT id,"):
                return [("original", tracking, state, receipt, "uuid", "uuid")]
            if sql.startswith("SELECT state,"):
                return [(x, "") for x in (events or ["ADMITTED", "DISPATCHING"] +
                                         (["VERIFYING"] if state == "VERIFYING" else []))]
            if sql.startswith("SELECT nonce,"):
                return [("nonce", "original")] if mark else []
            if sql == "PRAGMA integrity_check":
                return [("ok",)]
            raise AssertionError(sql)
        return ack_proof(query, cp, "real", "nonce", phase)

    def test_before_uncommitted(self):
        self.assertEqual(self.check("before", "DISPATCHING", "", False), ("DISPATCHING", ""))

    def test_before_or_after_committed(self):
        for phase in ("before", "after"):
            self.assertEqual(self.check(phase, "VERIFYING", "real", True), ("VERIFYING", "real"))

    def test_after_missing_commit(self):
        with self.assertRaises(AssertionError):
            self.check("after", "DISPATCHING", "", False)

    def test_wrong_native_receipt(self):
        with self.assertRaises(AssertionError):
            self.check("after", "VERIFYING", "fabricated", True)

    def test_partial_commit(self):
        with self.assertRaises(AssertionError):
            self.check("before", "VERIFYING", "real", False)

    def test_terminal_or_mismatched_original(self):
        with self.assertRaises(AssertionError):
            self.check("after", "SUCCEEDED", "real", True)
        with self.assertRaises(AssertionError):
            self.check("after", "VERIFYING", "real", True, tracking="other")
        with self.assertRaises(AssertionError):
            self.check("after", "VERIFYING", "real", True, events=["ADMITTED", "VERIFYING"])


if __name__ == "__main__":
    unittest.main()

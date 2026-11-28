"""Private oracle tests; no native fixture is executed by this suite."""
import unittest
from pr191_scoped_update_ack_wal import ack_proof, restart_proof


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


class RestartOracle(unittest.TestCase):
    def fixture(self, committed):
        cp = {"update_id": "original", "tracking_id": "tracking", "box_id": "uuid"}
        live = ("VERIFYING", "native") if committed else ("DISPATCHING", "")
        identity = ("original", "tracking", live[0], live[1], "uuid", "uuid")
        claims = [("project", "box:name")]
        events = [(1, "ADMITTED", ""), (2, "DISPATCHING", "")]
        if committed:
            events.append((3, "VERIFYING", ""))
        cut = ([identity + (1, "")], claims, events, [])
        after_identity = identity[:2] + ("UNKNOWN",) + identity[3:]
        after = ([after_identity + (2, "PROCESS_RESTART_OR_LOST_ACK")], claims,
                 events + [(len(events) + 1, "UNKNOWN", "PROCESS_RESTART_OR_LOST_ACK")], [])
        final = dict(zip(("operation_id", "tracking_id", "state", "native_operation_id",
                          "box_uuid", "incarnation"), after_identity))
        return cut, after, final, cp, live, "project", "name"

    def test_both_ack_outcomes(self):
        for committed in (False, True):
            restart_proof(*self.fixture(committed))

    def test_mcp_mismatch(self):
        for field in ("operation_id", "tracking_id", "state", "native_operation_id", "box_uuid", "incarnation"):
            args = self.fixture(True)
            args[2][field] = "wrong"
            with self.assertRaises(AssertionError):
                restart_proof(*args)

    def test_ledger_mismatch(self):
        for index in (0, 1, 2, 3, 4, 5, 7):
            args = self.fixture(True)
            row = list(args[1][0][0])
            row[index] = "wrong"
            args[1][0][0] = tuple(row)
            with self.assertRaises(AssertionError):
                restart_proof(*args)

    def test_claim_event_and_audit_mismatch(self):
        for index in (1, 2, 3):
            args = self.fixture(True)
            after = list(args[1])
            after[index] = [("wrong", "wrong")]
            with self.assertRaises(AssertionError):
                restart_proof(args[0], tuple(after), *args[2:])


if __name__ == "__main__":
    unittest.main()

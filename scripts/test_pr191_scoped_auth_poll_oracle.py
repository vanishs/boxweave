"""Calibrate the independent installed long-poll/cursor witness in memory."""
from copy import deepcopy
import unittest

from pr191_scoped_auth_poll_oracle import CURSOR, GET, KEYS, PASS, digest, verify


class OriginalPollCursorOracle(unittest.TestCase):
    def setUp(self):
        self.project = "user-1001"
        self.before = {k: [] for k in ("ops", "events", "claims", "outputs", "audit")}
        self.before["ops"].append(["a"*32, "b"*32, self.project, "c"*32, "c"*32,
                                   "box_exec", "previous", "prior", "d"*64, "SUCCEEDED",
                                   "", "", "", "", 1])
        self.before["events"].append(["a"*32, 1, "SUCCEEDED"])
        self.after = deepcopy(self.before)
        self.log = [m+"=PASS" for m in PASS]
        for i in range(7):
            oid, tracking = f"{i+1:032x}", f"{i+10:032x}"
            if i < 2:
                work, agent = GET, ("revoke", "expire")[i]
                actor = f"{i+20:032x}"
                key = KEYS[(work, agent)]
                self.log.append(f"P1B_SCOPED_GET_AUTH_ORIGINAL {agent} {oid} {tracking}")
            else:
                work, agent = CURSOR, "p1-cursor"
                actor = f"{22:032x}"
                key = KEYS[(work, agent)] % (i-2)
            self.after["ops"].append([oid, tracking, self.project, actor, actor, "box_exec",
                                      work, agent, digest(key), "CANCELLED",
                                      "", "", "", "CANCELLED_BEFORE_DISPATCH", i+2])
            self.after["events"] += [[oid, 2*i+2, "ADMITTED"], [oid, 2*i+3, "CANCELLED"]]

    def test_exact_originals(self):
        verify(self.before, self.after, self.project, self.log)

    def test_native_or_prior_mutation_refused(self):
        for index, field, value in ((0, 9, "UNKNOWN"), (1, 10, "unexpected"),
                                    (2, 1, "f"*32), (3, 2, "foreign")):
            after = deepcopy(self.after)
            after["ops"][index][field] = value
            with self.subTest(index=index, field=field), self.assertRaises(AssertionError):
                verify(self.before, after, self.project, self.log)

    def test_missing_event_and_marker_refused(self):
        after = deepcopy(self.after)
        after["events"].pop()
        with self.assertRaises(AssertionError):
            verify(self.before, after, self.project, self.log)
        with self.assertRaises(AssertionError):
            verify(self.before, self.after, self.project, self.log[1:])


if __name__ == "__main__":
    unittest.main()

"""Shared golden vectors for diagnostic aggregation across trusted origins.

The root-owned host CLI and the generated client script each embed their own
small aggregation logic; identical check sets must classify identically.
These vectors freeze that contract for both implementations.
"""
import unittest

from bwmini import host_cli


def c(cid, state, required=True):
    return host_cli.check(cid, state, "CODE_X", "fixture", required=required)


AGGREGATE_VECTORS = (
    ((), "incomplete", 2),
    ((c("a", "passed"),), "passed", 0),
    ((c("a", "failed"),), "failed", 1),
    ((c("a", "passed"), c("b", "passed")), "passed", 0),
    ((c("a", "passed"), c("b", "failed")), "failed", 1),
    ((c("a", "failed"), c("b", "not_verified")), "failed", 1),
    ((c("a", "passed"), c("b", "not_verified")), "incomplete", 2),
    ((c("a", "not_verified"), c("b", "not_checked")), "incomplete", 2),
    ((c("a", "passed"), c("b", "not_checked", required=False)), "passed", 0),
    ((c("a", "failed"), c("b", "not_checked", required=False)), "failed", 1),
    ((c("a", "not_checked"),), "incomplete", 2),
    ((c("a", "unexpected"),), "incomplete", 2),
)


class HostGoldenTests(unittest.TestCase):
    def test_host_cli_aggregation_matches_golden_vectors(self):
        for checks, overall, code in AGGREGATE_VECTORS:
            with self.subTest(overall=overall, size=len(checks)):
                self.assertEqual(host_cli.overall_state(list(checks)), overall)
                self.assertEqual(host_cli.exit_code_for(overall), code)

    def test_check_shape_is_fixed(self):
        self.assertEqual(host_cli.check("x", "passed", "C", "scope"),
                         {"id": "x", "state": "passed", "code": "C",
                          "scope": "scope", "required": True})
        self.assertEqual(host_cli.check("x", "failed", "C", "scope", required=False, detail="d")["detail"], "d")

    def test_unit_classification_definite_negative_only(self):
        healthy = {"LoadState": "loaded", "ActiveState": "active", "UnitFileState": "enabled"}
        self.assertEqual(host_cli.classify_unit(healthy, expect_enabled=True), ("passed", "UNIT_OBSERVED"))
        static = {"LoadState": "loaded", "ActiveState": "active", "UnitFileState": "static"}
        self.assertEqual(host_cli.classify_unit(static, expect_enabled=False), ("passed", "UNIT_OBSERVED"))
        self.assertEqual(host_cli.classify_unit(static, expect_enabled=True)[0], "not_verified")
        self.assertEqual(host_cli.classify_unit({"ActiveState": "inactive"}, True),
                         ("failed", "UNIT_OBSERVED_INACTIVE"))
        self.assertEqual(host_cli.classify_unit({"ActiveState": "failed"}, True),
                         ("failed", "UNIT_OBSERVED_FAILED"))
        self.assertEqual(host_cli.classify_unit({"state": "unavailable"}, True),
                         ("not_verified", "UNIT_STATE_UNAVAILABLE"))
        self.assertEqual(host_cli.classify_unit(None, True), ("not_verified", "UNIT_STATE_UNAVAILABLE"))
        self.assertEqual(host_cli.classify_unit({"LoadState": "not-found"}, True)[0], "not_verified")


if __name__ == "__main__":
    unittest.main()

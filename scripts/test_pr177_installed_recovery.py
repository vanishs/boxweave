"""Regression tests for the integrated installed-recovery fixture."""
import unittest
from unittest.mock import Mock, patch

import pr177_d4_installed_recovery as fixture


class InstalledRecoveryTest(unittest.TestCase):
    def test_legacy_cleanup_stops_only_sibling_before_delete(self):
        states = {"a": "Stopped", "sibling": "Running"}
        events = []
        client = Mock()

        def native_op(_client, name, args, key):
            self.assertIs(_client, client)
            self.assertNotIn(key, [e[2] for e in events])
            events.append((name, args, key))
            if name == "box_stop":
                self.assertEqual(states[args["box_id"]], "Running")
                states[args["box_id"]] = "Stopped"
            elif name == "box_delete":
                self.assertEqual(states[args["box_id"]], "Stopped")
                del states[args["box_id"]]

        def wait(_client, predicate, _timeout):
            self.assertIs(_client, client)
            self.assertEqual(states, {})
            self.assertFalse(predicate({"address_retirement": {"pending": 1}}))
            self.assertTrue(predicate({"address_retirement": {"pending": 0}}))

        with patch.object(fixture, "op", side_effect=native_op), patch.object(
            fixture, "wait_status", side_effect=wait
        ):
            fixture.cleanup_legacy(client, {"d4-a": "a", "d4-sibling": "sibling"}, "f")
        self.assertEqual([event[0] for event in events],
                         ["port_forward_delete", "box_stop", "box_delete", "box_delete"])
        self.assertEqual(events[1][1], {"box_id": "sibling"})

    def test_legacy_cleanup_refusal_does_not_continue(self):
        client = Mock()
        with patch.object(fixture, "op", side_effect=[None, RuntimeError("stop failed")]) as op:
            with self.assertRaisesRegex(RuntimeError, "stop failed"):
                fixture.cleanup_legacy(client, {"d4-a": "a", "d4-sibling": "sibling"}, "f")
            self.assertEqual(op.call_count, 2)

    def test_independent_postcheck_rejects_disagreement(self):
        client = Mock()
        row = {"operation_id": "o", "state": "SUCCEEDED", "native_operation_id": "n"}
        client.operation_get.return_value = row
        client.activity.return_value = {"evidence": {"original": dict(row)}}
        fixture.recheck_original(client, "o", "SUCCEEDED", "n")
        for key, value in (("operation_id", "other"), ("state", "UNKNOWN"),
                           ("native_operation_id", "other")):
            with self.subTest(key=key):
                client.activity.return_value = {"evidence": {"original": dict(row, **{key: value})}}
                with self.assertRaises(AssertionError):
                    fixture.recheck_original(client, "o", "SUCCEEDED", "n")

    def test_postcheck_accepts_absent_native_id_without_inventing_one(self):
        client = Mock()
        row = {"operation_id": "o", "state": "SUCCEEDED"}
        client.operation_get.return_value = row
        client.activity.return_value = {"evidence": {"original": dict(row)}}
        fixture.recheck_original(client, "o", "SUCCEEDED", None)


if __name__ == "__main__":
    unittest.main()

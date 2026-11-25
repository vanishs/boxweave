"""Offline guards for the new live roundtrip fixture; not network proof."""
import unittest
from unittest.mock import patch, Mock

import pr188_scoped_roundtrip as fixture


class RoundtripTest(unittest.TestCase):
    def test_guest_checks_actual_exit_not_operation_state(self):
        c=Mock()
        c.operation_output.return_value={"exit_code":7}
        with patch.object(fixture,"op",return_value={"operation_id":"o"}):
            self.assertEqual(fixture.guest(c,"b",fixture.HTTP,"key","ip",expected=7),"o")
            with self.assertRaises(AssertionError):
                fixture.guest(c,"b",fixture.HTTP,"key","ip")

    def test_first_deny_failure_does_not_admit_grant_or_forward(self):
        with patch.object(fixture,"guest",side_effect=[None]*6+[AssertionError("deny failed")]), patch.object(fixture,"binding",return_value={"ipv4":"10.0.0.2"}), patch.object(fixture,"op") as op:
            with self.assertRaisesRegex(AssertionError,"deny failed"):
                fixture.network(Mock(),"a","b","first")
            op.assert_not_called()

    def test_forward_requires_actual_http(self):
        with patch.object(fixture,"guest"), patch.object(fixture,"binding",return_value={"ipv4":"10.0.0.2"}), patch.object(fixture,"op") as op, patch.object(fixture,"http_status",return_value=503):
            with self.assertRaises(AssertionError):
                fixture.network(Mock(),"a","b","first")
            self.assertEqual([call.args[1] for call in op.call_args_list],["box_peer_set","port_forward_create"])

    def test_roundtrip_never_applies_on_unapproved_runner(self):
        with patch.dict(fixture.os.environ,{"GITHUB_ACTIONS":"false","RUNNER_ENVIRONMENT":"github-hosted"}), patch.object(fixture.os,"geteuid",return_value=0), patch.object(fixture,"tty_run") as tty:
            with self.assertRaises(AssertionError):
                fixture.second_install("user-1001","old","candidate","digest",("a","b"),1)
            tty.assert_not_called()


if __name__=="__main__":
    unittest.main()

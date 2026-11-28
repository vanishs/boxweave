"""Source-only negative-path calibration, no real credentials or native host."""
import contextlib
import io
import json
import unittest
from unittest.mock import patch

import s25_dual_auth as lab
import s25_independent_verifier as verifier

ENV = {"GITHUB_ACTIONS":"true","RUNNER_ENVIRONMENT":"github-hosted","P1B_S25_SCOPED":"1"}

class DualScopeAuth(unittest.TestCase):
    def exercise(self, fail=False):
        calls=[]
        def fake(prefix,*args,payload=None):
            if args[0]=="id":
                return ("1002" if "incus" in prefix else "1001")+"\n"
            if args[0]==lab.BIN:
                return json.dumps({"scoped_peer_mode":"SCOPED_PEER_FRESH_V2",
                                   "service_active":True,"native_project":args[-1]})
            assert args[:2]==("python3","-c") and args[2]==lab.PROBE
            calls.append(json.loads(payload))
            if fail and len(calls)==2:
                raise AssertionError("foreign auth unexpectedly accepted")
            return ""
        issued=[("a"*32,"synthetic-a"),("b"*32,"synthetic-b")]
        with patch.dict(lab.os.environ,ENV,clear=True), patch.object(lab,"run",side_effect=fake), \
             patch.object(lab,"issue",side_effect=issued), patch.object(lab,"revoke") as revoke, \
             contextlib.redirect_stdout(io.StringIO()) as out:
            if fail:
                with self.assertRaises(AssertionError):
                    lab.verify("user-1001")
                self.assertNotIn("=PASS",out.getvalue())
            else:
                lab.verify("user-1001")
                self.assertIn("=PASS",out.getvalue())
            self.assertEqual(calls,[["synthetic-a","synthetic-b"],["synthetic-b","synthetic-a"]])
            self.assertEqual([c.args[-1] for c in revoke.call_args_list],["a"*32,"b"*32])
            self.assertNotIn("synthetic-",out.getvalue())

    def test_both_directions_revoke(self):
        self.exercise()

    def test_failure_revoke_without_pass(self):
        self.exercise(fail=True)

    def test_outside_lane_refused_before_cli(self):
        with patch.dict(lab.os.environ,{},clear=True),patch.object(lab,"run") as run:
            with self.assertRaises(AssertionError):
                lab.verify("user-1001")
            run.assert_not_called()

    def test_independent_oracle_refuses_collapsed_original(self):
        outer={"name":"bw-s25-mcp-l1","uuid":"a","create_op":"b"}
        with patch.dict(lab.os.environ,ENV,clear=True), \
             patch.object(verifier,"inspect_l1",return_value=outer), \
             patch.object(verifier,"inner",return_value=json.dumps(outer)), \
             patch.object(verifier,"cli") as query:
            with self.assertRaises(SystemExit):
                verifier.verify_dual_scope()
            query.assert_not_called()

    def test_independent_oracle_checks_both_ledgers(self):
        outer={"name":"bw-s25-mcp-l1","uuid":"a","create_op":"b"}
        nested={"name":"bw-s25-mcp-l1","uuid":"c","create_op":"d"}
        with patch.dict(lab.os.environ,ENV,clear=True), \
             patch.object(verifier,"inspect_l1",return_value=outer), \
             patch.object(verifier,"inner",return_value=json.dumps(nested)) as inside, \
             patch.object(verifier,"cli") as outside, \
             patch.object(verifier,"verify_scoped_layers") as pins, \
             contextlib.redirect_stdout(io.StringIO()):
            verifier.verify_dual_scope()
            self.assertEqual(inside.call_args.args[-2:],("a","b"))
            self.assertEqual(outside.call_args.args[-2:],("c","d"))
            pins.assert_called_once_with()

    def test_scoped_same_name_legacy_distinct(self):
        with patch.dict(lab.os.environ,ENV,clear=True):
            self.assertEqual(verifier.inner_name(),"bw-s25-mcp-l1")
        with patch.dict(lab.os.environ,{},clear=True):
            self.assertEqual(verifier.inner_name(),"bw-s25-mcp-l2")

if __name__=="__main__":
    unittest.main()

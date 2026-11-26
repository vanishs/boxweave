"""Source-only tests for invalid ledger creation and refusal oracle."""
import json
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from pr191_scoped_schema_refusal import fixtures, refused
import pr191_scoped_update_transport as update


class SchemaRefusal(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.directory = Path(temp.name)
        self.cases = dict(fixtures(self.directory, 7))

    def result(self, code=1, ready=False, reason="DIAGNOSTIC_DB_SCHEMA_OR_INTEGRITY_UNVERIFIED"):
        return subprocess.CompletedProcess([], code, json.dumps({
            "ready":ready,"diagnostic":{"database":{"state":"UNVERIFIED","reason":reason}}}),"")

    def test_five_distinct_layouts(self):
        self.assertEqual(len(self.cases),5)
        for name in ("old","ambiguous","versioned-foreign"):
            with sqlite3.connect(self.cases[name]) as db:
                expected = {"old":[(6,)],"ambiguous":[(7,),(7,)],"versioned-foreign":[(7,)]}[name]
                self.assertEqual(db.execute("SELECT version FROM schema_version").fetchall(),expected)
        for path in self.cases.values():
            refused(self.result(),path,path.read_bytes())

    def test_not_generic_failure(self):
        path = self.cases["foreign"]
        for result in (self.result(code=0),self.result(ready=True),self.result(reason="DIAGNOSTIC_DB_IDENTITY_UNVERIFIED")):
            with self.assertRaises(AssertionError):
                refused(result,path,path.read_bytes())

    def test_detect_byte_or_sidecar_change(self):
        path = self.cases["corrupt"]
        with self.assertRaises(AssertionError):
            refused(self.result(),path,b"different")
        path.with_name(path.name+"-wal").write_bytes(b"new sidecar")
        with self.assertRaises(AssertionError):
            refused(self.result(),path,path.read_bytes())


class ObsoleteUnknownRoute(unittest.TestCase):
    def test_all_obsolete_actions_refused(self):
        original = ([("a","b","UNKNOWN")],[("user-1001","box:fixture")],[],[])
        reply = subprocess.CompletedProcess([],1,"","scoped original requires the reviewed recovery route")
        with patch.object(update,"snapshot",return_value=original), patch.object(update,"rows",return_value=[]), patch.object(update.subprocess,"run",return_value=reply) as run:
            update.legacy_unknown_refusals("user-1001","a"*32,"b"*36)
            self.assertEqual([c.args[0][c.args[0].index("-action")+1] for c in run.call_args_list],["inspect","challenge","release"])

    def test_generic_failure_does_not_pass(self):
        original = ([("a","b","UNKNOWN")],[("user-1001","box:fixture")],[],[])
        reply = subprocess.CompletedProcess([],1,"","unrelated failure")
        with patch.object(update,"snapshot",return_value=original), patch.object(update,"rows",return_value=[]), patch.object(update.subprocess,"run",return_value=reply):
            with self.assertRaises(AssertionError):
                update.legacy_unknown_refusals("user-1001","a"*32,"b"*36)

    def test_changed_original_detected(self):
        original = ([("a","b","UNKNOWN")],[("user-1001","box:fixture")],[],[])
        changed = (original[0],[],[],[])
        reply = subprocess.CompletedProcess([],1,"","scoped original requires the reviewed recovery route")
        with patch.object(update,"snapshot",side_effect=[original,changed]), patch.object(update,"rows",return_value=[]), patch.object(update.subprocess,"run",return_value=reply):
            with self.assertRaises(AssertionError):
                update.legacy_unknown_refusals("user-1001","a"*32,"b"*36)


if __name__=="__main__":
    unittest.main()

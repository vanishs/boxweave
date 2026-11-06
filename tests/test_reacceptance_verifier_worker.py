"""A workflow rerun cannot silently reinstall an uncertain original Box."""
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from tools.reacceptance import verifier_worker as subject
from tools.reacceptance.engine import AcceptanceFault


class WorkerSafetyTests(unittest.TestCase):
    def test_missing_workflow_actor_is_not_invented_from_sudo_environment(self):
        with patch.object(subject.os,'geteuid',return_value=0), \
             patch.object(subject,'install') as install:
            with self.assertRaisesRegex(AcceptanceFault,'ACTOR_IDENTITY_REQUIRED'):
                subject.run('/unused','a'*40,'/missing',subject.WORKER_OUTPUT)
            install.assert_not_called()

    def test_unapproved_box_never_reaches_install(self):
        with patch.object(subject.os,'geteuid',return_value=0), \
             patch.object(subject,'check_owned_lab',side_effect=AcceptanceFault('GRANT_EXPIRED')), \
             patch.object(subject,'install') as install:
            with self.assertRaisesRegex(AcceptanceFault,'GRANT_EXPIRED'):
                subject.run('/unused','a'*40,'/missing',subject.WORKER_OUTPUT,workflow_run_id=1,workflow_run_attempt=1)
            install.assert_not_called()

    def test_existing_original_intent_blocks_reinstall_before_dispatch(self):
        with tempfile.TemporaryDirectory() as d:
            root=Path(d); original=root/'original-worker';original.mkdir()
            (original/'worker-intent.json').write_text('{"state":"UNKNOWN"}')
            (root/'release.json').write_text('{"source":"'+'a'*40+'"}')
            with patch.object(subject.os,'geteuid',return_value=0), \
                 patch.object(subject,'check_owned_lab',return_value={}), \
                 patch.object(subject,'asset_index',return_value={}), \
                 patch.object(subject,'WORKER_OUTPUT',original), \
                 patch.object(subject,'install') as install:
                with self.assertRaisesRegex(AcceptanceFault,'NO_REPLAY'):
                    subject.run(root,'a'*40,'/grant',original,workflow_run_id=1,workflow_run_attempt=1)
                install.assert_not_called()
            self.assertEqual((original/'worker-intent.json').read_text(),'{"state":"UNKNOWN"}')

    def test_caller_cannot_choose_new_path_to_avoid_original_marker(self):
        with tempfile.TemporaryDirectory() as d:
            root=Path(d)
            (root/'release.json').write_text('{"source":"'+'a'*40+'"}')
            with patch.object(subject.os,'geteuid',return_value=0), \
                 patch.object(subject,'check_owned_lab',return_value={}), \
                 patch.object(subject,'asset_index',return_value={}), \
                 patch.object(subject,'install') as install:
                with self.assertRaisesRegex(AcceptanceFault,'NO_REPLAY'):
                    subject.run(root,'a'*40,'/grant',root/'new-attempt',workflow_run_id=1,workflow_run_attempt=1)
                install.assert_not_called()

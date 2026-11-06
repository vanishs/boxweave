"""Evidence transport calibration: fixtures never claim a real installed PASS."""
import base64
import contextlib
import io
import json
from pathlib import Path
import shutil
import unittest
import zipfile

from tools.reacceptance import box_acceptance as subject
from tools.reacceptance.engine import AcceptanceFault
from tests import test_reacceptance_signing_preflight as signing_fixture


class BoxAcceptanceTests(unittest.TestCase):
    def setUp(self):
        self.fixture = signing_fixture.SigningPreflightTests('test_complete_fixture_is_only_eligible_never_signature_or_authority')
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        f = self.fixture
        self.input = f.root / 'submission-input'
        self.input.mkdir()
        shutil.copyfile(f.attestation, self.input / 'release-gate.json')
        shutil.copytree(f.root / 'cases', self.input / 'cases')
        self.output = f.root / 'workflow-input.json'

    def pack(self):
        f = self.fixture
        return subject.pack(f.assets, self.input, f.source, f.context['candidate_run_id'], self.output)

    def roundtrip(self):
        self.pack()
        payload = json.loads(self.output.read_text())
        dest = self.fixture.root / 'recorded'
        subject.unpack(payload['inputs']['evidence_bundle_b64'], dest)
        return dest

    def test_roundtrip_preserves_original_and_records_only_actual_context(self):
        f = self.fixture
        original = (self.input / 'release-gate.json').read_bytes()
        dest = self.roundtrip()
        self.assertEqual((dest / 'original-receipt.json').read_bytes(), original)
        submitted = (dest / 'submitted-receipt.json').read_bytes()
        result = subject.stamp_context(f.assets, dest, f.source, f.context)
        self.assertEqual(result['decision'], 'BOX_ACCEPTANCE_EVIDENCE_VALIDATED')
        self.assertFalse(result['signature_produced'])
        self.assertFalse(result['tests_executed_by_this_command'])
        self.assertEqual((dest / 'submitted-receipt.json').read_bytes(), submitted)
        self.assertEqual((dest / 'original-receipt.json').read_bytes(), original)
        self.assertEqual(subject.check(f.assets, dest, f.source, f.context), result)
        with self.assertRaisesRegex(AcceptanceFault, 'ALREADY_RECORDED'):
            subject.stamp_context(f.assets, dest, f.source, f.context)

    def test_partial_and_same_actor_reports_cannot_be_packaged_as_acceptance(self):
        p = self.input / 'release-gate.json'
        value = json.loads(p.read_text())
        value['scenarios']['SYS-LIFE-01']['status'] = 'PARTIAL'
        p.write_text(json.dumps(value))
        with self.assertRaisesRegex(AcceptanceFault, 'REQUIRED_SCENARIO_NOT_VERIFIED'):
            self.pack()
        value['scenarios']['SYS-LIFE-01']['status'] = 'PASS'
        value['verifier']['actor_id'] = value['finder']['actor_id']
        p.write_text(json.dumps(value))
        with self.assertRaisesRegex(AcceptanceFault, 'SAME_ACTOR'):
            self.pack()
        self.assertFalse(self.output.exists())

    def test_changed_case_is_not_laundered_by_recomputed_digest(self):
        p = self.input / 'cases/SYS-LIFE-01/oracle.json'
        value = json.loads(p.read_text())
        value['observed_expected_effect'] = False
        p.write_text(json.dumps(value))
        with self.assertRaisesRegex(AcceptanceFault, 'SUPPLIED_DIGEST_MISMATCH'):
            self.pack()

    def test_absent_hash_is_derived_but_observations_still_validated(self):
        p = self.input / 'release-gate.json'
        value = json.loads(p.read_text())
        del value['scenarios']['SYS-LIFE-01']['oracle_sha256']
        p.write_text(json.dumps(value))
        dest = self.roundtrip()
        report = json.loads((dest / 'release-gate.json').read_text())
        self.assertEqual(report['scenarios']['SYS-LIFE-01']['oracle_sha256'],
                         subject.sha((dest / 'cases/SYS-LIFE-01/oracle.json').read_bytes()))
        self.assertNotIn('workflow', report)

    def test_context_stamping_refuses_candidate_substitution(self):
        f = self.fixture
        dest = self.roundtrip()
        changed = dict(f.context, candidate_run_id=f.context['candidate_run_id'] + 1)
        with self.assertRaisesRegex(AcceptanceFault, 'SUBMISSION_RUN_MISMATCH'):
            subject.stamp_context(f.assets, dest, f.source, changed)
        self.assertFalse((dest / 'recording.json').exists())

    def test_forged_archive_cannot_promote_preserved_partial_original(self):
        # Construct a coherent forged submission directly, deliberately bypassing
        # pack(). No transport digest can authorize changed original results.
        import copy
        f = self.fixture
        for changed_field in ('status', 'decision', 'actor', 'supplied_digest'):
            with self.subTest(changed_field=changed_field):
                original = copy.deepcopy(f.evidence)
                if changed_field == 'status':
                    original['scenarios']['SYS-LIFE-01']['status'] = 'PARTIAL'
                elif changed_field == 'decision':
                    original['decision'] = 'DENIED'
                elif changed_field == 'actor':
                    original['verifier']['actor_id'] = 'another-original-actor'
                else:
                    original['scenarios']['SYS-LIFE-01']['oracle_sha256'] = 'f' * 64
                submitted = copy.deepcopy(f.evidence)
                submitted.pop('workflow', None)
                files = {p.relative_to(self.input).as_posix(): p.read_bytes()
                         for p in self.input.rglob('*') if p.is_file()}
                files['original-receipt.json'] = subject.encoded_object(original)
                files['submitted-receipt.json'] = subject.encoded_object(submitted)
                files['release-gate.json'] = files['submitted-receipt.json']
                files['submission.json'] = subject.encoded_object({'schema_version': 1,
                    'candidate_run_id': f.context['candidate_run_id'],
                    'files_sha256': {name: subject.sha(data) for name, data in files.items()
                                     if name != 'release-gate.json'}})
                dest = f.root / ('forged-' + changed_field)
                dest.mkdir()
                for name, data in files.items():
                    p = dest / name
                    p.parent.mkdir(parents=True, exist_ok=True)
                    p.write_bytes(data)
                with self.assertRaisesRegex(AcceptanceFault, 'ORIGINAL_OBSERVATIONS_CHANGED|SUPPLIED_DIGEST_MISMATCH'):
                    subject.stamp_context(f.assets, dest, f.source, f.context)
                self.assertFalse((dest / 'recording.json').exists())
                archive = io.BytesIO()
                with zipfile.ZipFile(archive, 'w', compression=zipfile.ZIP_DEFLATED) as z:
                    for name, data in files.items():
                        z.writestr(name, data)
                out = f.root / ('forged-unpack-' + changed_field)
                with self.assertRaises(AcceptanceFault):
                    subject.unpack(base64.b64encode(archive.getvalue()).decode(), out)
                self.assertFalse(out.exists())

    def test_recorded_receipt_cannot_change_observations(self):
        f = self.fixture
        dest = self.roundtrip()
        subject.stamp_context(f.assets, dest, f.source, f.context)
        p = dest / 'release-gate.json'
        report = json.loads(p.read_text())
        report['verifier']['actor_id'] = 'forged-later'
        p.write_text(json.dumps(report))
        with self.assertRaisesRegex(AcceptanceFault, 'OBSERVATIONS_CHANGED'):
            subject.check(f.assets, dest, f.source, f.context)

    def test_archive_rejects_traversal_duplicate_symlink_and_extra_before_writes(self):
        for mode in ('traversal', 'duplicate', 'symlink', 'extra'):
            with self.subTest(mode=mode):
                b = io.BytesIO()
                with zipfile.ZipFile(b, 'w', compression=zipfile.ZIP_DEFLATED) as z:
                    name = '../escape' if mode == 'traversal' else 'script.py' if mode == 'extra' else 'release-gate.json'
                    info = zipfile.ZipInfo(name)
                    info.external_attr = (0o120777 if mode == 'symlink' else 0o100600) << 16
                    z.writestr(info, '{}')
                    if mode == 'duplicate':
                        import warnings
                        with warnings.catch_warnings():
                            warnings.simplefilter('ignore')
                            z.writestr(info, '{}')
                out = self.fixture.root / ('bad-' + mode)
                with self.assertRaises(AcceptanceFault):
                    subject.unpack(base64.b64encode(b.getvalue()).decode(), out)
                self.assertFalse(out.exists())

    def test_small_zip_with_large_expanded_member_is_rejected(self):
        b = io.BytesIO()
        with zipfile.ZipFile(b, 'w', compression=zipfile.ZIP_DEFLATED) as z:
            z.writestr('release-gate.json', '{"padding":"' + 'x' * subject.MAX_FILE + '"}')
        out = self.fixture.root / 'bomb'
        with self.assertRaisesRegex(AcceptanceFault, 'OVERSIZED_MEMBER'):
            subject.unpack(base64.b64encode(b.getvalue()).decode(), out)
        self.assertFalse(out.exists())

    def test_encoded_size_and_extra_source_files_fail_closed(self):
        with self.assertRaisesRegex(AcceptanceFault, 'ENCODED_SIZE'):
            subject.unpack('A' * (subject.MAX_ENCODED + 1), self.fixture.root / 'oversized')
        (self.input / 'unreviewed.log').write_text('not transported')
        with self.assertRaisesRegex(AcceptanceFault, 'EXTRA_OR_UNSAFE'):
            self.pack()

    def test_source_and_output_symlinks_are_refused(self):
        alias = self.fixture.root / 'output-alias'
        alias.symlink_to(self.fixture.root, target_is_directory=True)
        with self.assertRaisesRegex(AcceptanceFault, 'PARENT_SYMLINK'):
            subject.unpack('e30=', alias / 'escape')
        (self.input / 'cases/SYS-LIFE-01/oracle.json').unlink()
        (self.input / 'cases/SYS-LIFE-01/oracle.json').symlink_to(self.fixture.root / 'cases/SYS-LIFE-01/oracle.json')
        with self.assertRaisesRegex(AcceptanceFault, 'SYMLINK_REFUSED'):
            self.pack()

    def test_incompressible_evidence_does_not_get_silently_truncated(self):
        from unittest.mock import patch
        with patch.object(subject, 'MAX_ENCODED', 100):
            with self.assertRaisesRegex(AcceptanceFault, 'BUNDLE_TOO_LARGE'):
                self.pack()
        self.assertFalse(self.output.exists())

    def test_stdin_cli_denies_bad_archive_without_echoing_input(self):
        secret_like_input = 'not valid base64 input'
        from unittest.mock import patch
        out = io.StringIO()
        with patch('sys.stdin', io.StringIO(secret_like_input)), contextlib.redirect_stdout(out):
            code = subject.main(['unpack', '--out', str(self.fixture.root / 'invalid')])
        self.assertEqual(code, 2)
        self.assertNotIn(secret_like_input, out.getvalue())
        self.assertEqual(json.loads(out.getvalue())['decision'], 'DENIED')


if __name__ == '__main__':
    unittest.main()

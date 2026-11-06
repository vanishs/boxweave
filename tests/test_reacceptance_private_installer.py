"""Adversarial controls for trusted pre-execution private Host installer."""
import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from types import SimpleNamespace

from tools.reacceptance import private_installer as subject
from tools.reacceptance.engine import AcceptanceFault


class PrivateCandidateTrustBeforeExecutionTests(unittest.TestCase):
    def fixture(self, d):
        root = Path(d)
        stage = root / "private-stage"
        stage.mkdir(mode=0o700)
        stage.chmod(0o700)
        files = {"online.py": b"print('would execute')",
                 "boxweave.zip": b"candidate bytes"}
        for name, raw in files.items():
            path = stage / name
            path.write_bytes(raw)
            path.chmod(0o600)
        source = "a"*40
        manifest = {"format": 1, "version": "0.49.0", "source": source,
                    "files": {name: hashlib.sha256(data).hexdigest()
                              for name, data in files.items()}}
        f = stage/"release.json"
        f.write_text(json.dumps(manifest))
        f.chmod(0o600)
        pin = hashlib.sha256(f.read_bytes()).hexdigest()
        return stage, pin, source

    def patch_owners(self):
        original = Path.lstat
        def root_stats(path, *args, **kwargs):
            item = original(path, *args, **kwargs)
            return SimpleNamespace(st_mode=item.st_mode, st_uid=0,
                                   st_nlink=item.st_nlink, st_size=item.st_size)
        return patch.object(Path, "lstat", root_stats)

    def test_fixed_reviewed_bytes_validated_before_candidate_execution(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as d:
            stage, pin, source = self.fixture(d)
            with patch.object(subject, "RELEASE_FILES", ("online.py", "boxweave.zip")), (
                 self.patch_owners()), (
                 patch.object(subject, "verify_zip", return_value={
                    "source":{"commit":source,"state":"clean"},"version":"0.49.0"})) as validate, (
                 patch.object(subject.os, "geteuid", return_value=0)), (
                 patch.object(subject.subprocess, "run")) as execute:
                result = subject.run(stage,"0.49.0",pin,source,
                                     Path(d)/"preflight")
                self.assertEqual(result["state"], "PREFLIGHT_ONLY")
                validate.assert_called_once()
                execute.assert_not_called()
                self.assertTrue((Path(d)/"preflight/candidate-preflight.json").exists())

    def test_changed_candidate_online_script_is_rejected_before_execute(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as d:
            stage, pin, source = self.fixture(d)
            (stage/"online.py").write_bytes(b"changed malicious bytes")
            with patch.object(subject,"RELEASE_FILES", ("online.py", "boxweave.zip")), (
                 self.patch_owners()), (
                 patch.object(subject, "verify_zip")) as validate, (
                 patch.object(subject.os, "geteuid", return_value=0)), (
                 patch.object(subject.subprocess, "run")) as execute:
                with self.assertRaisesRegex(AcceptanceFault,
                                            "CANDIDATE_ASSET_SHA_MISMATCH_online.py"):
                    subject.run(stage,"0.49.0",pin,source,Path(d)/"out")
                execute.assert_not_called()
                validate.assert_not_called()

    def test_apply_requires_specific_disposable_identity(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as d:
            stage, pin, source = self.fixture(d)
            with patch.object(subject,"RELEASE_FILES", ("online.py", "boxweave.zip")), (
                 self.patch_owners()), (
                 patch.object(subject, "verify_zip", return_value={
                     "source":{"commit":source,"state":"clean"},"version":"0.49.0"})), (
                 patch.object(subject.os,"geteuid",return_value=0)), (
                 patch.object(subject.subprocess,"run")) as execute:
                with self.assertRaisesRegex(AcceptanceFault,
                                            "PRIVATE_INSTALL_EXPLICIT_CONFIRMATION_REQUIRED"):
                    subject.run(stage,"0.49.0",pin,source,
                                Path(d)/"out",apply=True)
                execute.assert_not_called()
                self.assertFalse((Path(d)/"out").exists())

    def test_root_owned_symlink_and_extra_stage_are_refused(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as d:
            stage, pin, source = self.fixture(d)
            (stage/"extra.sh").write_text("no")
            with patch.object(subject,"RELEASE_FILES", ("online.py", "boxweave.zip")), (
                 self.patch_owners()), (
                 patch.object(subject,"verify_zip")) as validate, (
                 patch.object(subject.subprocess,"run")) as execute:
                with self.assertRaisesRegex(AcceptanceFault,
                                            "CANDIDATE_EXTRA_OR_MISSING_ASSETS"):
                    subject.exact_stage(stage,"0.49.0",pin,source)
                validate.assert_not_called()
                execute.assert_not_called()
            (stage/"extra.sh").unlink()
            (stage/"online.py").unlink()
            (stage/"online.py").symlink_to(stage/"boxweave.zip")
            with patch.object(subject,"RELEASE_FILES", ("online.py", "boxweave.zip")), (
                 self.patch_owners()), (
                 patch.object(subject.subprocess,"run")) as execute:
                with self.assertRaisesRegex(AcceptanceFault,
                                            "CANDIDATE_ASSET_UNTRUSTED_online.py"):
                    subject.exact_stage(stage,"0.49.0",pin,source)
                execute.assert_not_called()


if __name__ == "__main__":
    unittest.main()


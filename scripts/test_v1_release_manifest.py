#!/usr/bin/env python3
"""Offline negative checks for the v1-only Release manifest and installer."""
import importlib.util
import contextlib
import io
import json
import socket
import sys
import types
from unittest import mock
import hashlib
import subprocess
from pathlib import Path
import struct
import tempfile
import unittest


ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location("v1_release_manifest", ROOT / "scripts/v1_release_manifest.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.dir = Path(self.temp.name)
        self.installer = self.dir / "source-installer.sh"
        self.installer.write_text("#!/bin/sh\n")
        for arch, machine in module.MACHINES.items():
            blob = bytearray(64)
            blob[0:6] = b"\x7fELF\x02\x01"
            struct.pack_into("<H", blob, 18, machine)
            (self.dir / ("boxweave-v1.0.0-rc.1-linux-" + arch)).write_bytes(blob)

    def test_good_manifest_and_integrity(self):
        m = module.assemble(self.dir, "v1.0.0-rc.1", "a" * 40, self.installer)
        self.assertEqual(m["product"], "boxweave-v1")
        lines = (self.dir / "SHA256SUMS").read_text().splitlines()
        self.assertEqual(len(lines), 5)
        output = subprocess.run(["sh", str(self.dir / "uninstall.sh"), "--help"],
                                text=True, capture_output=True)
        self.assertEqual(output.returncode, 0, output.stderr)
        self.assertIn("--full", output.stdout)
        self.assertIn("--scoped", output.stdout)
        self.assertIn("full-only Scoped Peer v2 retirement", output.stdout)
        self.assertIn("--resume-plan", output.stdout)
        self.assertIn("--resume", output.stdout)
        self.assertEqual(len(m["assets"]), 2)
        self.assertEqual(m["dependencies"]["go_toolchain"], "1.26.9")
        self.assertTrue(m["dependencies"]["incus_go_sdk"].startswith("v7."))

    def test_checkout_uninstall_wrapper_missing_core_is_clear(self):
        wrapper = self.dir / "uninstall-source.sh"
        wrapper.write_bytes((ROOT / "uninstall.sh").read_bytes())
        out = subprocess.run(["sh", str(wrapper), "--help"], text=True, capture_output=True)
        self.assertNotEqual(out.returncode, 0)
        self.assertIn("self-contained matching Release uninstall.sh", out.stderr)

    def test_wrong_machine_fails(self):
        arm = self.dir / "boxweave-v1.0.0-rc.1-linux-arm64"
        arm.write_bytes((self.dir / "boxweave-v1.0.0-rc.1-linux-amd64").read_bytes())
        with self.assertRaisesRegex(ValueError, "machine mismatch"):
            module.assemble(self.dir, "v1.0.0-rc.1", "b" * 40, self.installer)

    def test_reject_main_or_invalid_tag(self):
        with self.assertRaisesRegex(ValueError, "invalid v1"):
            module.assemble(self.dir, "v0.50.5", "b" * 40, self.installer)

    def test_external_installer_plan_and_wrong_digest(self):
        path = self.dir / "boxweave-v1.0.0-rc.1-linux-amd64"
        valid_sha = hashlib.sha256(path.read_bytes()).hexdigest()
        base = ["sh", str(ROOT / "install.sh"), "--local-binary", str(path),
                "--project", "user-1001", "--sha256"]
        # A complete plan now requires native review; the fake ELF cannot
        # provide one. Exercise the isolated reviewed path below.
        calls, error = self.exercise_guided_install("NO", plan_only=True)
        self.assertIsNone(error)
        self.assertEqual(calls, ["native-scoped-fresh-plan"])
        bad = subprocess.run(base + ["0" * 64, "--plan"], text=True, capture_output=True)
        self.assertNotEqual(bad.returncode, 0)
        self.assertIn("SHA256 mismatch", bad.stderr)

    def test_external_installer_rejects_legacy_tag_and_unapproved_apply(self):
        bad_tag = subprocess.run(["sh", str(ROOT / "install.sh"), "--version", "v0.50.5",
                                  "--plan"], text=True, capture_output=True)
        self.assertNotEqual(bad_tag.returncode, 0)
        self.assertIn("expected v1", bad_tag.stderr)
        path = self.dir / "boxweave-v1.0.0-rc.1-linux-amd64"
        valid_sha = hashlib.sha256(path.read_bytes()).hexdigest()
        denied = subprocess.run([
            "sh", str(ROOT / "install.sh"), "--local-binary", str(path),
            "--sha256", valid_sha, "--project", "user-1001",
            "--apply", "--confirm-host-sha256", "0" * 64,
        ], text=True, capture_output=True)
        self.assertNotEqual(denied.returncode, 0)
        self.assertIn("exact --scoped-fresh-review-sha256", denied.stderr)

    def test_rc_publishing_requires_explicit_one_shot_branch(self):
        workflow = (ROOT / ".github/workflows/v1-release.yml").read_text()
        self.assertIn('"release/v1.*-rc.*"', workflow)
        self.assertIn("refs/heads/release/v1.", workflow)
        self.assertNotIn("workflow_dispatch", workflow)
        self.assertIn("v1-prerelease-native-S2-isolated-runner", workflow)
        self.assertIn("v1-prerelease-nested-S2.5-isolated-runner", workflow)
        self.assertIn("needs: [source, linux, native-s2-gate, native-s25-gate, candidate]", workflow)
        s2 = workflow.split("\n  native-s2-gate:\n", 1)[1].split("\n  native-s25-gate:\n", 1)[0]
        s25 = workflow.split("\n  native-s25-gate:\n", 1)[1].split("\n  candidate:\n", 1)[0]
        self.assertNotIn("sh scripts/accept_s2_native.sh", s2)
        self.assertIn("needs: [native-s25-gate, candidate]", s2)
        self.assertIn("actions: read", s2)
        self.assertIn("name: v1-release-candidate-manifest", s2)
        self.assertIn("sha256sum -c SHA256SUMS", s2)
        self.assertIn("python3 scripts/pr191_s2_ci_gate.py --release release-assets", s2)
        self.assertIn("P2_R4_SCOPED_RELEASE_S2_EXACT_CI_CROSSWALK=PASS", s2)
        self.assertIn("native_manifest_sha256: ", s2)
        publish = workflow.split("\n  publish:\n", 1)[1]
        self.assertIn("needs.native-s2-gate.outputs.native_manifest_sha256", publish)
        self.assertLess(publish.index('= "$EXPECTED_NATIVE_MANIFEST_SHA256"'),
                        publish.index('gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$INPUT_TAG"'))
        n2 = (ROOT / ".github/workflows/v1-n2-installed-lifecycle.yml").read_text()
        self.assertIn('branches: ["release/v1.*-rc.*"]', n2)
        self.assertIn("python3 scripts/pr191_s2_ci_gate.py --release-version", n2)
        self.assertIn('gh api "repos/$GITHUB_REPOSITORY/git/ref/heads/v1"', n2)
        self.assertNotIn("sh scripts/s25_bridgehead_lab.sh", s2)
        self.assertIn("sh scripts/s25_bridgehead_lab.sh", s25)
        self.assertNotIn("sh scripts/accept_s2_native.sh", s25)
        # The original packet/native evidence assertion moved into the
        # shared transcript verifier, which must still run after the lab.
        lab = 'sh scripts/s25_bridgehead_lab.sh 2>&1 | tee "$RUNNER_TEMP/native-s25-gate.log"'
        check = 'sh scripts/pr191_s25_evidence_check.sh "$RUNNER_TEMP/native-s25-gate.log" "$P1B_S25_EXACT_HEAD"'
        self.assertIn("set -euo pipefail", s25)
        self.assertIn("python3 scripts/test_pr191_s25_evidence_check.py", s25)
        self.assertLess(s25.index(lab), s25.index(check))
        verifier = (ROOT / "scripts/pr191_s25_evidence_check.sh").read_text()
        self.assertIn("grep -qx 'S25_SIMWAN6_BOTH_LAYER_RAW_PACKET_AND_NATIVE_EVIDENCE_ARCHIVED=PASS'", verifier)
        self.assertIn("runs-on: ubuntu-24.04", s2)
        self.assertIn("runs-on: ubuntu-24.04", s25)

    def exercise_guided_install(self, answer, drift=False, native_ready=True, missing_reader=False, reader_drift=False, plan_only=False, oauth_answer='n', oauth_origin='', oauth_error=False, explicit=False, status_override=None, doctor_override=None, existing_oauth=False):
        content = (ROOT / "install.sh").read_text()
        embedded = content.split("<<'PY_BOXWEAVE_INSTALL'\n", 1)[1].rsplit("\nPY_BOXWEAVE_INSTALL", 1)[0]
        ns = {}
        exec(compile(embedded.rsplit("\ntry:\n    main()", 1)[0], "guided-test", "exec"), ns)
        source = "b" * 40
        approval = "a" * 64
        candidate = b"fake reviewed release binary"
        target = dict(project="user-1001", pool="default", network="incusbr-1001", acl="bw-user-1001-acl")
        native_plan = {"Mode": "SCOPED_PEER_FRESH_V2", "Project": target["project"], "ReviewSHA256": approval, "NativeSHA256": "c" * 64, "GuardSHA256": "d" * 64}
        calls = []
        socket_path = Path(tempfile.mkdtemp(dir=self.dir)) / "incus.sock"
        sock = socket.socket(socket.AF_UNIX)
        sock.bind(str(socket_path))
        self.addCleanup(sock.close)

        def fake_path(raw):
            if str(raw) == "/var/lib/incus/unix.socket.user":
                # Validation sees the actual approved path string and an
                # isolated test socket's type; no real host file is read.
                p = mock.Mock(wraps=socket_path)
                p.__str__ = mock.Mock(return_value=str(raw))
                p.parts = Path(raw).parts
                p.is_absolute.return_value = True
                p.is_symlink.return_value = False
                p.exists.return_value = True
                p.stat.return_value = socket_path.stat()
                return p
            return self.dir / ("isolated-" + Path(raw).name)

        def output(argv, **kwargs):
            if argv[:5] == ["runuser", "-u", "boxweave", "--", "env"]:
                self.assertEqual(argv[5:13], ["-u", "INCUS_CONF", "-u", "INCUS_CONFIG_DIR",
                                                "-u", "XDG_CONFIG_HOME", "HOME=/home/boxweave",
                                                "XDG_CONFIG_HOME=/home/boxweave/.config"])
                self.assertEqual(argv[13:], [str(fake_path("/usr/local/bin/boxweave")),
                                              "doctor", "-project", target["project"]])
                action = argv[14]
            else:
                action = argv[1] if argv[0] != "runuser" else argv[5]
            calls.append(action)
            if action == "native-scoped-fresh-plan":
                if missing_reader or (reader_drift and calls.count(action) > 1):
                    raise ns["InstallError"]("INSTALL_MISSING_PREREQUISITE: /usr/sbin/conntrack")
                result = dict(native_plan)
                result["HostSHA256"] = hashlib.sha256(b"boxweave-native-host-v1|fixture-machine").hexdigest()
                if drift and calls.count(action) > 1:
                    result["ReviewSHA256"] = "f" * 64
                return json.dumps(result)
            if action == "install-status":
                return json.dumps({**{"binary_sha256": hashlib.sha256(candidate).hexdigest(),
                                   "native_project": target["project"], "service_user": "boxweave",
                                   "service_active": True, "restart_required": False,
                                   "version": "1.0.0-rc.8", "build_commit": source,
                                   "scoped_peer_mode": "SCOPED_PEER_FRESH_V2"}, **(status_override or {})})
            if action == "oauth-status":
                password = existing_oauth or "oauth-setup" in calls
                origin = ("https://existing.example.test" if existing_oauth else
                          oauth_origin if "oauth-set-origin" in calls and not oauth_error else "")
                return json.dumps({"Enabled": bool(origin) and password, "Origin": origin,
                                   "PasswordConfigured": password})
            if action == "doctor":
                return json.dumps({**{"ready": True, "mcp_ready": True, "native": {"ready": True},
                                   "version": "1.0.0-rc.8", "commit": source}, **(doctor_override or {})})
            raise AssertionError(argv)

        def run(argv, **kwargs):
            action = argv[1] if argv[0] != "runuser" else argv[5]
            calls.append(action)
            if action == "install-stage":
                self.assertIn(approval, argv)
                if not native_ready:
                    raise subprocess.CalledProcessError(1, argv)
            if oauth_error and action == "oauth-set-origin":
                raise subprocess.CalledProcessError(1, argv)
            return mock.Mock(returncode=0)

        checks = []
        def reader_preflight():
            checks.append(True)
            if missing_reader or (reader_drift and len(checks) > 1):
                raise ns["InstallError"]("INSTALL_MISSING_PREREQUISITE: /usr/sbin/conntrack")
        ns["SCOPED_PATHS"] = tuple(self.dir / ("scope-" + str(i))
                                   for i, _ in enumerate(ns["SCOPED_PATHS"]))
        ns["STATE_DIR"] = self.dir / "isolated-state"
        ns["retirement_reader_preflight"] = reader_preflight
        ns["Path"] = fake_path
        (self.dir / "isolated-machine-id").write_text("fixture-machine")
        ns["current_host_digest"] = lambda: "e" * 64
        ns["select_release"] = lambda *_: (candidate, "1.0.0-rc.8", source, hashlib.sha256(candidate).hexdigest())
        # The temporary candidate must use a real Path, not our host-path mapper.
        ns["Path"] = lambda raw: fake_path(raw) if str(raw).startswith(("/var/", "/usr/", "/etc/")) else Path(raw)
        mode = (["--apply", "--project", target["project"], "--confirm-host-sha256", "e" * 64,
                 "--scoped-fresh-review-sha256", approval] if explicit else
                ["--plan"] if plan_only else ["--interactive"])
        with mock.patch.object(ns["os"], "geteuid", return_value=0), \
                mock.patch.object(ns["os"].path, "lexists", return_value=False), \
                mock.patch.object(ns["pwd"], "getpwnam", return_value=types.SimpleNamespace(pw_uid=1001)), \
                mock.patch.object(ns["sys"], "argv", ["install.sh", "--version", "v1.0.0-rc.8"] + mode), \
                mock.patch.object(ns["subprocess"], "check_output", side_effect=output), \
                mock.patch.object(ns["subprocess"], "run", side_effect=run), \
                mock.patch("builtins.open", side_effect=[io.StringIO(answer + "\n"), io.StringIO(),
                                                         io.StringIO(oauth_answer + "\n" + oauth_origin + "\n"), io.StringIO()]), \
                contextlib.redirect_stdout(io.StringIO()):
            try:
                ns["main"]()
                return calls, None
            except ns["InstallError"] as exc:
                return calls, str(exc)
            except SystemExit as exc:
                return calls, "exit " + str(exc.code)

    def test_guided_install_approval_sequence_and_fail_closed(self):
        calls, error = self.exercise_guided_install("INSTALL SCOPED user-1001")
        self.assertIsNone(error)
        self.assertEqual(calls, ["native-scoped-fresh-plan", "native-scoped-fresh-plan", "install-stage",
                                 "install-activate", "install-status", "doctor", "oauth-status"])

    def test_explicit_apply_has_same_installed_postflight_without_oauth(self):
        calls, error = self.exercise_guided_install("NO", explicit=True)
        self.assertIsNone(error)
        self.assertEqual(calls, ["native-scoped-fresh-plan", "native-scoped-fresh-plan",
                                 "install-stage", "install-activate", "install-status", "doctor"])

    def test_both_install_modes_fail_closed_on_postflight_drift(self):
        for explicit in (False, True):
            for override in ({"binary_sha256": "0" * 64}, {"scoped_peer_mode": ""},
                             {"service_active": False}, {"restart_required": True}):
                with self.subTest(explicit=explicit, status=override):
                    calls, error = self.exercise_guided_install(
                        "INSTALL SCOPED user-1001", explicit=explicit, status_override=override)
                    self.assertIn("INSTALL_PHASE_UNVERIFIED: INSTALLED_STATUS", error)
                    self.assertEqual(calls[-1], "install-status")
            for override in ({"commit": "f" * 40}, {"ready": False}, {"native": {"ready": False}}):
                with self.subTest(explicit=explicit, doctor=override):
                    calls, error = self.exercise_guided_install(
                        "INSTALL SCOPED user-1001", explicit=explicit, doctor_override=override)
                    self.assertIn("INSTALL_PHASE_UNVERIFIED: CONTROLLER_DOCTOR", error)
                    self.assertEqual(calls[-1], "doctor")
                    self.assertNotIn("oauth-status", calls)

    def test_guided_install_decline_and_changed_plan(self):
        calls, error = self.exercise_guided_install("NO")
        self.assertIn("OPERATOR_APPROVAL_REQUIRED", error)
        self.assertIn("not approved", error)
        self.assertEqual(calls, ["native-scoped-fresh-plan"])
        calls, error = self.exercise_guided_install("INSTALL SCOPED user-1001", drift=True)
        self.assertIn("plan changed", error)
        self.assertNotIn("install-stage", calls)

    def test_missing_reader_and_reader_drift_never_stage(self):
        calls, error = self.exercise_guided_install("INSTALL SCOPED user-1001", missing_reader=True)
        self.assertIn("INSTALL_MISSING_PREREQUISITE", error)
        self.assertEqual(calls, ["native-scoped-fresh-plan"])
        calls, error = self.exercise_guided_install("INSTALL SCOPED user-1001", reader_drift=True)
        self.assertIn("INSTALL_MISSING_PREREQUISITE", error)
        self.assertEqual(calls, ["native-scoped-fresh-plan", "native-scoped-fresh-plan"])

    def test_complete_guided_preview_never_mutates(self):
        calls, error = self.exercise_guided_install("NO", plan_only=True)
        self.assertIsNone(error)
        self.assertEqual(calls, ["native-scoped-fresh-plan"])
        calls, error = self.exercise_guided_install("NO", plan_only=True, missing_reader=True)
        self.assertIn("INSTALL_MISSING_PREREQUISITE", error)
        self.assertEqual(calls, ["native-scoped-fresh-plan"])

    def test_oauth_deferred_and_failed_after_installed_postflight(self):
        calls, error = self.exercise_guided_install("INSTALL SCOPED user-1001", oauth_answer="y")
        self.assertIsNone(error)
        self.assertEqual(calls[-4:], ["doctor", "oauth-status", "oauth-setup", "oauth-status"])
        calls, error = self.exercise_guided_install("INSTALL SCOPED user-1001", oauth_answer="y", oauth_origin="https://example.test")
        self.assertIsNone(error)
        self.assertEqual(calls[-6:], ["doctor", "oauth-status", "oauth-setup", "oauth-status", "oauth-set-origin", "oauth-status"])
        calls, error = self.exercise_guided_install("INSTALL SCOPED user-1001", oauth_answer="y", oauth_origin="https://example.test", oauth_error=True)
        self.assertEqual(error, "exit 2")
        self.assertEqual(calls.count("install-activate"), 1)
        self.assertEqual(calls[-2:], ["oauth-set-origin", "oauth-status"])
        self.assertEqual(calls.count("oauth-setup"), 1)
        self.assertEqual(calls.count("oauth-set-origin"), 1)

    def test_guided_install_preserves_existing_oauth_password_and_origin(self):
        calls, error = self.exercise_guided_install("INSTALL SCOPED user-1001", existing_oauth=True)
        self.assertIsNone(error)
        self.assertEqual(calls[-2:], ["doctor", "oauth-status"])
        self.assertNotIn("oauth-setup", calls)
        self.assertNotIn("oauth-set-origin", calls)

    def test_guided_install_incomplete_native_never_activates(self):
        calls, error = self.exercise_guided_install("INSTALL SCOPED user-1001", native_ready=False)
        self.assertIn("INSTALL_PHASE_UNVERIFIED: STAGE", error)
        self.assertNotIn("install-activate", calls)

    def test_embedded_guided_installer_keeps_explicit_plan_and_approvals(self):
        source = (ROOT / "install.sh").read_text()
        self.assertIn("--interactive", source)
        self.assertIn('open("/dev/tty", "r")', source)
        self.assertIn('native plan changed after approval; review again', source)
        main = source.split("def main():", 1)[1]
        self.assertIn("legacy installation and upgrade unsupported", main)
        self.assertIn("scoped_review(candidate, args.project) != scoped", main)
        self.assertNotIn('"native-prepare-apply"', main)
        self.assertLess(main.index('install_phase("STAGE"'), main.index('install_phase("ACTIVATE"'))


    def test_s25_doctor_requires_initialized_and_activated_controller(self):
        # Current Scoped install helper uses the exact wrapper, not the
        # removed Legacy direct stage/token/activate sequence.
        wrapper = (ROOT / "install.sh").read_text().split("def main():", 1)[1]
        stage = wrapper.index('install_phase("STAGE"')
        activate = wrapper.index('install_phase("ACTIVATE"')
        status = wrapper.index('install_phase("INSTALLED_STATUS"')
        doctor = wrapper.index("doctor = doctor_postflight")
        self.assertLess(stage, activate)
        self.assertLess(activate, status)
        self.assertLess(status, doctor)
        self.assertIn('doctor.get("mcp_ready")', wrapper[doctor:])
        self.assertIn('doctor.get("native", {}).get("ready")', wrapper[doctor:])
        helper = (ROOT / "scripts/pr191_s25_scoped_setup.sh").read_text()
        apply = helper.index('sh "$wrapper" --apply --scoped-peer-fresh', helper.index(':apply"'))
        active = helper.index("systemctl is-active --quiet boxweave.service", apply)
        self.assertLess(apply, active)
        self.assertIn('s["service_active"] and s["restart_required"] is False', helper[active:])
        for name in ("s25_bridgehead_lab.sh", "s25_inner_setup.sh"):
            with self.subTest(layer=name):
                source = (ROOT / "scripts" / name).read_text()
                install = source.index("pr191_s25_scoped_setup.sh install")
                token = source.index(" token-create >", install)
                doctor = source.index(" doctor -project ", token)
                self.assertLess(install, token)
                self.assertLess(token, doctor)
                self.assertIn('assert s["ready"] and s["native"]["ready"]', source[doctor:])
                self.assertIn("nftables conntrack", source)

    def test_rc_flag_not_required_and_removed(self):
        script = (ROOT / "install.sh").read_text()
        self.assertNotIn("allow-prerelease", script)

    def test_embedded_installer_python_compiles(self):
        content = (ROOT / "install.sh").read_text()
        self.assertIn("PY_BOXWEAVE_INSTALL", content)
        embedded = content.split("<<'PY_BOXWEAVE_INSTALL'\n", 1)[1].rsplit("\nPY_BOXWEAVE_INSTALL", 1)[0]
        compile(embedded, "install.sh:python", "exec")


if __name__ == "__main__":
    unittest.main()

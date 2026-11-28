"""Current fresh installer and retained refusal/metadata tests; historical upgrade tests remain in Git."""
import io
import json
from pathlib import Path
import tempfile
import unittest
import sys
import stat
from unittest.mock import patch, Mock

SOURCE = Path(__file__).resolve().parents[1] / "install.sh"
NS = {}
code = SOURCE.read_text().split("<<'PY_BOXWEAVE_INSTALL'\n", 1)[1].rsplit("\ntry:\n    main()", 1)[0]
exec(compile(code, str(SOURCE), "exec"), NS)
ERR = NS["InstallError"]
SHA, OLD, PREV = "a"*64, "b"*64, "c"*64

class FreshInstallerTests(unittest.TestCase):
    def test_inherited_dropin_preflight_is_advisory_and_read_only(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            generic = root / "service.d"
            generic.mkdir()
            hint = NS["inherited_service_dropin_hint"]
            self.assertEqual(hint([generic, root / "absent"]), "NO_GLOBAL_HINT_NOT_EFFECTIVE_PROOF")
            policy = generic / "zzz-lxc-service.conf"
            policy.write_bytes(b"[Service]\nNoNewPrivileges=no\n")
            before = policy.read_bytes()
            self.assertEqual(hint([generic]), "POTENTIAL_INHERITED_SERVICE_DROPIN")
            self.assertEqual(policy.read_bytes(), before)
            self.assertEqual(hint([policy]), "INHERITED_SERVICE_DROPIN_UNVERIFIED")

    def test_known_lxc_hint_is_exact_and_unknown_additions_are_not_recognized(self):
        with tempfile.TemporaryDirectory() as folder:
            policy = Path(folder) / "zzz-lxc-service.conf"
            content = b"[Service]\nNoNewPrivileges=no\n"
            policy.write_bytes(content)
            hint = NS["inherited_service_dropin_hint"]
            with patch.dict(NS, {"KNOWN_LXC_POLICY_PATH": policy,
                                 "KNOWN_LXC_POLICY_SHA": NS["digest"](content)}):
                self.assertEqual(hint([policy.parent]), "KNOWN_LXC255_POLICY_BYTES_REVIEW_REQUIRED")
                extra = policy.with_name("extra.conf")
                extra.write_text("[Service]\n")
                self.assertEqual(hint([policy.parent]), "POTENTIAL_INHERITED_SERVICE_DROPIN")
                extra.unlink()
                policy.write_bytes(content+b"# changed\n")
                self.assertEqual(hint([policy.parent]), "POTENTIAL_INHERITED_SERVICE_DROPIN")
                policy.unlink()
                other = policy.with_name("other")
                other.write_bytes(content)
                policy.symlink_to(other)
                self.assertEqual(hint([policy.parent]), "POTENTIAL_INHERITED_SERVICE_DROPIN")
                self.assertEqual(other.read_bytes(), content)

    def test_known_lxc_preflight_distinguishes_scoped_and_legacy_without_approval(self):
        from contextlib import redirect_stdout
        from io import StringIO
        with patch.dict(NS, {"inherited_service_dropin_hint": lambda: "KNOWN_LXC255_POLICY_BYTES_REVIEW_REQUIRED"}):
            out = StringIO()
            with redirect_stdout(out):
                NS["print_lifecycle_preflight"](scoped=True)
            self.assertIn("reviewed compatibility route", out.getvalue())
            self.assertIn("not a runtime-hardening or uninstall approval", out.getvalue())
            out = StringIO()
            with redirect_stdout(out):
                NS["print_lifecycle_preflight"](scoped=False)
            self.assertIn("may block verified activation (Legacy)", out.getvalue())
            self.assertNotIn("reviewed compatibility route", out.getvalue())

    def test_scoped_residue_refuses_even_dangling_links(self):
        with tempfile.TemporaryDirectory() as folder:
            paths = tuple(Path(folder) / str(i) for i in range(5))
            with patch.dict(NS, {"SCOPED_PATHS": paths, "STATE_DIR": Path(folder) / "state"}):
                NS["refuse_scoped_existing"]()
                for path in paths:
                    path.symlink_to(path.with_name("missing"))
                    with self.assertRaisesRegex(ERR, "scoped installation evidence exists"):
                        NS["refuse_scoped_existing"]()
                    path.unlink()
                with patch.object(Path, "lstat", side_effect=PermissionError("unreadable")):
                    with self.assertRaisesRegex(ERR, "unreadable"):
                        NS["refuse_scoped_existing"]()

    def test_scoped_manifest_only_refuses_before_candidate_execution(self):
        with tempfile.TemporaryDirectory() as folder:
            state = Path(folder)
            manifest = state / "install-manifest.json"
            manifest.write_text("{}")
            with patch.dict(NS, {"SCOPED_PATHS": (), "STATE_DIR": state,
                                 "private_json": Mock(return_value=({"scoped_peer_mode": "SCOPED_PEER_FRESH_V2"}, SHA))}), patch("os.geteuid", return_value=0), patch("pwd.getpwnam", return_value=Mock(pw_uid=1001)):
                with self.assertRaisesRegex(ERR, "scoped manifest"):
                    NS["refuse_scoped_existing"]()

    def test_scoped_review_binds_native_host_and_mode(self):
        machine = "0123456789abcdef" * 2
        native_host = NS["digest"](("boxweave-native-host-v1|" + machine).encode())
        review = dict(Mode="SCOPED_PEER_FRESH_V2", Project="user-1001",
                      HostSHA256=native_host, ReviewSHA256=SHA,
                      GuardSHA256=OLD, NativeSHA256=PREV)
        with patch.object(Path, "read_text", return_value=machine), patch.dict(NS, {"command_json": Mock(return_value=review)}):
            self.assertEqual(NS["scoped_review"](Path("/candidate"), "user-1001"), review)
            for key, val in (("Mode", "legacy"), ("Project", "user-9999"),
                             ("HostSHA256", SHA), ("ReviewSHA256", "bad"),
                             ("NativeSHA256", ""), ("GuardSHA256", "")):
                with self.subTest(key=key):
                    NS["command_json"].return_value = dict(review, **{key: val})
                    with self.assertRaisesRegex(ERR, "mismatch"):
                        NS["scoped_review"](Path("/candidate"), "user-1001")

    def test_scoped_plan_calls_native_readonly_before_apply(self):
        review = dict(ReviewSHA256=SHA)
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "candidate"
            path.write_bytes(b"test fixture")
            binary_sha = NS["digest"](path.read_bytes())
            argv = ["install.sh", "--plan", "--scoped-peer-fresh",
                    "--local-binary", str(path), "--sha256", binary_sha]
            reader = Mock(return_value=review)
            with patch.object(sys, "argv", argv), patch("os.geteuid", return_value=0), patch("pwd.getpwnam", return_value=Mock(pw_uid=1001)), patch.dict(NS, {
                "STATE_DIR": Path(folder) / "unused-state",
                "refuse_scoped_existing": Mock(),
                "current_host_digest": Mock(return_value=OLD),
                "scoped_review": reader,
            }), patch("subprocess.run") as run, patch("sys.stdout", new_callable=io.StringIO) as out:
                NS["main"]()
                reader.assert_called_once()
                self.assertEqual(reader.call_args.args[1], "user-1001")
                self.assertIn("scoped_fresh_review_sha256=" + SHA, out.getvalue())
                self.assertIn("PLAN_ONLY:", out.getvalue())
                run.assert_not_called()

    def test_retirement_reader_preflight_matches_go_fixed_paths(self):
        paths = [p for p, _ in NS["RETIREMENT_READERS"]]
        self.assertEqual(paths, ["/usr/sbin/ip", "/usr/sbin/bridge", "/usr/sbin/nft", "/usr/sbin/conntrack"])
        source = (SOURCE.parent / "internal/install/local.go").read_text()
        body = source.split("func verifyRetirementExecutables()", 1)[1]
        self.assertIn(", ".join('"' + p + '"' for p in paths), body)
        good = Mock(st_uid=0, st_mode=stat.S_IFREG | 0o755)
        self.assertTrue(NS["retirement_reader_preflight"](lambda _: good))
        for bad in (Mock(st_uid=1001, st_mode=stat.S_IFREG | 0o755),
                    Mock(st_uid=0, st_mode=stat.S_IFREG | 0o775),
                    Mock(st_uid=0, st_mode=stat.S_IFREG | 0o644),
                    Mock(st_uid=0, st_mode=stat.S_IFDIR | 0o755)):
            with self.assertRaisesRegex(ERR, "INSTALL_PREREQUISITE_UNVERIFIED"):
                NS["retirement_reader_preflight"](lambda _: bad)
        def missing(path):
            if path.endswith(("conntrack", "nft")):
                raise FileNotFoundError(path)
            return good
        with self.assertRaisesRegex(ERR, "INSTALL_MISSING_PREREQUISITE.*nftables.*conntrack"):
            NS["retirement_reader_preflight"](missing)
        with self.assertRaisesRegex(ERR, "INSTALL_PREREQUISITE_UNVERIFIED"):
            NS["retirement_reader_preflight"](Mock(side_effect=PermissionError()))

    def test_private_metadata_rejects_symlink_and_bad_mode(self):
        with tempfile.TemporaryDirectory() as folder:
            p = Path(folder)/"receipt.json"
            p.write_text("{}")
            p.chmod(0o600)
            self.assertEqual(NS["private_json"](p, p.stat().st_uid)[0], {})
            link = p.with_name("link")
            link.symlink_to(p)
            with self.assertRaises(OSError):
                NS["private_json"](link, p.stat().st_uid)
            p.chmod(0o644)
            with self.assertRaises(ERR):
                NS["private_json"](p, p.stat().st_uid)

if __name__ == "__main__":
    unittest.main()

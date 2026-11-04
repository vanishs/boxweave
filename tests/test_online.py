"""Non-root contract checks for streamed release, SSH trust and client enrollment."""
import base64,hashlib,json,os,struct,tempfile,time,unittest
from pathlib import Path
from unittest.mock import patch
from bwmini import online,hostkeys,mcp_gate
from scripts.build_release_manifest import build

class StreamedInstallTests(unittest.TestCase):
    def key(self):
        blob=struct.pack(">I",11)+b"ssh-ed25519"+struct.pack(">I",32)+b"x"*32
        return "ssh-ed25519 "+base64.b64encode(blob).decode()+" test-public"

    def test_public_keys_require_full_valid_ed25519_not_fingerprints(self):
        self.assertTrue(online.public_key(self.key()).startswith("ssh-ed25519 "))
        for value in ("SHA256:fingerprint","ssh-ed25519 AAAA","not-a-key"):
            with self.subTest(value=value),self.assertRaises((online.Failure,ValueError)):online.public_key(value)

    def test_application_mcp_resolves_after_fresh_bootstrap_not_release_asset(self):
        with tempfile.TemporaryDirectory() as directory:
            mcp = Path(directory) / "mcp-server"
            with self.assertRaisesRegex(online.Failure, "INSTALLED_MCP_SERVER_UNTRUSTED_OR_MISSING"):
                online.application_mcp_binary(mcp)
            mcp.write_bytes(b"fixture-binary")
            mcp.chmod(0o755)
            with patch.object(online, "trusted_provider_binary", return_value=True):
                self.assertEqual(online.application_mcp_binary(mcp), str(mcp))
            with patch.object(online, "trusted_provider_binary", return_value=False):
                with self.assertRaisesRegex(online.Failure, "INSTALLED_MCP_SERVER_UNTRUSTED_OR_MISSING"):
                    online.application_mcp_binary(mcp)

    def test_server_adds_workload_key_when_client_omits_it(self):
        extra = "ssh-ed25519 " + base64.b64encode(
            struct.pack(">I",11)+b"ssh-ed25519"+struct.pack(">I",32)+b"y"*32).decode()
        with tempfile.TemporaryDirectory() as folder:
            path=Path(folder)/"id_ed25519.pub"
            path.write_text(self.key()+"\n")
            path.chmod(0o644)  # Avoid test outcome depending on developer umask.
            with patch.object(mcp_gate,"MANAGED_PUBLIC_KEY_FILE",path):
                managed=mcp_gate.normalized_public_key(self.key())
                for input_args in ({}, {"ssh_keys":[]}, {"ssh_keys":[extra]},
                                   {"ssh_keys":[managed]}, {"ssh_keys":[managed,extra]}):
                    with self.subTest(input_args=input_args):
                        result=mcp_gate.managed_create_arguments(input_args)
                        self.assertEqual(result["ssh_keys"].count(managed),1)
                self.assertEqual(mcp_gate.managed_create_arguments({"ssh_keys":[extra]})["ssh_keys"],
                                 [extra,managed])
                for bad in (None, {"ssh_keys":None}, {"ssh_keys":["SHA256:invalid"]},
                            {"ssh_keys":[extra,extra]}, {"ssh_keys":[extra]*17}):
                    with self.subTest(bad=bad), self.assertRaises(ValueError):
                        mcp_gate.managed_create_arguments(bad)
                many=["ssh-ed25519 "+base64.b64encode(
                    struct.pack(">I",11)+b"ssh-ed25519"+struct.pack(">I",32)+bytes([i])*32).decode()
                    for i in range(1,17)]
                with self.assertRaisesRegex(ValueError,"TOO_MANY_PUBLIC_KEYS"):
                    mcp_gate.managed_create_arguments({"ssh_keys":many})
                path.chmod(0o666)
                with self.assertRaisesRegex(ValueError,"MANAGED_IDENTITY_UNAVAILABLE"):
                    mcp_gate.managed_create_arguments({})
                path.chmod(0o644);path.unlink();path.symlink_to(Path(folder)/"missing")
                with self.assertRaisesRegex(ValueError,"MANAGED_IDENTITY_UNAVAILABLE"):
                    mcp_gate.managed_create_arguments({})

    def test_catalog_no_longer_requires_managed_key_from_client(self):
        sample={"result":{"tools":[{"name":"create_container",
            "inputSchema":{"properties":{"ssh_keys":{"type":"array"}},
                           "required":["username","ssh_keys"]}}]}}
        annotated=mcp_gate.catalog(sample)
        schema=annotated["result"]["tools"][0]["inputSchema"]
        self.assertEqual(schema["required"],["username"])
        self.assertIn("automatically supplies",schema["properties"]["ssh_keys"]["description"])

    def test_connect_gate_requires_fresh_exact_incarnation_and_ip(self):
        target={"name":"work-container","created":"123","user":"work","ip":"10.203.0.2","keys":[self.key()]}
        registry={"observed_at":time.time(),"targets":{"work":target}}
        observed={"name":"work-container","createdAt":"123","username":"work","network":{"ipAddress":"10.203.0.2"},"state":"CONTAINER_STATE_RUNNING"}
        mcp_gate.trusted({"box":"work"},registry,observed)
        for change in ({"createdAt":"124"},{"network":{"ipAddress":"10.203.0.3"}},{"state":"CONTAINER_STATE_STOPPED"}):
            with self.subTest(change=change),self.assertRaises(ValueError):mcp_gate.trusted({"box":"work"},registry,observed|change)
        for args in ({"box":"other"},{"box":"work","host":"example.com"},{"box":"work","user":"root"}):
            with self.subTest(args=args),self.assertRaises(ValueError):mcp_gate.trusted(args,registry,observed)
        with self.assertRaises(ValueError):mcp_gate.trusted({"box":"work"},registry|{"observed_at":time.time()-60},observed)

    def test_connect_alias_requires_exact_registry_and_no_collision(self):
        target={"name":"work-container","created":"123","user":"work",
                "ip":"10.203.0.2","keys":[self.key()]}
        registry={"observed_at":time.time(),"targets":{"work":target}}
        self.assertEqual(mcp_gate.resolve_managed_box("work",registry),"work")
        self.assertEqual(mcp_gate.resolve_managed_box("work-container",registry),"work")
        for value in ("missing", "missing-container", "work-container-container"):
            with self.subTest(value=value),self.assertRaisesRegex(ValueError,"TRUST_NOT_READY"):
                mcp_gate.resolve_managed_box(value,registry)
        for value in ("", "bad/name", "work_container", 42):
            with self.subTest(value=value),self.assertRaisesRegex(ValueError,"EXACT_MANAGED_TARGET_REQUIRED"):
                mcp_gate.resolve_managed_box(value,registry)
        collision={**registry,"targets":{**registry["targets"],
            "work-container":{**target,"name":"work-container-container","user":"work-container"}}}
        with self.assertRaisesRegex(ValueError,"BOX_TARGET_AMBIGUOUS"):
            mcp_gate.resolve_managed_box("work-container",collision)

    def test_trusted_registry_and_network_type_fail_closed(self):
        target={"name":"work-container","created":"123","user":"work",
                "ip":"10.203.0.2","keys":[self.key()]}
        observed={"name":"work-container","createdAt":"123","username":"work",
                  "network":{"ipAddress":"10.203.0.2"},"state":"CONTAINER_STATE_RUNNING"}
        registry={"observed_at":time.time(),"targets":{"work":target}}
        for value in ("bad", None, True):
            with self.subTest(value=value),self.assertRaisesRegex(ValueError,"TRUST_REGISTRY_UNAVAILABLE"):
                mcp_gate.trusted({"box":"work"},{**registry,"observed_at":value},observed)
        with self.assertRaisesRegex(ValueError,"TARGET_CHANGED"):
            mcp_gate.trusted({"box":"work"},registry,observed|{"network":None})
        with self.assertRaisesRegex(ValueError,"TARGET_OVERRIDE_REFUSED"):
            mcp_gate.trusted({"box":"work","host":"10.203.0.3"},registry,observed)

    def test_hostkey_inventory_excludes_core_and_unmanaged_instances(self):
        normal={"name":"work-container","status":"Running","config":{"user.containarium.label.os-type":"ubuntu_2404"}}
        self.assertTrue(hostkeys.eligible(normal))
        for change in ({"name":"containarium-core-postgres"},{"config":{}},{"status":"Stopped"},{"name":"other"}):
            self.assertFalse(hostkeys.eligible(normal|change))

    def test_release_manifest_covers_every_root_helper_and_exact_archive(self):
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder)/"root";out=Path(folder)/"out";(root/"bwmini").mkdir(parents=True);out.mkdir()
            (root/"VERSION").write_text("0.15.0\n");(root/"install.sh").write_text("fixture")
            for name in online.RELEASE_FILES:
                if name=="boxweave.zip":(out/name).write_bytes(b"reviewed-app")
                else:(root/"bwmini"/name).write_text("public-"+name)
            manifest=build(root,out,"a"*40)
            self.assertEqual(set(manifest["files"]),set(online.RELEASE_FILES))
            self.assertEqual(online.manifest_read(out/"release.json","0.15.0"),manifest)
            manifest["files"]["unexpected.py"]="0"*64;(out/"release.json").write_text(json.dumps(manifest))
            with self.assertRaises(online.Failure):online.manifest_read(out/"release.json","0.15.0")

    def test_download_never_publishes_unverified_bytes(self):
        with tempfile.TemporaryDirectory() as folder:
            path=Path(folder)/"artifact"
            with self.assertRaises(online.Failure):online.download("http://example.com/file",path,"0"*64)
            with self.assertRaises(online.Failure):online.download("https://example.com/file",path,"bad")
            path.write_bytes(b"changed")
            with self.assertRaises(online.Failure):online.download("https://example.com/file",path,"0"*64)
            self.assertEqual(path.read_bytes(),b"changed")

    def test_unmanaged_postgres_is_refused_before_any_mutation(self):
        with tempfile.TemporaryDirectory() as folder:
            base=Path(folder);etc=base/'etc';data=base/'data';etc.mkdir();data.mkdir()
            (etc/'foreign-cluster').mkdir()
            with patch.object(online,'PG_ETC',etc),patch.object(online,'PG_DATA',data),patch.object(online,'atomic') as write,patch.object(online,'run') as run:
                with self.assertRaisesRegex(online.Failure,'FOREIGN_POSTGRES_CONFIGURATION'):online.prepare_postgres({},time.monotonic()+600)
                write.assert_not_called();run.assert_not_called()

    def test_nonroot_install_stops_before_creating_host_state(self):
        with patch.object(online.os,"geteuid",return_value=1000),patch.object(online,"directory") as mutate:
            with self.assertRaisesRegex(online.Failure,"ROOT_REQUIRED"):online.install(Path("/unused"),"0.15.0")
            mutate.assert_not_called()

class IncusPackageResolutionTests(unittest.TestCase):
    def test_new_incus_uses_apt_solver_without_partial_version_pins(self):
        calls=[]
        with patch.object(online, "run", side_effect=lambda args,*_rest: calls.append(args)):
            online.install_missing_incus(1234, None, {"DEBIAN_FRONTEND":"noninteractive"})
        self.assertEqual(len(calls),2)
        self.assertEqual(calls[0],["apt-get","-s","install","-y","--no-upgrade",
                                   "incus","incus-client","incus-extra"])
        self.assertEqual(calls[1],["apt-get","install","-y","-qq","--no-upgrade",
                                   "incus","incus-client","incus-extra"])
        self.assertNotIn("incus-base", online.INCUS_PACKAGES)

    def test_failed_apt_simulation_prevents_package_installation(self):
        calls=[]
        def fail_first(args,*_rest):
            calls.append(args)
            raise online.Failure("APT_RESOLUTION_FAILED")
        with patch.object(online,"run",side_effect=fail_first):
            with self.assertRaisesRegex(online.Failure,"APT_RESOLUTION_FAILED"):
                online.install_missing_incus(1234,None,{})
        self.assertEqual(len(calls),1)
        self.assertIn("-s",calls[0])


class InstallCompletionTextTests(unittest.TestCase):
    def test_first_time_client_is_prominent_other_steps_are_optional(self):
        summary = online.installation_summary(18640, "0.18.2")
        self.assertIn("BoxWeave v0.18.2 installed successfully", summary)
        self.assertIn("Configure your client", summary)
        self.assertRegex(summary, r"(?m)^  boxweave client$")
        self.assertRegex(summary, r"(?m)^  boxweave client --help  # More options$")
        self.assertLess(summary.index("  boxweave client\n"), summary.index("Optional operations:"))
        self.assertIn("http://127.0.0.1:18640", summary)
        self.assertIn("ssh -N -L 18640:127.0.0.1:18640", summary)
        self.assertIn("/home/boxweave/boxweave/initial-admin-password", summary)
        self.assertIn("administrator approval", summary)
        self.assertIn("BOXWEAVE_BATCH", summary)
        self.assertIn("--no-enroll", summary)
        self.assertIn("--already-enrolled", summary)
        self.assertIn("online.py enroll --public-key KEY", summary)
        self.assertNotRegex(summary, r"(?m)^\\s*[1-9]\\.\\s")


class HttpMcpInstallSummaryTests(unittest.TestCase):
    def test_default_http_mcp_quickstart_is_in_human_success_screen(self):
        text = online.installation_summary(18640, "0.27.2")
        self.assertIn("HTTP MCP (authenticated, loopback only)", text)
        self.assertIn("Local MCP URL: http://127.0.0.1:18844/mcp", text)
        self.assertIn("Local info page: http://127.0.0.1:18844/", text)
        self.assertIn("boxweave http status", text)
        self.assertIn("sudo boxweave http doctor", text)
        self.assertIn("cloudflared tunnel --url http://127.0.0.1:18844", text)
        self.assertIn("sudo boxweave http configure --public-url", text)
        self.assertIn("https://YOUR_HOST.trycloudflare.com/mcp", text)
        self.assertIn("Finish OAuth", text)
        self.assertIn("NOT enabled by default", text)
        self.assertLess(text.index("HTTP MCP (authenticated"), text.index("Optional operations:"))
        self.assertIn("Configure your client", text)
        self.assertIn("Web dashboard (server-local only)", text)

    def test_custom_local_port_propagates_into_tunnel_and_url(self):
        text = online.installation_summary(18640, "0.27.2",
                                           {"enabled": True, "port": 19984})
        self.assertIn("http://127.0.0.1:19984/mcp", text)
        self.assertIn("cloudflared tunnel --url http://127.0.0.1:19984", text)
        self.assertNotIn("18844", text)

    def test_preconfigured_https_displays_exact_client_mcp_url_without_tunnel_instructions(self):
        text = online.installation_summary(18640, "0.27.2",
                                           {"enabled": True, "port": 18844,
                                            "public_url": "https://mcp.example.test"})
        self.assertIn("Configured public MCP URL: https://mcp.example.test/mcp", text)
        self.assertIn("Public HTTPS/Tunnel availability is NOT verified", text)
        self.assertNotIn("cloudflared tunnel --url", text)
        self.assertNotIn("YOUR_HOST.trycloudflare.com", text)
        self.assertNotIn("https://mcp.example.test/mcp/mcp", text)

    def test_explicitly_disabled_does_not_suggest_running_or_exposing_mcp(self):
        text = online.installation_summary(18640, "0.27.2",
                                           {"enabled": False, "port": 18844})
        self.assertIn("Disabled by remote_mcp.enabled=false", text)
        self.assertNotIn("Local MCP URL:", text)
        self.assertNotIn("cloudflared tunnel", text)
        self.assertNotIn("boxweave http restart", text)
        self.assertNotIn("sudo boxweave http configure --public-url", text)

    def test_version_validation_and_no_implied_live_success(self):
        for version in ("0.27.2\\nTOKEN", "0.27.2-rc1", "invalid"):
            with self.subTest(version=version), self.assertRaises(online.Failure):
                online.installation_summary(18640, version)
        output = online.installation_summary(18640, "0.27.2")
        self.assertIn("installation does not prove live reachability", output)
        self.assertNotIn("initial-admin-password:", output)


class InstallProvenanceTests(unittest.TestCase):
    def test_provenance_is_public_shape_without_secrets(self):
        old = {"source": "a" * 40, "version": "0.21.0", "provider_mode": "standalone",
               "managed_files": {"/usr/local/bin/boxweave": "d" * 64},
               "phase_history": [{"name": "release_download", "started_utc": "2026-10-31T01:02:03+00:00"},
                                  {"name": "local_provider", "started_utc": "2026-10-31T01:03:04+00:00"}]}
        descriptor = {"version": "0.21.0", "host": "10.0.0.5", "port": 22, "web_port": 18640,
                      "application_user": "boxweave"}
        record = online.build_provenance(old, descriptor, "2026-10-31T01:09:09+00:00")
        self.assertEqual(record["schema_version"], 1)
        self.assertEqual(record["source"], "a" * 40)
        self.assertEqual(record["managed_files"], 1)
        self.assertEqual([p["name"] for p in record["phases"]], ["release_download", "local_provider"])
        self.assertEqual(record["installed_utc"], "2026-10-31T01:09:09+00:00")
        self.assertEqual(set(record), {"schema_version", "version", "source", "installed_utc",
                                       "provider_mode", "host", "port", "web_port",
                                       "application_user", "managed_files", "phases"})
        blob = json.dumps(record)
        self.assertNotIn("secret", blob)
        self.assertNotIn("password", blob)


if __name__=="__main__":unittest.main()

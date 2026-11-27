"""Installed status and Doctor invalid-ledger witness; no live DB writes."""
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import pwd
import sqlite3
import subprocess
import sys
import tempfile

from pr188_h1a_observe import snapshot
from pr191_scoped_ack_loss import guard_identity, native_names
from pr191_scoped_lifecycle_recovery import rows
from pr178_n2_scoped_lifecycle import acls
from n2_installed_lifecycle import pid


def require_schema7(versions):
    assert versions == [(7,)], "installed ledger must have exactly schema 7"


def fixtures(directory, version):
    for name in ("foreign", "corrupt", "old", "future", "partial", "ambiguous", "versioned-foreign"):
        path = directory / (name + ".sqlite")
        assert not path.exists()
        if name == "corrupt":
            path.write_bytes(b"not a SQLite database; preserve exactly")
        else:
            with closing(sqlite3.connect(path)) as db:
                if name == "foreign":
                    db.execute("CREATE TABLE foreign_marker(proof TEXT)")
                    db.execute("INSERT INTO foreign_marker VALUES ('do-not-adopt')")
                else:
                    db.execute("CREATE TABLE schema_version(version INTEGER NOT NULL)")
                    db.execute("INSERT INTO schema_version VALUES (?)",
                               (version-1 if name == "old" else version+1 if name == "future" else version,))
                    if name == "ambiguous":
                        db.execute("INSERT INTO schema_version VALUES (?)", (version,))
                    if name == "versioned-foreign":
                        db.execute("CREATE TABLE outsider(proof TEXT)")
                db.commit()
        path.chmod(0o600)
        yield name, path


STATUS_REFUSAL = {
    "foreign": "existing database missing version: refuse foreign adoption",
    "corrupt": "file is not a database",
    "old": "unsupported schema 6 (expected 7):",
    "future": "unsupported schema 8 (expected 7):",
    "partial": "unexpected SQLite schema/migration tables: refuse foreign adoption",
    "ambiguous": "ambiguous SQLite migration history: 2 version rows",
    "versioned-foreign": "unexpected SQLite schema/migration tables: refuse foreign adoption",
}


def unchanged(path, expected):
    assert path.read_bytes() == expected, "invalid ledger modified"
    assert not any(path.with_name(path.name+suffix).exists()
                   for suffix in ("-wal","-shm","-journal",".exec-archive",".serve.lock"))


def status_refused(result, name, path, expected):
    assert result.returncode == 1 and not result.stdout.strip(), "status accepted or did not classify invalid ledger"
    assert result.stderr.strip().startswith("boxweave: " + STATUS_REFUSAL[name]), "status refused for unrelated reason"
    unchanged(path, expected)


def refused(result, path, expected):
    assert result.returncode != 0, "Doctor accepted invalid ledger"
    report = json.loads(result.stdout)
    database = report["diagnostic"]["database"]
    assert report["ready"] is False and database["state"] == "UNVERIFIED"
    assert database["reason"] == "DIAGNOSTIC_DB_SCHEMA_OR_INTEGRITY_UNVERIFIED"
    unchanged(path, expected)


def main(project, candidate_sha):
    assert os.geteuid()==0 and os.environ.get("GITHUB_ACTIONS")=="true"
    assert os.environ.get("RUNNER_ENVIRONMENT")=="github-hosted"
    account = pwd.getpwnam("boxweave")
    assert project=="user-"+str(account.pw_uid)
    binary = Path("/usr/local/bin/boxweave")
    assert hashlib.sha256(binary.read_bytes()).hexdigest()==candidate_sha
    status = json.loads(subprocess.check_output([str(binary),"install-status","-project",project],text=True,timeout=30))
    assert status["scoped_peer_mode"]=="SCOPED_PEER_FRESH_V2" and status["service_active"]
    assert not native_names() and acls()=={"bw-peer-quarantine"}
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','OBSERVING','UNKNOWN')")
    original, guard = snapshot(project), guard_identity()
    units = ('incus.service','boxweave-scoped-lifecycle.service')
    pids = [pid(unit) for unit in units]
    def history():
        return [rows("SELECT * FROM "+t+" ORDER BY rowid") for t in
                ("operations","operation_events","operation_resource_claims","recovery_audit","owned_boxes","network_bindings")]
    before = history()
    versions = rows("SELECT version FROM schema_version")
    require_schema7(versions)
    with tempfile.TemporaryDirectory(prefix="pr191-schema-",dir="/tmp") as temp:
        directory = Path(temp)
        cases = list(fixtures(directory,versions[0][0]))
        for _,path in cases:
            os.chown(path,account.pw_uid,account.pw_gid)
        os.chown(directory,account.pw_uid,account.pw_gid)
        for name,path in cases:
            expected = path.read_bytes()
            result = subprocess.run(["runuser","-u","boxweave","--",str(binary),
                                     "status","-project",project,"-db",str(path),
                                     "-incus-socket",str(directory/"absent-native.sock")],
                                    text=True,capture_output=True,timeout=40)
            status_refused(result,name,path,expected)
            print("P1B_SCOPED_STATUS_SCHEMA_REFUSED=PASS",name,flush=True)
            result = subprocess.run(["runuser","-u","boxweave","--",str(binary),
                                     "doctor","-project",project,"-db",str(path)],
                                    text=True,capture_output=True,timeout=40)
            refused(result,path,expected)
            print("P1B_SCOPED_SCHEMA_REFUSED=PASS",name,flush=True)
        assert set(directory.iterdir())=={path for _,path in cases}
    assert snapshot(project)==original and history()==before
    assert [pid(unit) for unit in units]==pids
    assert guard_identity()==guard and not native_names() and acls()=={"bw-peer-quarantine"}
    print("P1B_SCOPED_INSTALLED_SCHEMA7_EXACT=PASS",flush=True)
    print("S2_NATIVE_SQLITE_FOREIGN_CORRUPT_FAIL_CLOSED=PASS",flush=True)
    print("S2_NATIVE_SQLITE_OLD_MULTIPLE_VERSION_AND_FOREIGN_TABLES_DENIED=PASS",flush=True)
    print("P1B_SCOPED_SCHEMA_FIXED_INSTALLED_NO_EFFECT=PASS",flush=True)


if __name__=="__main__":
    main(*sys.argv[1:])

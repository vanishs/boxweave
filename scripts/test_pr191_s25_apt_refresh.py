"""Calibrate bounded signed-index refresh with fake APT, never install packages."""
import os
import pathlib
import subprocess
import tempfile
import unittest

SOURCE = pathlib.Path(__file__).with_name("s25_inner_setup.sh").read_text()
FUNCTION = SOURCE.split("s25_update_indices() {", 1)[1].split("\ns25_update_indices\n", 1)[0]

class IndexRefresh(unittest.TestCase):
    def check(self, sequence, expected, attempts):
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp)
            count = path / "count"
            count.write_text("0")
            sequence_file = path / "sequence"
            sequence_file.write_text("\n".join(sequence) + "\n")
            apt = path / "apt-get"
            apt.write_text(r'''#!/bin/sh
set -eu
    test "$*" = "-o APT::Update::Error-Mode=any -o Acquire::By-Hash=true -o Acquire::Retries=2 update -qq"
    n=$(cat "$COUNT")
    n=$((n+1))
    echo "$n" > "$COUNT"
    value=$(sed -n "${n}p" "$SEQUENCE")
    test -n "$value"
    if [ "$value" = OK ]; then exit 0; fi
    printf '%s\n' "$value" >&2
    exit 100
''' )
            apt.chmod(0o700)
            script = r'''
set -eu
sleep() { test "$*" = 10; }
s25_update_indices() {''' + FUNCTION + "\ns25_update_indices\n"
            env = dict(os.environ, PATH=str(path)+":"+os.environ["PATH"], COUNT=str(count), SEQUENCE=str(sequence_file))
            run = subprocess.run(["sh", "-c", script], env=env, capture_output=True, text=True, timeout=10)
            self.assertEqual(run.returncode, expected, run.stderr)
            self.assertEqual(int(count.read_text()), attempts)
            self.assertEqual("S25_INNER_SIGNED_INDEX_REFRESH=PASS" in run.stdout, expected == 0)

    def test_first_success(self):
        self.check(["OK"], 0, 1)

    def test_mirror_sync_recovers(self):
        self.check(["File has unexpected size", "Hash Sum mismatch", "OK"], 0, 3)

    def test_bounded_failure(self):
        self.check(["Hash Sum mismatch"] * 3, 1, 3)

    def test_signature_refusal(self):
        self.check(["Hash Sum mismatch; NO_PUBKEY"], 1, 1)

    def test_other_failure(self):
        self.check(["E: repository unavailable"], 1, 1)

if __name__ == "__main__":
    unittest.main()

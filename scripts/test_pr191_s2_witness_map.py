"""Synthetic lane consumer calibration; no native acceptance."""
from copy import deepcopy
from fnmatch import fnmatchcase
from pathlib import Path
import re
import unittest
from pr191_s2_witness_map import load_map, requirements, evaluate, evaluate_set, ci_receipt, evaluate_receipts
from s2_p3_evidence_audit import LIST_WITNESS

SOURCE = "a" * 40
BINARY = "b" * 64
ROWS = load_map()
LANES = sorted({lane for row in ROWS for lane in row["requires"]})


def example(pattern):
    if not pattern.startswith("re:"):
        return pattern
    text = pattern[3:]
    text = re.sub(r"\[([a-z0-9-]+)\]\{(\d+)\}", lambda m: ("a" if "a" in m[1] else "0") * int(m[2]), text)
    text = re.sub(r"\(\?:([^|]+)\|[^)]+\)", r"\1", text)
    text = text.replace("[0-9]+", "8").replace(".+", "synthetic")
    assert re.fullmatch(pattern[3:], text), (pattern, text)
    return text


def fixture(lane):
    if lane == "s25":
        headers = ["S25_LAB_SOURCE_SHA=" + SOURCE, "S25_LAB_CANDIDATE_SHA256=" + BINARY]
    else:
        headers = ["N2_INSTALL_SOURCE=" + SOURCE,
                   "N2_INSTALL_CANDIDATE_SHA256=" + BINARY,
                   "N2_INSTALLED_CANDIDATE_SHA256=" + BINARY,
                   "N2_EVIDENCE_LANE=" + lane]
    witnesses = [example(x) for x in requirements(ROWS,lane)]
    if lane in ("scoped-create-ack","scoped-delete-ack"):
        witnesses.append(next(x for x in witnesses if x.startswith("P1B_SCOPED_ACK_ORIGINAL_RETAINED ")))
    elif re.fullmatch(r"scoped-(create|delete|exec)-wal-(before|after)",lane):
        order = ["P1B_SCOPED_WAL_EXACT_CONTROLLER_CUT", "P1B_SCOPED_WAL_SQLITE_NATIVE_CROSSCHECK",
                 "P1B_SCOPED_EXEC_WAL_ATOMIC", "P1B_SCOPED_INSTALLED_WAL_CUT"]
        witnesses.sort(key=lambda x: next(i for i,p in enumerate(order) if x.startswith(p)))
        if lane == "scoped-exec-wal-before":
            witnesses = [x.replace(" exec-before SUCCEEDED ", " exec-before UNKNOWN ")
                         if x.startswith("P1B_SCOPED_EXEC_WAL_ATOMIC=PASS ") else x
                         for x in witnesses]
    return headers + witnesses + (list(LIST_WITNESS) if lane == "scoped-auth-races" else [])


def run(lane, lines, source=SOURCE):
    return evaluate(("\n".join(lines)+"\n").encode(),source,lane)


class WitnessMap(unittest.TestCase):
    def test_exact_union_and_lanes(self):
        self.assertEqual(sum(len(x["old"]) for x in ROWS),72)
        self.assertEqual(sum(len(x["old"]) for x in ROWS if "disposition" not in x),48)
        self.assertEqual(len(LANES),23)
        for lane in LANES:
            with self.subTest(lane=lane):
                report=run(lane,fixture(lane))
                self.assertEqual(report["acceptance"],"INDEPENDENT_REVIEW_REQUIRED")

    def test_workflow_lanes_exact_partition(self):
        workflow = (Path(__file__).resolve().parents[1] / ".github/workflows/v1-n2-installed-lifecycle.yml").read_text()
        modes = set(re.search(r"^        mode: \[(.+)\]$", workflow, re.M)[1].split(", "))
        unmapped = {"legacy", "legacy-offline", "scoped-retirement",
                    "scoped-update-ack", "scoped-update-ack-receipt", "scoped-noquota"}
        self.assertEqual(modes, (set(LANES) - {"s25"}) | unmapped)
        self.assertFalse((set(LANES) - {"s25"}) & unmapped)
        self.assertIn("|".join(("legacy", "legacy-offline", "scoped-retirement",
                               "scoped-update-ack", "scoped-update-ack-receipt", "scoped-noquota")) + ")", workflow)

    def test_n2_cutover_runtime_changes_trigger_native_review(self):
        workflow=(Path(__file__).resolve().parents[1]/".github/workflows/v1-n2-installed-lifecycle.yml").read_text()
        paths=re.findall(r"^      - ([^\n]+)$",workflow.split("  workflow_dispatch:",1)[0],re.M)
        for path in ("cmd/boxweave/main.go","internal/state/store.go",
                     "internal/state/schema_entrypoint_refusal_test.go",
                     "internal/state/network_create_lineage.go",
                     "internal/state/network_admission_guard.go",
                     "internal/state/network_pin_repair.go",
                     "internal/state/delete_dependencies.go",
                     "internal/state/lifecycle_recovery_pin.go",
                     "internal/state/scoped_drain.go",
                     "internal/control/box_tools.go",
                     "internal/control/delete_dependencies.go"):
            with self.subTest(path=path):
                self.assertTrue(any(fnmatchcase(path,p) for p in paths),
                                "N2 path filter omits cutover dependency")

    def test_each_required_line_missing(self):
        for lane in LANES:
            lines=fixture(lane)
            for required in requirements(ROWS,lane):
                with self.subTest(lane=lane,required=required):
                    matches=[x for x in lines if (
                        re.fullmatch(required[3:],x) if required.startswith("re:")
                        else x==required)]
                    self.assertTrue(matches)
                    with self.assertRaises(ValueError):
                        run(lane,[x for x in lines if x not in matches]+
                            ["echo "+x for x in matches])

    def test_wrong_identity_and_lane(self):
        for lane in LANES:
            with self.subTest(lane=lane):
                lines=fixture(lane)
                with self.assertRaises(ValueError):
                    run(lane,lines,"c"*40)
                with self.assertRaises(ValueError):
                    run(lane,lines+[lines[0]])
                if lane!="s25":
                    with self.assertRaises(ValueError):
                        run(lane,[x.replace("N2_EVIDENCE_LANE="+lane,"N2_EVIDENCE_LANE=other") for x in lines])

    def test_ordered_list_witness(self):
        lane="scoped-auth-races"
        lines=fixture(lane)
        for bad in (lines[:-1],lines+[LIST_WITNESS[0]],lines[:-6]+list(reversed(LIST_WITNESS))):
            with self.assertRaises(ValueError):
                run(lane,bad)

    def test_complete_set_and_no_missing_or_swapped_lane(self):
        logs = {lane: ("\n".join(fixture(lane))+"\n").encode() for lane in LANES}
        report = evaluate_set(logs, SOURCE)
        self.assertEqual(report["original_marker_rows"],72)
        self.assertEqual(len(report["replacement_review_required"]),24)
        self.assertEqual(report["public_wan"],"SEPARATE_C04_GATE_NOT_INFERRED")
        for lane in LANES:
            with self.assertRaises(ValueError):
                evaluate_set({k:v for k,v in logs.items() if k!=lane}, SOURCE)
        bad = dict(logs)
        bad["scoped-create-ack"] = logs["scoped-delete-ack"]
        with self.assertRaises(ValueError):
            evaluate_set(bad,SOURCE)

    def test_s25_binary_identity(self):
        lines = fixture("s25")
        for bad in (lines[:1]+lines[2:], lines+[lines[1]],
                    lines[:1]+["S25_LAB_CANDIDATE_SHA256=invalid"]+lines[2:]):
            with self.assertRaises(ValueError):
                run("s25",bad)

    def test_original_lineage_must_match(self):
        for lane,prefix,index in (
            ("scoped-create-ack","P1B_SCOPED_ACK_ORIGINAL_RETAINED",1),
            ("scoped-delete-ack","P1B_SCOPED_ACK_ORIGINAL_RETAINED",1),
            ("scoped-create-wal-before","P1B_SCOPED_WAL_SQLITE_NATIVE_CROSSCHECK",3),
            ("scoped-delete-wal-after","P1B_SCOPED_WAL_SQLITE_NATIVE_CROSSCHECK",3),
            ("scoped-exec-wal-before","P1B_SCOPED_EXEC_WAL_ATOMIC",4),
            ("scoped-update-ack-wal-after","P1B_SCOPED_UPDATE_NATIVE_ACK_WAL_RESTART_RETAINED",2),
        ):
            with self.subTest(lane=lane):
                lines=fixture(lane)
                target=next(i for i,line in enumerate(lines) if line.startswith(prefix))
                fields=lines[target].split()
                fields[index]=fields[index].replace("a","c")
                mixed=lines.copy()
                mixed[target]=" ".join(fields)
                with self.assertRaises(ValueError):
                    run(lane,mixed)
                if "ACK_ORIGINAL_RETAINED" in prefix:
                    with self.assertRaises(ValueError):
                        run(lane,lines[:target]+lines[target+1:])
                else:
                    with self.assertRaises(ValueError):
                        run(lane,lines+[lines[target]])
                    with self.assertRaises(ValueError):
                        run(lane,list(reversed(lines)))

    def test_exec_wal_final_state_matches_pre_restart(self):
        for lane, expected in (("scoped-exec-wal-before", "UNKNOWN"),
                               ("scoped-exec-wal-after", "SUCCEEDED")):
            with self.subTest(lane=lane):
                lines=fixture(lane)
                i=next(i for i,x in enumerate(lines)
                       if x.startswith("P1B_SCOPED_EXEC_WAL_ATOMIC=PASS "))
                fields=lines[i].split()
                fields[2]=expected
                lines[i]=" ".join(fields)
                run(lane,lines)
                fields[2]="SUCCEEDED" if expected=="UNKNOWN" else "UNKNOWN"
                lines[i]=" ".join(fields)
                with self.assertRaises(ValueError):
                    run(lane,lines)
                if lane=="scoped-exec-wal-before":
                    j=next(j for j,x in enumerate(lines)
                           if x.startswith("P1B_SCOPED_WAL_SQLITE_NATIVE_CROSSCHECK=PASS "))
                    lines[j]=lines[j].replace(" VERIFYING "," SUCCEEDED ")
                    run(lane,lines)

    def test_ci_locator_is_not_conclusion(self):
        report=run("scoped-create-ack",fixture("scoped-create-ack"))
        env={"GITHUB_ACTIONS":"true","RUNNER_ENVIRONMENT":"github-hosted",
             "GITHUB_REPOSITORY":"vanishs/boxweave",
             "GITHUB_WORKFLOW_REF":"vanishs/boxweave/.github/workflows/v1-n2-installed-lifecycle.yml@refs/pull/191/merge",
             "GITHUB_JOB":"installed-lifecycle","GITHUB_EVENT_NAME":"pull_request",
             "GITHUB_SHA":"d"*40,"GITHUB_RUN_ID":"123","GITHUB_RUN_ATTEMPT":"2",
             "UNRELATED_SECRET":"not-in-receipt"}
        receipt=ci_receipt(report,env)
        self.assertEqual(receipt["conclusion"],"VERIFY_ORIGINAL_CI_JOB")
        self.assertEqual(receipt["ci"]["run_attempt"],2)
        self.assertEqual(receipt["report"]["source"],SOURCE)
        self.assertNotIn("not-in-receipt",str(receipt))
        for k,v in (("GITHUB_JOB","other"),("GITHUB_RUN_ATTEMPT","0"),
                    ("GITHUB_RUN_ID",""),("GITHUB_ACTIONS","false"),
                    ("GITHUB_WORKFLOW_REF","other"),("GITHUB_EVENT_NAME","push")):
            with self.assertRaises(ValueError):
                ci_receipt(report,{**env,k:v})
        s25=run("s25",fixture("s25"))
        with self.assertRaises(ValueError):
            ci_receipt(s25,env)
        receipt=ci_receipt(s25,{**env,"GITHUB_JOB":"s25-bridgehead-lab",
            "GITHUB_WORKFLOW_REF":"vanishs/boxweave/.github/workflows/v1-pr191-s25-scoped.yml@refs/pull/191/merge"})
        self.assertEqual(receipt["report"]["lane"],"s25")

    def test_complete_ci_receipt_set(self):
        def receipt(lane):
            job,workflow=("s25-bridgehead-lab","v1-pr191-s25-scoped.yml") if lane=="s25" else ("installed-lifecycle","v1-n2-installed-lifecycle.yml")
            env={"GITHUB_ACTIONS":"true","RUNNER_ENVIRONMENT":"github-hosted",
                 "GITHUB_REPOSITORY":"vanishs/boxweave",
                 "GITHUB_WORKFLOW_REF":"vanishs/boxweave/.github/workflows/"+workflow+"@refs/pull/191/merge",
                 "GITHUB_JOB":job,"GITHUB_EVENT_NAME":"pull_request",
                 "GITHUB_SHA":"d"*40,"GITHUB_RUN_ID":"123" if lane=="s25" else "456",
                 "GITHUB_RUN_ATTEMPT":"2"}
            return ci_receipt(run(lane,fixture(lane)),env)
        receipts=[receipt(lane) for lane in LANES]
        result=evaluate_receipts(receipts,SOURCE)
        self.assertEqual(result["scope"],"CI_DECLARED_RECEIPT_INDEX_ONLY")
        self.assertEqual(result["acceptance"],"VERIFY_ORIGINAL_CI_JOBS_AND_REPLACEMENT_SEMANTICS")
        self.assertEqual(len(result["replacement_review_required"]),24)
        for bad in (receipts[:-1],receipts+[receipts[0]]):
            with self.assertRaises(ValueError):
                evaluate_receipts(bad,SOURCE)
        for path,value in ((("map_sha256",),"0"*64),
                           (("report","source"),"0"*40),
                           (("report","requirements_observed"),0),
                           (("ci","run_attempt"),0),
                           (("conclusion",),"SUCCESS")):
            bad=deepcopy(receipts)
            obj=bad[0]
            for key in path[:-1]:
                obj=obj[key]
            obj[path[-1]]=value
            with self.assertRaises(ValueError):
                evaluate_receipts(bad,SOURCE)
        bad=deepcopy(receipts)
        bad[0]["report"]["transcript_sha256"]=bad[1]["report"]["transcript_sha256"]
        with self.assertRaises(ValueError):
            evaluate_receipts(bad,SOURCE)

    def test_ci_receipts_after_existing_gates(self):
        root=Path(__file__).resolve().parents[1]/".github/workflows"
        n2=(root/"v1-n2-installed-lifecycle.yml").read_text()
        self.assertLess(n2.index("grep -Fxq RC14_LEGACY_TO_SCOPED_REINSTALL=PASS"),n2.index("--ci-receipt"))
        self.assertLess(n2.index("--ci-receipt"),n2.index("uses: actions/upload-artifact@v4"))
        for name in ("v1-pr191-s25-scoped.yml","v1-release.yml"):
            s=(root/name).read_text()
            self.assertLess(s.index("sh scripts/pr191_s25_evidence_check.sh "),s.index("--ci-receipt"))
            self.assertIn("pr191-s2-s25-attempt-",s)
        for name in ("v1-n2-installed-lifecycle.yml","v1-pr191-s25-scoped.yml"):
            s=(root/name).read_text().split("permissions:",1)[0]
            for file in ("scripts/s2_p3_evidence_audit.py","scripts/s2_p3_scoped_acceptance.json"):
                self.assertIn(file,s)

    def test_wan_replacement_not_public_positive(self):
        self.assertNotIn("S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS",requirements(ROWS,"scoped-s2-native"))
        self.assertEqual(run("s25",fixture("s25"))["scope"],"LANE_LOG_CHECK_ONLY")

if __name__=="__main__":
    unittest.main()

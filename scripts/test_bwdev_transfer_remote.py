"""Exercise the exact portable JS template with a simulated MCP adapter.

Only disposable local fixtures; no actual remote effects, credentials or
network. The Python planner is real. Node is optional on a source runner;
the independent native connector acceptance is separate.
"""
import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TEMPLATE = ROOT / "dev/examples/TRANSFER_REMOTE_STEP.js"
NODE = shutil.which("node") or shutil.which("nodejs")
BOX = "12345678-1234-1234-1234-123456789abc"

PROGRAM = r'''
const assert = require("node:assert/strict");
const fs = require("node:fs");
const cp = require("node:child_process");
const crypto = require("node:crypto");
const {createTransferRemote} = require(process.argv[1]);
const fixture = JSON.parse(process.argv[2]);
const cfg = fixture.cfg, plan = fixture.plan, rows = plan.actions;
const digest = b => crypto.createHash("sha256").update(b).digest("hex");
const operations = new Map(), keys = new Map(), events = [];
let seq = 0, fault = fixture.fault;
const readJson = (args) => {
    const p = cp.spawnSync(fixture.python,args,{cwd:cfg.cwd,encoding:"utf8"});
    assert.equal(p.status,0,p.stderr);
    return JSON.parse(p.stdout);
};
function create(tool,args) {
    assert.equal(args.box_id,cfg.box_id);
    assert.equal(args.work_ref,cfg.work_ref);
    const old = keys.get(args.idempotency_key);
    if(old) return {...operations.get(old).o,state:"ADMITTED",duplicate:true};
    const id = (++seq).toString(16).padStart(32,"0");
    let stdout = Buffer.alloc(0), stderr = Buffer.alloc(0), exit=0, state="SUCCEEDED";
    if(tool==="box_exec") {
        assert.equal(args.argv[0],"python3");
        assert.equal(args.argv[1],cfg.cwd+"/dev/transfer_plan.py");
        const p=cp.spawnSync(fixture.python,args.argv.slice(1),{cwd:cfg.cwd});
        stdout=p.stdout;stderr=p.stderr;exit=p.status;
        if(fault==="wrong-claim" && args.argv[2]==="begin") {
            const j=JSON.parse(stdout);j.request_sha256="0".repeat(64);
            stdout=Buffer.from(JSON.stringify(j));
        }
    } else {
        assert.equal(tool,"box_file_apply");
        assert.equal(args.mode,"create");
        try {fs.writeFileSync(args.path,Buffer.from(args.content_base64,"base64"),{flag:"wx"});}
        catch(err) {state="FAILED";}
        if(fault==="unknown")state="UNKNOWN";
    }
    const o={operation_id:id,tracking_id:id,box_uuid:cfg.box_id,incarnation:cfg.box_id,
        action:tool,work_ref:cfg.work_ref,state};
    operations.set(id,{o,stdout,stderr,exit});keys.set(args.idempotency_key,id);
    if(tool==="box_file_apply" && fault==="lost-response")throw Error("simulated lost acknowledgement");
    return {...o,state:"ADMITTED",duplicate:fault==="duplicate"&&tool==="box_exec"};
}
async function call(tool,args) {
    events.push({tool,key:args.idempotency_key,path:args.path});
    if(tool==="box_exec"||tool==="box_file_apply")return create(tool,args);
    if(tool==="operation_get")return {operation:operations.get(args.operation_id).o};
    if(tool==="operation_output_get") {
        const o=operations.get(args.operation_id), b=o[args.stream], off=args.offset;
        const data=b.subarray(off,off+args.limit);
        return {operation_id:args.operation_id,tracking_id:args.operation_id,stream:args.stream,
            availability:"available",offset:off,next_offset:off+data.length,ended:off+data.length===b.length,
            truncated:fault==="truncated",exit_code:o.exit,content_base64:data.toString("base64")};
    }
    if(tool==="box_file_read") {
        const b=fs.readFileSync(args.path),off=args.offset||0,data=b.subarray(off,off+args.length);
        return {box_id:cfg.box_id,path:args.path,tracking_id:"read",size:b.length,offset:off,
            data_base64:data.toString("base64"),page_sha256:fault==="bad-page"&&args.path!==fixture.source?"0".repeat(64):digest(data),
            ended:off+data.length===b.length,whole_file:off===0&&data.length===b.length};
    }
    throw Error("Unexpected tool "+tool);
}
async function run(method,args,budget=16) {
    const before=events.length;
    const c=createTransferRemote(call,{...cfg,budget});
    const r=await c.run(method,args);
    assert.equal(r.calls,events.length-before);
    assert.equal(r.calls,r.trace.length);
    return r;
}
const expectState = () => readJson([cfg.cwd+"/dev/transfer_plan.py","assess",cfg.bundle]);
async function main() {
    const c=createTransferRemote(()=>{throw Error("unexpected call")},cfg);
    for(const n of [0,1,55,56,57,64,65,32768,40960,65536,131072]) {
        const b=Buffer.alloc(n);
        for(let i=0;i<n;i++)b[i]=(i*31+i%17)&255;
        assert.equal(c.sha256Base64(b.toString("base64")),digest(b));
    }
    assert.throws(()=>c.sha256Base64("YQ="));
    if(fault) {
        const r=await run("step",{row:rows[0],expect:1,command_key:"control-1",
            source:{path:fixture.source,size:plan.size,offset:0}},fault==="budget"?2:16);
        assert.equal(r.status,"RECONCILE",JSON.stringify(r));
        assert.equal(expectState().status,"RECONCILE");
        const writes=events.filter(e=>e.tool==="box_file_apply");
        if(["budget","duplicate","wrong-claim","truncated"].includes(fault))assert.equal(writes.length,0);
        else assert.equal(writes.length,1);
        if(fault==="lost-response")assert.equal(r.trace.at(-1).state,"CALL_RESULT_UNCERTAIN");
        assert.equal(fs.existsSync(cfg.target),false);
        return;
    }
    let expect=1,previous=null,assembly=null,offset=0;
    for(const row of rows) {
        const first=row.action==="part-0000";
        const args={row,expect,command_key:"control-"+row.action,previous,pause:fixture.pause&&first};
        if(row.action.startsWith("part-")) {
            args.source={path:fixture.source,size:plan.size,offset};offset+=row.size;
        }
        let r=await run("step",args);
        if(fixture.pause&&first) {
            assert.equal(r.status,"PAUSED_ORIGINAL",JSON.stringify(r));
            assert.equal(expectState().status,"RECONCILE");
            expect=r.journal_revision;
            const before=events.filter(e=>e.tool==="box_file_apply"&&e.path===row.path).length;
            r=await run("collect",{row,original:r.original,stageNotSubmitted:true});
            assert.equal(events.filter(e=>e.tool==="box_file_apply"&&e.path===row.path).length,before);
        } else expect=r.journal_revision;
        if(row.action==="assemble") {
            assert.equal(r.status,"ASSEMBLED_READBACK_REQUIRED",JSON.stringify(r));assembly=r;
        } else {
            assert.equal(r.status,"EVIDENCE_STAGED",JSON.stringify(r));previous=r;
        }
    }
    for(let index=0;index*32768<plan.size;index++) {
        const p=await run("stageAssemblyPage",{row:rows.at(-1),index,source:fixture.source});
        assert.equal(p.status,"PAGE_STAGED",JSON.stringify(p));
    }
    const final=await run("finish",{row:rows.at(-1),expect,original:assembly.original,command_key:"control-final"});
    assert.equal(final.status,"VERIFIED_READBACK",JSON.stringify(final));
    assert.equal(expectState().status,"VERIFIED_READBACK");
    assert.deepEqual(fs.readFileSync(cfg.target),fs.readFileSync(fixture.source));
    const attempts=events.filter(e=>e.tool==="box_file_apply"&&e.path===rows[0].path);
    assert.equal(attempts.length,1);
    console.log(JSON.stringify({status:final.status,calls:events.length,pause:fixture.pause}));
}
main().catch(e=>{console.error(e);process.exitCode=1;});
'''


@unittest.skipUnless(NODE, "Node unavailable on this runner; use separate JS test lane")
class RemoteTemplateTests(unittest.TestCase):
    def fixture(self, pause=False, fault=None):
        with tempfile.TemporaryDirectory(prefix="bwtr-remote-") as tmp:
            directory = Path(tmp).resolve()
            source = directory / "source.bin"
            source.write_bytes(bytes((i*19+i//256)%256 for i in range(81920)))
            bundle = directory / "bundle"
            target = directory / "target.bin"
            result = subprocess.run([sys.executable, str(ROOT/"dev/transfer_plan.py"),
                "plan", "--source", str(source), "--output", str(bundle), "--box", BOX,
                "--target", str(target), "--work-ref", "fixture/remote", "--transfer-ref", "fixture-1",
                "--chunk-size", "40960"], check=True, capture_output=True, text=True)
            plan = json.loads(result.stdout)
            cfg = dict(box_id=BOX, work_ref="fixture/remote", cwd=str(ROOT), bundle=str(bundle),
                       target=str(target), transfer_id=plan["transfer_id"], budget=16)
            fixture = dict(cfg=cfg, plan=plan, source=str(source), pause=pause, fault=fault,
                           python=sys.executable)
            proc = subprocess.run([NODE, "-e", PROGRAM, str(TEMPLATE), json.dumps(fixture)],
                                  capture_output=True, text=True, timeout=45)
            self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)

    def test_real_planner_simulated_connector_normal(self):
        self.fixture()

    def test_real_planner_simulated_connector_pause(self):
        self.fixture(pause=True)

    def test_fail_closed_boundaries(self):
        for fault in ("budget", "duplicate", "wrong-claim", "truncated", "unknown",
                      "bad-page", "lost-response"):
            with self.subTest(fault=fault):
                self.fixture(fault=fault)


if __name__ == "__main__":
    unittest.main()

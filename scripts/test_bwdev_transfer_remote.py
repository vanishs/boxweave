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
        if(fault==="binding-unrelated"&&args.path===rows[0].path) {
            // The intended operation does not execute. A separate fixture
            // write leaves matching bytes; a wrong admission/lookup pair
            // represents another successful same-work Operation.
            fs.writeFileSync(args.path,Buffer.from(args.content_base64,"base64"),{flag:"wx"});
        } else {
            try {fs.writeFileSync(args.path,Buffer.from(args.content_base64,"base64"),{flag:"wx"});}
            catch(err) {state="FAILED";}
        }
        if(fault==="unknown")state="UNKNOWN";
    }
    const tracking=(seq+10000).toString(16).padStart(32,"0");
    const o={operation_id:id,tracking_id:tracking,box_uuid:cfg.box_id,incarnation:cfg.box_id,
        action:tool,work_ref:cfg.work_ref,state};
    operations.set(id,{o,stdout,stderr,exit});keys.set(args.idempotency_key,id);
    if(tool==="box_file_apply" && fault==="lost-response")throw Error("simulated lost acknowledgement");
    return {...o,tracking_id:fault==="binding-unrelated"&&args.path===rows[0].path?"d".repeat(32):tracking,
        state:"ADMITTED",duplicate:fault==="duplicate"&&tool==="box_exec"};
}
async function call(tool,args) {
    events.push({tool,key:args.idempotency_key,path:args.path});
    if(tool==="box_exec"||tool==="box_file_apply")return create(tool,args);
    if(tool==="operation_get")return {tracking_id:"e".repeat(32),operation:operations.get(args.operation_id).o};
    if(tool==="operation_output_get") {
        const o=operations.get(args.operation_id), b=o[args.stream], off=args.offset;
        const data=b.subarray(off,off+args.limit);
        return {operation_id:args.operation_id,tracking_id:args.operation_id,stream:args.stream,
            availability:"available",offset:off,next_offset:off+data.length,ended:off+data.length===b.length,
            truncated:fault==="truncated",exit_code:o.exit,content_base64:data.toString("base64")};
    }
    if(tool==="box_file_list") {
        const entries=fs.readdirSync(args.path).sort().filter(name=>name>args.cursor)
            .slice(0,args.limit).map(name=>({name,type:"file"}));
        return {box_id:cfg.box_id,path:args.path,entries,consistent_snapshot:false};
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
    if(fault&&fault.startsWith("binding-")) {
        const row=rows[0];
        const paused=await run("step",{row,expect:1,command_key:"control-1",pause:true,
            source:{path:fixture.source,size:plan.size,offset:0}});
        assert.equal(paused.status,"PAUSED_ORIGINAL",JSON.stringify(paused));
        assert.equal(expectState().journal_revision,2);
        const original=JSON.parse(JSON.stringify(paused.original));
        const before=events.filter(e=>e.tool==="box_file_apply").length;
        const canonical=v=>Array.isArray(v)?"["+v.map(canonical).join(",")+"]":
            v!==null&&typeof v==="object"?"{"+Object.keys(v).sort().map(k=>JSON.stringify(k)+":"+canonical(v[k])).join(",")+"}":JSON.stringify(v);
        const reseal=a=>{
            delete a.association_sha256;
            a.association_sha256=digest(Buffer.from(canonical(a)+"\n"));
        };
        let expect=2;
        if(fault==="binding-missing")delete original.tracking_id;
        if(fault==="binding-stale")expect=3;
        if(fault==="binding-advanced") {
            const receipt={idempotency_key:row.idempotency_key,operation_id:original.operation_id,
                state:"UNKNOWN",pages:[],exit_code:null};
            const path=cfg.bundle+"/test-unknown.json";
            fs.writeFileSync(path,JSON.stringify(receipt),{flag:"wx"});
            readJson([cfg.cwd+"/dev/transfer_plan.py","record",cfg.bundle,"--expect","2",
                "--action",row.action,"--receipt",path]);
            assert.equal(expectState().journal_revision,3);
        }
        if(fault==="binding-request") {original.request_sha256="0".repeat(64);reseal(original);}
        if(fault==="binding-claim") {original.context.claim_sha256="0".repeat(64);reseal(original);}
        if(fault==="binding-key") {original.idempotency_key="other-key";reseal(original);}
        if(fault==="binding-receipt") {
            const collected=await run("collect",{row,original,expect:2,stageNotSubmitted:true});
            assert.equal(collected.status,"EVIDENCE_STAGED",JSON.stringify(collected));
            operations.get(collected.receipt.operation_id).o.tracking_id="f".repeat(32);
            const count=events.length;
            const r=await run("step",{row:rows[1],expect:2,command_key:"control-next",previous:collected});
            assert.equal(r.status,"RECONCILE",JSON.stringify(r));
            assert.equal(events.slice(count).filter(e=>e.tool==="box_exec"||e.tool==="box_file_apply").length,0);
        } else if(fault==="binding-finish"||fault==="binding-page") {
            let previous=await run("collect",{row,original,expect:2,stageNotSubmitted:true});
            assert.equal(previous.status,"EVIDENCE_STAGED",JSON.stringify(previous));
            let revision=2,assembly;
            for(const next of rows.slice(1)) {
                const r=await run("step",{row:next,expect:revision,command_key:"control-"+next.action,previous});
                revision=r.journal_revision;
                assert.equal(r.status,next.action==="assemble"?"ASSEMBLED_READBACK_REQUIRED":"EVIDENCE_STAGED",JSON.stringify(r));
                if(next.action==="assemble")assembly=r;else previous=r;
            }
            const finalRow=rows.at(-1);
            if(fault==="binding-finish") {
                for(let index=0;index*32768<plan.size;index++) {
                    const p=await run("stageAssemblyPage",{row:finalRow,index,source:fixture.source,original:assembly.original,expect:revision});
                    assert.equal(p.status,"PAGE_STAGED",JSON.stringify(p));
                }
            }
            operations.get(assembly.original.operation_id).o.tracking_id="f".repeat(32);
            const count=events.length;
            const r=fault==="binding-finish"?await run("finish",{row:finalRow,expect:revision,original:assembly.original,command_key:"control-final"}):
                await run("stageAssemblyPage",{row:finalRow,index:0,source:fixture.source,original:assembly.original,expect:revision});
            assert.equal(r.status,"RECONCILE",JSON.stringify(r));
            assert.equal(events.slice(count).filter(e=>e.tool==="box_exec"||e.tool==="box_file_apply").length,0);
            assert.equal(expectState().journal_revision,revision);
            assert.equal(expectState().status,"RECONCILE");
            return;
        } else {
            const r=await run("collect",{row,original,expect,stageNotSubmitted:true});
            assert.equal(r.status,"RECONCILE",JSON.stringify(r));
            assert.equal(events.filter(e=>e.tool==="box_file_apply").length,before);
        }
        assert.equal(expectState().journal_revision,fault==="binding-advanced"?3:2);
        assert.equal(expectState().status,"RECONCILE");
        return;
    }
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
            r=await run("collect",{row,original:JSON.parse(JSON.stringify(r.original)),expect,stageNotSubmitted:true});
            assert.equal(events.filter(e=>e.tool==="box_file_apply"&&e.path===row.path).length,before);
        } else expect=r.journal_revision;
        if(row.action==="assemble") {
            assert.equal(r.status,"ASSEMBLED_READBACK_REQUIRED",JSON.stringify(r));assembly=r;
        } else {
            assert.equal(r.status,"EVIDENCE_STAGED",JSON.stringify(r));previous=r;
        }
    }
    for(let index=0;index*32768<plan.size;index++) {
        const p=await run("stageAssemblyPage",{row:rows.at(-1),index,source:fixture.source,original:assembly.original,expect});
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

    def test_original_admission_association(self):
        for fault in ("binding-unrelated", "binding-missing", "binding-stale",
                      "binding-request", "binding-claim", "binding-key", "binding-advanced",
                      "binding-receipt", "binding-page", "binding-finish"):
            with self.subTest(fault=fault):
                self.fixture(fault=fault)

    def test_fail_closed_boundaries(self):
        for fault in ("budget", "duplicate", "wrong-claim", "truncated", "unknown",
                      "bad-page", "lost-response"):
            with self.subTest(fault=fault):
                self.fixture(fault=fault)


if __name__ == "__main__":
    unittest.main()

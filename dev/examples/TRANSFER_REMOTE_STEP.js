function createTransferRemote(call, cfg, emit = () => {}) {
    // Copy this reviewed function verbatim into a connector scripting surface.
    // No dynamic loading, network credentials, retry, or durable client storage.
    const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    const need = (ok, message) => { if (!ok) throw Error(message); };
    const ascii = s => {
        need(typeof s === "string" && /^[\x00-\x7f]*$/.test(s), "Expected ASCII");
        return Uint8Array.from(s, c => c.charCodeAt(0));
    };
    const b64 = bytes => {
        let s = "";
        for (let i = 0; i < bytes.length; i += 3) {
            const n = (bytes[i] << 16) | ((bytes[i+1] || 0) << 8) | (bytes[i+2] || 0);
            s += alphabet[n >>> 18] + alphabet[(n >>> 12)&63]
                + (i+1 < bytes.length ? alphabet[(n >>> 6)&63] : "=")
                + (i+2 < bytes.length ? alphabet[n&63] : "=");
        }
        return s;
    };
    const un64 = s => {
        need(typeof s === "string" && s.length <= 180000 && s.length % 4 === 0
            && /^[A-Za-z0-9+/]*={0,2}$/.test(s), "Invalid bounded base64");
        let acc = 0, bits = 0, result = [];
        for (const c of s) {
            if (c === "=") break;
            acc = (acc << 6) | alphabet.indexOf(c); bits += 6;
            if (bits >= 8) { bits -= 8; result.push((acc >>> bits)&255); }
        }
        const out = Uint8Array.from(result);
        need(b64(out) === s, "Noncanonical base64");
        return out;
    };
    const sha = input => {
        need(input.length <= 131072, "Digest input exceeds page bound");
        const k = [0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2];
        let h = [0x6a09e667,0xbb67ae85,0x3c6ef372,0xa54ff53a,0x510e527f,0x9b05688c,0x1f83d9ab,0x5be0cd19];
        const p = new Uint8Array(Math.ceil((input.length+9)/64)*64);
        p.set(input); p[input.length] = 128;
        new DataView(p.buffer).setUint32(p.length-4, input.length*8);
        const rot = (v,n) => (v>>>n)|(v<<(32-n)), w = new Uint32Array(64);
        const view = new DataView(p.buffer);
        for (let off=0; off<p.length; off+=64) {
            for (let i=0;i<16;i++) w[i]=view.getUint32(off+4*i);
            for (let i=16;i<64;i++) {
                const x=w[i-15],y=w[i-2];
                w[i]=w[i-16]+(rot(x,7)^rot(x,18)^(x>>>3))+w[i-7]+(rot(y,17)^rot(y,19)^(y>>>10));
            }
            let [a,b,c,d,e,f,g,j]=h;
            for (let i=0;i<64;i++) {
                const t1=(j+(rot(e,6)^rot(e,11)^rot(e,25))+((e&f)^(~e&g))+k[i]+w[i])>>>0;
                const t2=((rot(a,2)^rot(a,13)^rot(a,22))+((a&b)^(a&c)^(b&c)))>>>0;
                j=g;g=f;f=e;e=(d+t1)>>>0;d=c;c=b;b=a;a=(t1+t2)>>>0;
            }
            h=h.map((v,i)=>(v+[a,b,c,d,e,f,g,j][i])>>>0);
        }
        return h.map(v=>v.toString(16).padStart(8,"0")).join("");
    };
    const str = bytes => Array.from(bytes,c=>String.fromCharCode(c)).join("");
    const id = v => typeof v === "string" && /^[a-f0-9]{32}$/.test(v);
    const path = v => typeof v === "string" && /^\/[A-Za-z0-9._/-]+$/.test(v)
        && !v.includes("//") && !v.endsWith("/") && !v.split("/").some(x=>x==="."||x==="..") && v.length<=512;
    const ref = v => typeof v === "string" && /^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/.test(v);
    need(typeof call === "function" && /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/.test(cfg.box_id)
        && ref(cfg.work_ref) && path(cfg.cwd) && path(cfg.bundle) && path(cfg.target) && id(cfg.transfer_id)
        && Number.isInteger(cfg.budget) && cfg.budget>=1 && cfg.budget<=16, "Invalid explicit remote scope");
    const trace = [];
    let calls = 0;
    async function invoke(tool,args) {
        need(calls<cfg.budget,"Call budget exhausted; reconcile original intent");
        const item = {tool,call:++calls,idempotency_key:args.idempotency_key||null,
            path:args.path||null,operation_id:args.operation_id||null,tracking_id:null,state:"CALL_STARTED"};
        trace.push(item); emit({...item});
        try {
            const r = await call(tool,args);
            Object.assign(item,{operation_id:r?.operation_id||r?.operation?.operation_id||item.operation_id,
                tracking_id:r?.tracking_id||r?.operation?.tracking_id||null,state:r?.state||r?.operation?.state||"RESPONSE"});
            emit({...item}); return r;
        } catch(err) {
            item.state="CALL_RESULT_UNCERTAIN"; emit({...item}); throw err;
        }
    }
    // Canonical digests bind retained caller evidence, not authenticate the
    // server or allow reconstruction of a missing admission association.
    const canonical = value => {
        if (Array.isArray(value)) return "["+value.map(canonical).join(",")+"]";
        if (value!==null && typeof value==="object")
            return "{"+Object.keys(value).sort().map(k=>JSON.stringify(k)+":"+canonical(value[k])).join(",")+"}";
        need(value===null||typeof value==="string"||typeof value==="boolean"||
            (typeof value==="number"&&Number.isSafeInteger(value)), "Invalid canonical value");
        return JSON.stringify(value);
    };
    const digestObject = value => sha(ascii(canonical(value)+"\n"));
    const hex = value => typeof value==="string" && /^[a-f0-9]{64}$/.test(value);
    const identity = (o,action) => o && id(o.operation_id) && id(o.tracking_id) && o.box_uuid===cfg.box_id
        && o.incarnation===cfg.box_id && o.action===action && o.work_ref===cfg.work_ref;
    async function admit(tool,args,context=null) {
        // Calculate the exact request digest before issuing the one call.
        const request_sha256=digestObject(args);
        const r=await invoke(tool,args);
        need(identity(r,tool) && r.state==="ADMITTED" && r.duplicate!==true,
            "No fresh admitted receipt; reconcile, never dispatch again");
        const association={version:1,operation_id:r.operation_id,tracking_id:r.tracking_id,
            action:tool,box_uuid:r.box_uuid,incarnation:r.incarnation,work_ref:r.work_ref,
            idempotency_key:args.idempotency_key,request_sha256,context};
        association.association_sha256=digestObject(association);
        // Retain a complete admission mapping even if a later call hits the
        // platform cap. Never construct one from a lookup or target bytes.
        trace[trace.length-1].association=association;
        emit({...trace[trace.length-1]});
        return association;
    }
    function binding(a,tool,context) {
        need(a && a.version===1 && identity(a,tool) && ref(a.idempotency_key)
            && hex(a.request_sha256) && hex(a.association_sha256)
            && Object.keys(a).sort().join(",")==="action,association_sha256,box_uuid,context,idempotency_key,incarnation,operation_id,request_sha256,tracking_id,version,work_ref",
            "Complete original admission association required");
        const {association_sha256,...fields}=a;
        need(digestObject(fields)===association_sha256 &&
            (context===undefined||canonical(a.context)===canonical(context)),
            "Original admission association or context differs");
        return a;
    }
    function requestBinding(a,args) {
        need(a.idempotency_key===args.idempotency_key && a.request_sha256===digestObject(args),
            "Original admitted key/request differs");
    }
    async function terminal(a,tool) {
        binding(a,tool);
        const r=await invoke("operation_get",{operation_id:a.operation_id,wait_ms:5000});
        const o=r.operation;
        need(identity(o,tool) && o.operation_id===a.operation_id && o.tracking_id===a.tracking_id
            && o.state==="SUCCEEDED" && (o.idempotency_key===undefined||o.idempotency_key===a.idempotency_key),
            "Original operation/Tracking ID or success differs; stop");
        return o;
    }
    async function stream(operation,kind) {
        let offset=0, content="";
        for(let n=0;n<3;n++) {
            const r=await invoke("operation_output_get",{operation_id:operation,stream:kind,offset,limit:32768});
            const bytes=un64(r.content_base64);
            need(r.operation_id===operation && r.stream===kind && r.offset===offset && r.availability==="available"
                && r.truncated===false && r.exit_code===0 && r.next_offset===offset+bytes.length
                && (r.ended===true||bytes.length>0) && content.length+bytes.length<=131072,
                "Incomplete or failed original output");
            content+=str(bytes);offset=r.next_offset;
            if(r.ended===true)return content;
        }
        throw Error("Output page budget exhausted");
    }
    async function execResult(association) {
        await terminal(association,"box_exec");
        const out=await stream(association.operation_id,"stdout"), err=await stream(association.operation_id,"stderr");
        need(err==="","Unexpected guest stderr; inspect original outputs");
        return JSON.parse(out);
    }
    const rowFor = row => {
        need(row && /^(part-[0-9]{4}|manifest|assemble)$/.test(row.action)
            && Number.isInteger(row.size) && row.size>0 && row.size<=4194304
            && /^[a-f0-9]{64}$/.test(row.sha256) && row.idempotency_key===`bwtr-${cfg.transfer_id}-${row.action}`,
            "Invalid reviewed action");
        const split=cfg.target.lastIndexOf("/");
        const artifact=action=>cfg.target.slice(0,split+1)+"."+cfg.target.slice(split+1)+".bwtr-"+cfg.transfer_id+"."+action;
        need(row.path===(row.action==="assemble"?cfg.target:artifact(row.action))
            && (row.action==="assemble"||row.size<=65536),"Action path or size differs");
        return artifact;
    };
    function claimRequest(claim,row,revision) {
        const artifact=rowFor(row);
        need(claim && claim.status==="DISPATCH_INTENT_RECORDED" && claim.action===row.action
            && claim.journal_revision===revision && claim.box_id===cfg.box_id && claim.work_ref===cfg.work_ref
            && claim.idempotency_key===row.idempotency_key && sha(ascii(claim.request_json))===claim.request_sha256,
            "Fresh claim identity/digest mismatch");
        const q=JSON.parse(claim.request_json);
        need(q.box_id===cfg.box_id && q.work_ref===cfg.work_ref && q.idempotency_key===row.idempotency_key,
            "Request authority identity mismatch");
        if(row.action==="assemble") {
            need(Object.keys(q).sort().join(",")==="box_id,idempotency_key,manifest_path,target,work_ref"
                && q.target===cfg.target && q.manifest_path===artifact("manifest"),"Assembly request mismatch");
        } else {
            const data=un64(q.content_base64);
            need(Object.keys(q).sort().join(",")==="box_id,content_base64,idempotency_key,mode,path,work_ref"
                && q.mode==="create" && q.path===row.path && data.length===row.size && sha(data)===row.sha256,
                "File request mismatch");
        }
        return q;
    }
    function effectContext(row,revision,claim_sha256) {
        rowFor(row);
        need(Number.isInteger(revision)&&revision>=2&&hex(claim_sha256),"Exact dispatch revision and claim digest required");
        return {kind:"transfer",bundle:cfg.bundle,transfer_id:cfg.transfer_id,
            row:{action:row.action,path:row.path,size:row.size,sha256:row.sha256,idempotency_key:row.idempotency_key},
            journal_revision:revision,claim_sha256};
    }
    function effectBinding(row,a,expect) {
        const tool=row.action==="assemble"?"box_exec":"box_file_apply";
        binding(a,tool,effectContext(row,expect,a?.context?.claim_sha256));
        need(a.idempotency_key===row.idempotency_key,"Original transfer key differs");
    }
    function assemblyRequest(row) {
        const artifact=rowFor(row);
        return {box_id:cfg.box_id,idempotency_key:row.idempotency_key,work_ref:cfg.work_ref,
            manifest_path:artifact("manifest"),target:cfg.target};
    }
    function assemblyArgs(q) {
        return {box_id:q.box_id,idempotency_key:q.idempotency_key,work_ref:q.work_ref,cwd:cfg.cwd,
            argv:["python3",cfg.cwd+"/dev/transfer_plan.py","assemble",q.manifest_path],timeout_ms:30000};
    }
    function fileRequest(row,p) {
        const bytes=un64(p.data_base64);
        need(p.box_id===cfg.box_id&&p.path===row.path&&p.size===row.size&&p.offset===0
            &&p.ended===true&&p.whole_file===true&&bytes.length===row.size
            &&sha(bytes)===row.sha256&&p.page_sha256===row.sha256,"Original target bytes differ");
        return {box_id:cfg.box_id,idempotency_key:row.idempotency_key,work_ref:cfg.work_ref,
            path:row.path,mode:"create",content_base64:p.data_base64};
    }
    function checkEffectRequest(row,a,q) {
        need(a.context.claim_sha256===digestObject(q),"Canonical claim digest differs");
        requestBinding(a,row.action==="assemble"?assemblyArgs(q):q);
    }
    async function dispatchEvent(row,a,expect) {
        effectBinding(row,a,expect);
        const file=cfg.bundle+"/journal/"+String(expect).padStart(8,"0")+".json";
        const p=await invoke("box_file_read",{box_id:cfg.box_id,path:file,offset:0,length:65536});
        const bytes=un64(p.data_base64);
        need(p.box_id===cfg.box_id&&p.path===file&&p.offset===0&&p.ended===true&&p.whole_file===true
            &&p.size===bytes.length&&sha(bytes)===p.page_sha256,"Original dispatch journal event unavailable");
        const event={kind:"dispatch",action:row.action,request_sha256:a.context.claim_sha256,revision:expect};
        need(str(bytes)===canonical(event)+"\n","Original journal event differs");
        // Non-snapshot freshness check: an already advanced journal must
        // stop. This is not a distributed lock; retain the single-writer
        // requirement and the planner's final revision CAS.
        const directory=cfg.bundle+"/journal", cursor=String(expect).padStart(8,"0")+".json";
        const newer=await invoke("box_file_list",{box_id:cfg.box_id,path:directory,cursor,limit:1});
        need(newer.box_id===cfg.box_id&&newer.path===directory&&Array.isArray(newer.entries)
            &&newer.entries.length===0&&!newer.next_cursor,
            "Journal has advanced or freshness is uncertain; reconcile");
    }
    function evidenceContext(parent,file,key) {
        return {kind:"evidence",bundle:cfg.bundle,transfer_id:cfg.transfer_id,
            parent_association_sha256:parent.association_sha256,path:file,key};
    }
    function evidenceArgs(file,key,raw) {
        need(path(file)&&file.startsWith(cfg.bundle+"/")&&raw.length<=65536&&ref(key),"Evidence outside bundle or too large");
        return {box_id:cfg.box_id,path:file,mode:"create",content_base64:b64(raw),
            idempotency_key:key,work_ref:cfg.work_ref};
    }
    async function page(file,size,offset,length) {
        const p=await invoke("box_file_read",{box_id:cfg.box_id,path:file,offset,length});
        const bytes=un64(p.data_base64);
        need(p.box_id===cfg.box_id && p.path===file && p.size===size && p.offset===offset
            && bytes.length===Math.min(length,size-offset) && bytes.length>0 && sha(bytes)===p.page_sha256
            && p.ended===(offset+bytes.length===size) && p.whole_file===(offset===0&&bytes.length===size),
            "Readback page mismatch");
        return p;
    }
    async function stageEvidence(file,key,value,parent) {
        const raw=ascii(JSON.stringify(value)), args=evidenceArgs(file,key,raw);
        const association=await admit("box_file_apply",args,evidenceContext(parent,file,key));
        await terminal(association,"box_file_apply");
        return {path:file,operation_id:association.operation_id,association,size:raw.length,sha256:sha(raw)};
    }
    async function collect(row,original,expect,stageNotSubmitted) {
        rowFor(row);
        need(row.action!=="assemble" && stageNotSubmitted===true,
            "Reconcile original dispatch and any previous evidence write first");
        effectBinding(row,original,expect);
        await terminal(original,"box_file_apply");
        await dispatchEvent(row,original,expect);
        const p=await page(row.path,row.size,0,65536);
        checkEffectRequest(row,original,fileRequest(row,p));
        const file=cfg.bundle+"/receipt-"+row.action+"-"+original.operation_id;
        const staged=await stageEvidence(file+".json","bwtr-"+cfg.transfer_id+"-receipt-"+original.operation_id,
            {idempotency_key:row.idempotency_key,operation_id:original.operation_id,state:"SUCCEEDED",pages:[p],exit_code:null},original);
        return {status:"EVIDENCE_STAGED",action:row.action,row,journal_revision:expect,original,receipt:staged};
    }
    async function previousReceipt(previous,expect) {
        const row=previous?.row, original=previous?.original, receipt=previous?.receipt;
        need(previous?.status==="EVIDENCE_STAGED"&&row&&previous.action===row.action&&row.action!=="assemble"
            &&previous.journal_revision===expect,"Exact prior staged receipt and revision required");
        effectBinding(row,original,expect);
        const file=cfg.bundle+"/receipt-"+row.action+"-"+original.operation_id+".json";
        const key="bwtr-"+cfg.transfer_id+"-receipt-"+original.operation_id;
        need(receipt?.path===file&&receipt.operation_id===receipt.association?.operation_id
            &&Number.isInteger(receipt.size)&&receipt.size>0&&receipt.size<=65536&&hex(receipt.sha256),
            "Original evidence-create reference required");
        binding(receipt.association,"box_file_apply",evidenceContext(original,file,key));
        // Verify both original associations before consuming staged receipt
        // bytes. A same-work success or matching bytes alone is insufficient.
        await terminal(original,"box_file_apply");
        await terminal(receipt.association,"box_file_apply");
        const p=await page(file,receipt.size,0,65536), raw=un64(p.data_base64);
        need(p.page_sha256===receipt.sha256,"Staged evidence digest differs");
        requestBinding(receipt.association,evidenceArgs(file,key,raw));
        const value=JSON.parse(str(raw));
        need(Object.keys(value).sort().join(",")==="exit_code,idempotency_key,operation_id,pages,state"
            &&value.operation_id===original.operation_id&&value.idempotency_key===row.idempotency_key
            &&value.state==="SUCCEEDED"&&value.exit_code===null&&Array.isArray(value.pages)&&value.pages.length===1,
            "Staged receipt does not bind original effect");
        checkEffectRequest(row,original,fileRequest(row,value.pages[0]));
    }
    // Each call executes at most one fresh journal claim and one transfer
    // effect. On failure the original journal and trace, not retries, guide resume.
    async function step({row,expect,command_key,previous=null,pause=false,source=null}) {
        rowFor(row);
        need(Number.isInteger(expect)&&expect>=1&&ref(command_key)&&command_key!==row.idempotency_key,"Explicit fresh command key and revision required");
        let argv=["python3",cfg.cwd+"/dev/transfer_plan.py"];
        if(previous===null) argv.push("begin",cfg.bundle,"--expect",String(expect),"--action",row.action,"--emit-request");
        else {
            await previousReceipt(previous,expect);
            argv.push("advance",cfg.bundle,"--expect",String(expect),"--action",previous.action,
                "--receipt",previous.receipt.path,"--next-action",row.action);
        }
        // Validate optional lab source before publishing the next claim.
        let sourcePage=null;
        if(source!==null) {
            need(row.action.startsWith("part-") && path(source.path) && Number.isInteger(source.size)
                && Number.isInteger(source.offset) && source.offset>=0 && source.offset+row.size<=source.size,
                "Invalid explicit source page");
            sourcePage=await page(source.path,source.size,source.offset,row.size);
            need(sourcePage.page_sha256===row.sha256,"Source page differs from reviewed plan");
        }
        const control=await admit("box_exec",{box_id:cfg.box_id,idempotency_key:command_key,work_ref:cfg.work_ref,cwd:cfg.cwd,argv,timeout_ms:20000});
        const out=await execResult(control);
        if(previous!==null)need(out.status==="NEXT_INTENT_RECORDED" && out.recorded_revision===expect+1,"No next claim; reconcile");
        const claim=previous===null?out:out.claim, q=claimRequest(claim,row,expect+(previous===null?1:2));
        if(sourcePage!==null)need(sourcePage.data_base64===q.content_base64,"Actual source/request bytes differ");
        const context=effectContext(row,claim.journal_revision,claim.request_sha256);
        const original=row.action==="assemble"
            ? await admit("box_exec",assemblyArgs(q),context)
            : await admit("box_file_apply",q,context);
        const checkpoint={action:row.action,row,journal_revision:claim.journal_revision,control:control.operation_id,original};
        if(pause)return {status:"PAUSED_ORIGINAL",...checkpoint};
        if(row.action==="assemble") {
            checkEffectRequest(row,original,q);
            const result=await execResult(original);
            need(result.status==="ASSEMBLED_READBACK_REQUIRED"&&result.path===row.path&&result.size===row.size&&result.sha256===row.sha256,
                "Original assembly output mismatch");
            return {status:"ASSEMBLED_READBACK_REQUIRED",...checkpoint};
        }
        return {...await collect(row,original,claim.journal_revision,true),...checkpoint};
    }
    async function stageAssemblyPage({row,index,source=null,original,expect}) {
        rowFor(row);need(row.action==="assemble"&&Number.isInteger(index)&&index>=0&&index*32768<row.size,"Invalid final page");
        effectBinding(row,original,expect);
        checkEffectRequest(row,original,assemblyRequest(row));
        await terminal(original,"box_exec");
        await dispatchEvent(row,original,expect);
        const offset=index*32768,p=await page(row.path,row.size,offset,32768);
        if(source!==null) {
            need(path(source)&&source!==row.path,"Independent source path required");
            const s=await page(source,row.size,offset,32768);
            need(s.data_base64===p.data_base64&&s.page_sha256===p.page_sha256,"Final source/target bytes differ");
        }
        const name=String(index).padStart(4,"0");
        return {status:"PAGE_STAGED",index,ended:p.ended,sha256:p.page_sha256,
            receipt:await stageEvidence(`${cfg.bundle}/assembly-page-${name}.json`,`bwtr-${cfg.transfer_id}-page-${name}`,p,original)};
    }
    async function finish({row,expect,original,command_key}) {
        rowFor(row);
        need(row.action==="assemble"&&Number.isInteger(expect)&&expect>0&&ref(command_key)&&command_key!==row.idempotency_key,
            "Invalid final receipt reference");
        effectBinding(row,original,expect);
        checkEffectRequest(row,original,assemblyRequest(row));
        await dispatchEvent(row,original,expect);
        const assembled=await execResult(original);
        need(assembled.status==="ASSEMBLED_READBACK_REQUIRED"&&assembled.path===row.path&&assembled.size===row.size&&assembled.sha256===row.sha256,
            "Original assembly output mismatch");
        const r=await admit("box_exec",{box_id:cfg.box_id,idempotency_key:command_key,work_ref:cfg.work_ref,cwd:cfg.cwd,
            argv:["python3",cfg.cwd+"/dev/transfer_plan.py","record-assembly-pages",cfg.bundle,"--expect",String(expect),
                "--operation-id",original.operation_id,"--exit-code","0","--count",String(Math.ceil(row.size/32768))],timeout_ms:30000});
        const result=await execResult(r);
        need(result.status==="VERIFIED_READBACK"&&result.journal_revision===expect+1,"Final receipt not verified");
        return {status:result.status,journal_revision:result.journal_revision,record_operation:r.operation_id};
    }
    async function run(method,args) {
        try {
            const fn={step,collect: x=>collect(x.row,x.original,x.expect,x.stageNotSubmitted),stageAssemblyPage,finish}[method];
            need(fn,"Unknown explicit remote step");
            return {...await fn(args),calls,trace};
        } catch(err) {
            return {status:"RECONCILE",reason:String(err.message||err),calls,trace};
        }
    }
    return {run,sha256Ascii:s=>sha(ascii(s)),base64Ascii:s=>b64(ascii(s)),sha256Base64:s=>sha(un64(s))};
}

if (typeof module !== "undefined") module.exports = {createTransferRemote};

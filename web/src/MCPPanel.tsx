import React, {useState} from 'react';

type Resource = {logical:string};
type API = (path:string,method?:string,data?:unknown)=>Promise<any>;
type Receipt = {id:string;logical:string;action:string;state:string;code:string};
type Plan = {id:string;logical:string;action:string;expires:number};

export function MCPPanel({api,resources}:{api:API;resources:Resource[]}) {
  const [logical,setLogical] = useState('');
  const [catalog,setCatalog] = useState<any>(null);
  const [box,setBox] = useState<any>(null);
  const [error,setError] = useState('');
  const [busy,setBusy] = useState(false);
  const [readPath,setReadPath] = useState('');
  const [readResult,setReadResult] = useState('');
  const [command,setCommand] = useState('');
  const [cwd,setCwd] = useState('');
  const [autoSleep,setAutoSleep] = useState(false);
  const [idleMinutes,setIdleMinutes] = useState(30);
  const [plan,setPlan] = useState<Plan|null>(null);
  const [receipts,setReceipts] = useState<Receipt[]>([]);
  async function run(action:()=>Promise<void>) {
    setBusy(true);setError('');
    try { await action(); }
    catch(e) {setError(e instanceof Error?e.message:'REQUEST_FAILED');}
    finally {setBusy(false);}
  }
  function selected(value:string) {
    setLogical(value);setBox(null);setPlan(null);setReadResult('');
    const found=resources.find(r=>r.logical===value);
    if(!found)setCwd('');
  }
  async function status() {
    setCatalog(await api('mcp-capabilities'));
    if(logical)setBox(await api('boxes/'+encodeURIComponent(logical)+'/agent-box'));
  }
  async function planAction(action:string,args:Record<string,unknown>) {
    setPlan(await api('box-action-plans','POST',{logical,action,arguments:args}));
  }
  async function refreshReceipts() {
    setReceipts((await api('box-action-receipts')).operations);
  }
  return <section aria-label="MCP capabilities and Box operations">
    <h2>MCP capability integration</h2>
    <p>Platform tools are discovered through the application launcher, not automatically authorized.
      Agent-box access requires an explicitly configured, exact target with a trusted launcher.
      Source-only integrations require real Box verification before production use.</p>
    {error&&<p className="error" role="alert">{error}</p>}
    <label>Reviewed Box
      <select value={logical} onChange={e=>selected(e.target.value)}>
        <option value="">Select a reviewed Box</option>
        {resources.map(r=><option key={r.logical} value={r.logical}>{r.logical}</option>)}
      </select>
    </label>
    <button disabled={busy} onClick={()=>void run(status)}>Check discovered capabilities</button>
    {catalog&&<p>Application-side Platform tools advertised: {catalog.count} ·
      authorization: {catalog.authorization} · direct Agent gateway: {catalog.direct_agent_gateway}</p>}
    {box&&<details><summary>Box capabilities</summary><pre>{JSON.stringify(box,null,2)}</pre></details>}
    <h3>Read selected Box</h3>
    <p>Read-only request. The server enforces the exact target and configured workspace.</p>
    <label>File path<input value={readPath} onChange={e=>setReadPath(e.target.value)} placeholder="/home/user/project/README.md"/></label>
    <button disabled={busy||!logical||!readPath} onClick={()=>void run(async()=>{
      const value=await api('boxes/'+encodeURIComponent(logical)+'/agent-box/read','POST',
        {tool:'read_file',arguments:{path:readPath}});
      setReadResult(value.content);
    })}>Read file</button>
    {readResult&&<pre>{readResult}</pre>}
    <h3>Plan an approved action</h3>
    <p>Changes require a one-use plan and explicit confirmation. A lost response becomes unknown;
      the server does not replay it automatically.</p>
    <label>Command (Box execution only)<textarea value={command} onChange={e=>setCommand(e.target.value)}
      placeholder="A reviewed test command"/></label>
    <label>Working directory<input value={cwd} onChange={e=>setCwd(e.target.value)} placeholder="/home/user/project"/></label>
    <button disabled={busy||!logical||!command||!cwd} onClick={()=>void run(async()=>{
      await planAction('shell_exec',{command,cwd});
    })}>Preview command plan</button>
    <label><input type="checkbox" checked={autoSleep} onChange={e=>setAutoSleep(e.target.checked)}/>
      Enable automatic sleep</label>
    <label>Idle threshold (minutes)<input type="number" min={5} max={1440}
      value={idleMinutes} onChange={e=>setIdleMinutes(Number(e.target.value))}/></label>
    <button disabled={busy||!logical} onClick={()=>void run(async()=>{
      await planAction('toggle_auto_sleep',{enabled:autoSleep,idleThresholdMinutes:idleMinutes});
    })}>Preview auto-sleep plan</button>
    {plan&&<div className="review">
      <p>Plan for {plan.logical}: {plan.action}. Expires automatically. No automatic dispatch until confirmed.</p>
      <button disabled={busy} onClick={()=>void run(async()=>{
        if(!window.confirm('Execute this one-time action on '+plan.logical+'?'))return;
        await api('box-action-plans/'+plan.id+'/confirm','POST',
          {logical:plan.logical,confirm:true});
        setPlan(null);await refreshReceipts();
      })}>Confirm plan</button>
      <button className="quiet" onClick={()=>setPlan(null)}>Discard local view</button>
    </div>}
    <h3>Operation receipts</h3>
    <button disabled={busy} onClick={()=>void run(refreshReceipts)}>Refresh receipts</button>
    {receipts.map(r=><p key={r.id}>{r.logical} · {r.action} · {r.state} · {r.code}</p>)}
  </section>;
}

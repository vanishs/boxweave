import React, {useEffect, useState} from 'react';

type API=(path:string, method?:string, data?:unknown)=>Promise<any>;
type Entry={id:string;kind:string;label:string;created:number;expires:number|null;revoked:number};
type Summary={enabled:boolean;public_url?:string;mcp_url?:string;port?:number;
  listener?:string;externally_verified?:boolean;credentials?:Entry[];notice?:string};
export function RemoteMCP({api}:{api:API}){
  const [info,setInfo]=useState<Summary|null>(null);
  const [kind,setKind]=useState<'api'|'link'>('api');
  const [label,setLabel]=useState('');
  const [oneTime,setOneTime]=useState('');
  const [error,setError]=useState('');
  const [busy,setBusy]=useState(false);
  async function refresh(){
    setInfo(await api('remote-mcp'));
  }
  useEffect(()=>{void refresh().catch(e=>setError(e instanceof Error?e.message:'UNAVAILABLE'));},[]);
  async function run(action:()=>Promise<void>){
    setBusy(true);setError('');
    try{await action();}catch(e){setError(e instanceof Error?e.message:'REQUEST_FAILED');}
    finally{setBusy(false);}
  }
  async function copy(value:string){
    await navigator.clipboard.writeText(value);
  }
  return <section aria-label="Remote MCP client connections">
    <h2>Remote MCP — HTTPS and OAuth</h2>
    <p>The existing SSH client remains available. Remote MCP is optional and
      requires a separately configured public HTTPS origin and proxy.</p>
    {error&&<p role="alert" className="error">{error}</p>}
    {!info?<p>Reading configured remote MCP status…</p>:!info.enabled?
      <p>Remote MCP is disabled. Enable its reviewed server configuration and
        separately start the loopback remote-mcp service. No public endpoint was created.</p>:<>
      <p><strong>Configured endpoint:</strong> <code>{info.mcp_url}</code></p>
      <button className="quiet" onClick={()=>void run(()=>copy(info.mcp_url||''))}>Copy OAuth MCP URL</button>
      <p>Local listener: {info.listener}:{info.port}. External connectivity has
        <strong> not been verified</strong> by this Web page. Confirm DNS, TLS and
        an actual MCP initialize call through your reverse proxy or tunnel.</p>
      <p><strong>OAuth:</strong> In your MCP client, register the above URL,
        choose OAuth, then sign in using the existing BoxWeave administrator password
        and explicitly authorize the connection.</p>
      <h3>Optional long-lived credentials</h3>
      <p>API Token needs a Bearer Authorization header. A credential URL works
        only with clients that retain its full path. Both grant the current
        trusted operator's broad MCP tool authority, including writes.</p>
      <label>Connection type<select value={kind}
        onChange={e=>{setKind(e.target.value as 'api'|'link');setOneTime('');}}>
        <option value="api">API Token</option>
        <option value="link">Single URL credential</option>
      </select></label>
      <label>Local label<input maxLength={80} value={label}
        onChange={e=>setLabel(e.target.value)} placeholder="Personal coding Agent"/></label>
      <button disabled={busy||!label.trim()} onClick={()=>void run(async()=>{
        if(!window.confirm('Create a credential with broad BoxWeave MCP operator access?'))return;
        const r=await api('remote-mcp/credentials','POST',{kind,label});
        setOneTime(kind==='link'?r.connection_url:r.secret);
        setLabel('');await refresh();
      })}>Create {kind==='api'?'API Token':'connection URL'}</button>
      {oneTime&&<div className="review" role="status">
        <h3>Copy this credential now</h3>
        <p>Shown only once in the interface; do not send in chat, logs or screenshots.</p>
        <code style={{overflowWrap:'anywhere'}}>{oneTime}</code>
        <button onClick={()=>void run(()=>copy(oneTime))}>Copy credential</button>
        <button className="quiet" onClick={()=>setOneTime('')}>Hide value</button>
      </div>}
      <h3>Issued connections</h3>
      <button className="quiet" disabled={busy} onClick={()=>void run(refresh)}>Refresh list</button>
      {(info.credentials||[]).length===0?<p>No remote credentials.</p>:
        (info.credentials||[]).map(entry=><div className="row" key={entry.id}>
          <span><strong>{entry.label}</strong> · {entry.kind} ·
            {entry.revoked?' revoked':' active'}<br/><small>{entry.id}</small></span>
          {!entry.revoked&&<button disabled={busy} onClick={()=>void run(async()=>{
            if(!window.confirm('Revoke this connection immediately?'))return;
            await api('remote-mcp/credentials/'+entry.id,'DELETE',{});setOneTime('');await refresh();
          })}>Revoke</button>}
        </div>)}
      <p className="muted">No per-Agent role or Box-specific restrictions are
        implemented. A future SSH + HTTP identity-policy project is separate.</p>
    </>}
  </section>;
}

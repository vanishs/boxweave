import React, {useEffect, useState} from 'react';
import {createRoot} from 'react-dom/client';
import './style.css';

type Resource={logical:string,user:string,hostname:string};
type Project={id:string,name:string,revision:number};
type Receipt={id:string,state:string,code:string|null,result:{stdout:string,exit_code:number,evidence:string}|null};

function App(){
  const [csrf,setCsrf]=useState(''), [mode,setMode]=useState(''), [password,setPassword]=useState('');
  const [resources,setResources]=useState<Resource[]>([]), [projects,setProjects]=useState<Project[]>([]);
  const [logical,setLogical]=useState(''), [workspace,setWorkspace]=useState(''), [label,setLabel]=useState('');
  const [error,setError]=useState(''), [plan,setPlan]=useState<{id:string,target:Resource,workspace:string}|null>(null);
  const [receipt,setReceipt]=useState<Receipt|null>(null), [busy,setBusy]=useState(false);
  async function api(path:string,method='GET',data?:unknown,token=csrf){
    const r=await fetch('/api/'+path,{method,headers:{'Content-Type':'application/json','X-CSRF-Token':token},
      body:data===undefined?undefined:JSON.stringify(data)});
    const obj=await r.json();
    if(!r.ok) throw new Error(obj.code||'REQUEST_FAILED');
    return obj;
  }
  async function run(fn:()=>Promise<void>){setBusy(true);setError('');try{await fn();}catch(e){setError(e instanceof Error?e.message:'REQUEST_FAILED');}finally{setBusy(false);}}
  async function refresh(token=csrf){
    const a=await api('resources','GET',undefined,token);setResources(a.resources);
    if(a.resources.length){setLogical(a.resources[0].logical);setWorkspace('/home/'+a.resources[0].user);}
    const b=await api('projects','GET',undefined,token);setProjects(b.projects);
  }
  useEffect(()=>{api('session').then(async s=>{setCsrf(s.csrf);setMode(s.mode);await refresh(s.csrf);}).catch(()=>{});},[]);
  useEffect(()=>{
    if(!receipt||['succeeded','failed','unknown'].includes(receipt.state))return;
    const id=setInterval(()=>api('operations/'+receipt.id).then(setReceipt).catch(e=>setError(e.message)),500);
    return()=>clearInterval(id);
  },[receipt?.id,receipt?.state]);
  return <main>
    <header><span className="mark">B</span><div><p className="eyebrow">BOXWEAVE / MINIMUM SYSTEM</p><h1>Connect a project.<br/>Know what happened.</h1></div></header>
    {error&&<p role="alert" className="error">{error}</p>}
    {!csrf?<section><h2>Operator sign in</h2><p>Use the local administrator created for this instance.</p>
      <form onSubmit={e=>{e.preventDefault();void run(async()=>{const s=await api('login','POST',{password});setPassword('');setCsrf(s.csrf);const me=await api('session','GET',undefined,s.csrf);setMode(me.mode);await refresh(s.csrf);});}}>
        <label>Password<input type="password" autoComplete="current-password" value={password} onChange={e=>setPassword(e.target.value)}/></label>
        <button disabled={busy}>Sign in</button></form></section>:<>
      <aside className={mode==='fixture'?'fixture':'live'}>{mode==='fixture'?'Fixture mode — no live infrastructure is connected.':'Live MCP mode — target identity is checked before execution.'}</aside>
      <section><div className="sectionhead"><h2>01 / Bind your project</h2><button className="quiet" onClick={()=>void run(async()=>{await api('logout','POST',{});setCsrf('');setReceipt(null);setPlan(null);setProjects([]);setResources([]);})}>Sign out</button></div>
        <form onSubmit={e=>{e.preventDefault();void run(async()=>{await api('projects','POST',{name:label,logical,workspace});setLabel('');await refresh();});}}>
          <label>Project name<input value={label} required maxLength={80} onChange={e=>setLabel(e.target.value)}/></label>
          <label>Reviewed environment<select value={logical} onChange={e=>{setLogical(e.target.value);const r=resources.find(x=>x.logical===e.target.value);if(r)setWorkspace('/home/'+r.user);}}>
            {resources.map(r=><option key={r.logical} value={r.logical}>{r.logical} · {r.hostname}</option>)}</select></label>
          <label>Remote workspace<input value={workspace} required onChange={e=>setWorkspace(e.target.value)}/></label>
          <button disabled={busy||!logical}>Save binding</button></form></section>
      <section><h2>02 / Review a harmless check</h2><p>Only identity verification and a fixed Python hello are available. No arbitrary shell commands.</p>
        {projects.length===0?<p className="muted">No projects bound yet.</p>:projects.map(p=><div className="row" key={p.id}><strong>{p.name}</strong><span>revision {p.revision}</span>
          <button disabled={busy} onClick={()=>void run(async()=>setPlan(await api('plans','POST',{project:p.id,revision:p.revision})))}>Preview hello</button></div>)}
        {plan&&<div className="review"><h3>Confirm exact target</h3><p>{plan.target.logical} · {plan.target.hostname}</p><code>{plan.workspace}</code>
          <p>Expected output: hello world. The operation will not replay after uncertainty.</p><button disabled={busy} onClick={()=>void run(async()=>{setReceipt(await api('plans/'+plan.id+'/confirm','POST',{confirm:true}));setPlan(null);})}>Confirm hello</button></div>}
      </section>
      <section><h2>03 / Read the receipt</h2><p>Refreshing a receipt never repeats work.</p>
        <button className="quiet" onClick={()=>void run(async()=>{const r=await api('operations');if(r.operations.length)setReceipt(await api('operations/'+r.operations[0].id));else setError('NO_RECEIPTS');})}>Load latest receipt</button>
        {receipt&&<div className="receipt"><strong data-testid="operation-state">{receipt.state}</strong><p>{receipt.code}</p><small>{receipt.id}</small>
          {receipt.result&&<><pre>{receipt.result.stdout}</pre><p>Command exit: {receipt.result.exit_code} · evidence: {receipt.result.evidence}</p></>}
          {receipt.state==='unknown'&&<p>Completion is unknown. Inspect the resource before authorizing another operation.</p>}</div>}
      </section>
    </>}
    <footer>Phase 4 · project binding / trusted identity / durable evidence</footer>
  </main>
}
createRoot(document.getElementById('root')!).render(<App/>);

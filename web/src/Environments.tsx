import React,{useEffect,useState} from 'react';
type Api=(path:string,method?:string,data?:unknown)=>Promise<any>;
type Settings={enabled:boolean,shared_network_tenants?:string[],allowed_names?:string[],resources?:Record<string,string>,resize?:Record<string,string[]>,readiness_timeout?:number,environments:{logical:string,target:unknown,deleted:boolean}[]};
type Plan={nesting?:boolean|null,network_tenant?:string|null,resize?:unknown,id:string,action:string,logical:string,target:unknown,resources:unknown,scope:unknown,expected_state:string|null,readiness_timeout:number};
type Receipt={id:string,action:string,logical:string,state:string,phase:string,code:string|null,deadline:number,acknowledged:boolean,evidence:string,target:unknown,result:unknown};
export function Environments({api}:{api:Api}){
  const [settings,setSettings]=useState<Settings|null>(null),[logical,setLogical]=useState('');
  const [plan,setPlan]=useState<Plan|null>(null),[confirmation,setConfirmation]=useState('');
  const [receipt,setReceipt]=useState<Receipt|null>(null),[history,setHistory]=useState<Receipt[]>([]);
  const [networkTenant,setNetworkTenant]=useState('');
  const [nesting,setNesting]=useState('');
  const [resize,setResize]=useState<Record<string,string>>({cpu:'1',memory:'1GB',disk:'8GB'});
  const [busy,setBusy]=useState(false),[error,setError]=useState('');
  async function refresh(){
    const s=await api('environments');setSettings(s);
    if(!logical&&s.allowed_names?.length)setLogical(s.allowed_names[0]);
    const h=await api('environment-operations');setHistory(h.operations);
  }
  async function run(fn:()=>Promise<void>){setBusy(true);setError('');try{await fn();}catch(e){setError(e instanceof Error?e.message:'REQUEST_FAILED');}finally{setBusy(false);}}
  useEffect(()=>{void run(refresh);},[]);
  useEffect(()=>{
    if(!receipt)return;
    if(['succeeded','failed','unknown'].includes(receipt.state)){void refresh().catch(()=>{});return;}
    const id=setInterval(()=>api('environment-operations/'+receipt.id).then(setReceipt).catch(e=>setError(e.message)),700);
    return()=>clearInterval(id);
  },[receipt?.id,receipt?.state]);
  return <section className="environment-panel">
    <h2>Environment lifecycle</h2>
    <p>Create and manage only the names enabled by this instance. Existing reviewed workloads and provider core are protected.</p>
    {error&&<p role="alert" className="error">{error}</p>}
    {!settings?<p>Loading environment scope...</p>:!settings.enabled?<p>Lifecycle operations are not enabled for this instance.</p>:<>
      <label>Authorized new environment<select value={logical} onChange={e=>{setLogical(e.target.value);setPlan(null);setConfirmation('');}}>
        {settings.allowed_names?.map(n=><option key={n} value={n}>{n}</option>)}
      </select></label>
      {!!settings.shared_network_tenants?.length&&<label>New environment network Tenant<select aria-label="New environment network Tenant" value={networkTenant} onChange={e=>{setNetworkTenant(e.target.value);setPlan(null);}}><option value="">Default personal isolation</option>{settings.shared_network_tenants.map(t=><option key={t}>{t}</option>)}</select><small>Only new creation changes membership. Capability and Guard are checked at preview and dispatch.</small></label>}
      <label>Allow containers inside the new box<select aria-label="New box nesting" value={nesting} onChange={e=>{setNesting(e.target.value);setPlan(null);setConfirmation('');}}>
        <option value="">Provider default</option><option value="true">Enabled — for a BoxWeave server</option><option value="false">Disabled</option>
      </select><small>Independent of Podman. Preview checks Provider support. A changed setting may require restarting this box before child execution works; no automatic restart.</small></label>
      <div className="graph-toolbar" aria-label="Resize resources">
        <label>CPU<select aria-label="Resize CPU" value={resize.cpu} onChange={e=>{setResize({...resize,cpu:e.target.value});setPlan(null);}}>
          {(settings.resize?.cpu??['1','2','3','4','5','6','7','8']).map(v=><option key={v}>{v}</option>)}</select></label>
        <label>Memory<select aria-label="Resize memory" value={resize.memory} onChange={e=>{setResize({...resize,memory:e.target.value});setPlan(null);}}>
          {(settings.resize?.memory??['1GB','2GB','4GB','8GB']).map(v=><option key={v}>{v}</option>)}</select></label>
        <label>Disk<select aria-label="Resize disk" value={resize.disk} onChange={e=>{setResize({...resize,disk:e.target.value});setPlan(null);}}>
          {(settings.resize?.disk??['8GB','10GB','20GB','25GB','30GB','40GB','50GB','100GB']).map(v=><option key={v}>{v}</option>)}</select></label>
        <small>Resize applies to the selected existing box; disk never shrinks. CPU and memory resize online.</small>
      </div>
      <p>New environment profile: {JSON.stringify(settings.resources)}. Readiness budget: {settings.readiness_timeout}s.</p>
      <div className="graph-toolbar">{['create','start','stop','resize','delete'].map(action=><button key={action} disabled={busy||!logical} onClick={()=>void run(async()=>{
        setPlan(await api('environment-plans','POST',{action,logical,
          ...(action==='create'&&networkTenant?{network_tenant:networkTenant}:{}),
          ...(action==='create'&&nesting!==''?{nesting:nesting==='true'}:{}),
          ...(action==='resize'?{cpu:resize.cpu,memory:resize.memory,disk:resize.disk}:{})}));setConfirmation('');
      })}>Preview {action}</button>)}</div>
      <p>Stop, start and delete require an identity recorded from a successful app-managed creation. Delete requires the target to be stopped. No force fallback.</p>
      {plan&&<div className="review">
        <h3>Review {plan.action}: {plan.logical}</h3>
        <pre>{JSON.stringify({scope:plan.scope,nesting:plan.nesting,network_tenant:plan.network_tenant,resize:plan.resize,target:plan.target,resources:plan.resources,expected_state:plan.expected_state},null,2)}</pre>
        <label>Type the exact environment name<input value={confirmation} onChange={e=>setConfirmation(e.target.value)}/></label>
        <button disabled={busy||confirmation!==plan.logical} onClick={()=>void run(async()=>{
          setReceipt(await api('environment-plans/'+plan.id+'/confirm','POST',{confirm:true,logical:confirmation}));setPlan(null);setConfirmation('');
        })}>Confirm {plan.action}</button>
      </div>}
      <button className="quiet" disabled={busy} onClick={()=>void run(refresh)}>Refresh environment receipts</button>
      <div aria-label="Environment receipts">{history.map(item=><div className="row" key={item.id}>
        <span>{item.action} / {item.logical}<br/>{item.state} / {item.phase} / {item.evidence}</span>
        <button disabled={busy} onClick={()=>void run(async()=>setReceipt(await api('environment-operations/'+item.id)))}>Open operation {item.id.slice(0,8)}</button>
      </div>)}</div>
      {receipt&&<div className="receipt">
        <strong data-testid="environment-state">{receipt.state}</strong>
        <p>{receipt.action}: {receipt.logical}</p><p>Phase: {receipt.phase} / {receipt.code}</p>
        <p>Evidence: {receipt.evidence}. Provider response acknowledged: {String(receipt.acknowledged)}.</p>
        <small>{receipt.id}</small>
        <pre>{JSON.stringify({target:receipt.target,result:receipt.result},null,2)}</pre>
        {receipt.state==='unknown'&&<><p>Completion is unknown. Observation does not resend the operation or authorize another create.</p>
          <button disabled={busy} onClick={()=>void run(async()=>setReceipt(await api('environment-operations/'+receipt.id+'/observe','POST',{})))}>Observe current resource</button></>}
      </div>}
    </>}
  </section>;
}

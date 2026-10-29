import React,{useEffect,useState} from 'react';
type Api=(path:string,method?:string,data?:unknown)=>Promise<any>;
type Settings={enabled:boolean,shared_network_tenants?:string[],allowed_names?:string[],resources?:Record<string,string>,readiness_timeout?:number,environments:{logical:string,target:unknown,deleted:boolean}[]};
type Plan={network_tenant?:string|null,id:string,action:string,logical:string,target:unknown,resources:unknown,scope:unknown,expected_state:string|null,readiness_timeout:number};
type Receipt={id:string,action:string,logical:string,state:string,phase:string,code:string|null,deadline:number,acknowledged:boolean,evidence:string,target:unknown,result:unknown};
export function Environments({api}:{api:Api}){
  const [settings,setSettings]=useState<Settings|null>(null),[logical,setLogical]=useState('');
  const [plan,setPlan]=useState<Plan|null>(null),[confirmation,setConfirmation]=useState('');
  const [receipt,setReceipt]=useState<Receipt|null>(null),[history,setHistory]=useState<Receipt[]>([]);
  const [networkTenant,setNetworkTenant]=useState('');
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
      <p>New environment profile: {JSON.stringify(settings.resources)}. Readiness budget: {settings.readiness_timeout}s.</p>
      <div className="graph-toolbar">{['create','start','stop','delete'].map(action=><button key={action} disabled={busy||!logical} onClick={()=>void run(async()=>{
        setPlan(await api('environment-plans','POST',{action,logical,...(action==='create'&&networkTenant?{network_tenant:networkTenant}:{})}));setConfirmation('');
      })}>Preview {action}</button>)}</div>
      <p>Stop, start and delete require an identity recorded from a successful app-managed creation. Delete requires the target to be stopped. No force fallback.</p>
      {plan&&<div className="review">
        <h3>Review {plan.action}: {plan.logical}</h3>
        <pre>{JSON.stringify({scope:plan.scope,network_tenant:plan.network_tenant,target:plan.target,resources:plan.resources,expected_state:plan.expected_state},null,2)}</pre>
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

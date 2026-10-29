import React,{useEffect,useState} from 'react';
type Api=(path:string,method?:string,data?:unknown)=>Promise<any>;
type Settings={enabled:boolean,network:{supported:boolean,atomic_write?:boolean,guard_observed?:boolean,code?:string},tenants:{key:string,label:string,revision:number}[],port_profiles:{id:string,logical:string,protocol:string,external_port:number,target_port:number,audience:string,approved:boolean}[]};
type Plan={id:string,confirmation:string,summary:unknown,scope:unknown};
type Receipt={id:string,state:string,code:string,evidence:string,summary:unknown,result:unknown};
export function AccessPanel({api}:{api:Api}){
  const [data,setData]=useState<Settings|null>(null),[members,setMembers]=useState<any[]>([]),[labels,setLabels]=useState<Record<string,string>>({});
  const [source,setSource]=useState(''),[destination,setDestination]=useState(''),[both,setBoth]=useState(false),[profile,setProfile]=useState('');
  const [plan,setPlan]=useState<Plan|null>(null),[text,setText]=useState(''),[receipt,setReceipt]=useState<Receipt|null>(null),[history,setHistory]=useState<Receipt[]>([]),[busy,setBusy]=useState(false),[error,setError]=useState('');
  async function refresh(){
    const s=await api('access');setData(s);setLabels(Object.fromEntries(s.tenants.map((t:any)=>[t.key,t.label])));
    if(!source&&s.tenants.length)setSource(s.tenants[0].key);if(!destination&&s.tenants.length>1)setDestination(s.tenants[1].key);
    if(!profile&&s.port_profiles.length)setProfile(s.port_profiles[0].id);
    if(s.enabled&&s.network.supported)setMembers((await api('access/memberships')).memberships);
    setHistory((await api('access-operations')).operations);
  }
  async function run(action:()=>Promise<void>){setBusy(true);setError('');try{await action();}catch(e){setError(e instanceof Error?e.message:'REQUEST_FAILED');}finally{setBusy(false);}}
  useEffect(()=>{void run(refresh);},[]);
  useEffect(()=>{
    if(!receipt)return;
    if(['succeeded','failed','unknown'].includes(receipt.state)){void refresh().catch(()=>{});return;}
    const timer=setInterval(()=>api('access-operations/'+receipt.id).then(setReceipt).catch(e=>setError(e.message)),1000);return()=>clearInterval(timer);
  },[receipt?.id,receipt?.state]);
  async function network(action:'grant'|'revoke'){
    const pairs=[[source,destination],...(both?[[destination,source]]:[])];
    const changes=pairs.map(([s,d])=>({destination:d,add_sources:action==='grant'?[s]:[],remove_sources:action==='revoke'?[s]:[]}));
    setPlan(await api('access-plans/network','POST',{changes}));setText('');
  }
  return <section className="access-panel" aria-label="Network and ports"><div className="sectionhead"><h2>Network &amp; ports</h2><button disabled={busy} onClick={()=>void run(refresh)}>Reload access scope</button></div>
    {error&&<p role="alert" className="error">{error}</p>}
    {!data?<p>Loading access scope...</p>:!data.enabled?<p>Network and port changes are not enabled for this instance.</p>:<>
      <h3>Tenant display names</h3><p>Display names describe native network Tenant keys. Renaming a label keeps membership and enforcement unchanged.</p>
      {data.tenants.map(t=><form key={t.key} onSubmit={e=>{e.preventDefault();void run(async()=>{await api('tenants/'+encodeURIComponent(t.key)+'/label','PUT',{label:labels[t.key],revision:t.revision});await refresh();});}}>
        <label>Label for {t.key}<input aria-label={'Label for '+t.key} value={labels[t.key]||''} maxLength={80} onChange={e=>setLabels({...labels,[t.key]:e.target.value})}/></label><button disabled={busy||labels[t.key]===t.label}>Save label {t.key}</button>
      </form>)}
      <h3>Observed membership</h3>{members.map(m=><div className="access-member" key={m.logical}><strong>{m.logical}</strong>{m.code?<p>{m.code}</p>:<p>Account user: {m.account_user}<br/>Network Tenant: {m.network_tenant}</p>}</div>)}
      <h3>Cross-Tenant ingress</h3>
      {!data.network.supported?<p role="status">Network writes unavailable: {data.network.code}</p>:<p>Guard observed: {String(data.network.guard_observed)}. Packet connectivity needs a separate test.</p>}
      <label>Source network Tenant<select aria-label="Source network Tenant" value={source} onChange={e=>{setSource(e.target.value);setPlan(null);}}>{data.tenants.map(t=><option key={t.key} value={t.key}>{t.label} / {t.key}</option>)}</select></label>
      <label>Destination network Tenant<select aria-label="Destination network Tenant" value={destination} onChange={e=>{setDestination(e.target.value);setPlan(null);}}>{data.tenants.map(t=><option key={t.key} value={t.key}>{t.label} / {t.key}</option>)}</select></label>
      <label><input type="checkbox" checked={both} onChange={e=>{setBoth(e.target.checked);setPlan(null);}}/>Both directions in one atomic change</label>
      <div className="graph-toolbar"><button disabled={busy||!data.network.atomic_write||!data.network.guard_observed||!source||!destination||source===destination} onClick={()=>void run(()=>network('grant'))}>Preview grant</button><button disabled={busy||!data.network.atomic_write||!source||!destination||source===destination} onClick={()=>void run(()=>network('revoke'))}>Preview revoke</button></div>
      <h3>Approved port profiles</h3><p>Passthrough provides no source-address filter or TLS. Reachability must be checked from the intended client.</p>
      <label>Port profile<select aria-label="Port profile" value={profile} onChange={e=>{setProfile(e.target.value);setPlan(null);}}>{data.port_profiles.map(p=><option key={p.id} value={p.id}>{p.id} / {p.protocol} {p.external_port} �� {p.logical}:{p.target_port}</option>)}</select></label>
      {data.port_profiles.filter(p=>p.id===profile).map(p=><p key={p.id}>Audience: {p.audience}. Exposure approved in instance scope: {String(p.approved)}.</p>)}
      <div className="graph-toolbar">{['publish','unpublish'].map(action=><button key={action} disabled={busy||!profile||(action==='publish'&&!data.port_profiles.find(p=>p.id===profile)?.approved)} onClick={()=>void run(async()=>{setPlan(await api('access-plans/port','POST',{profile,action}));setText('');})}>Preview {action}</button>)}</div>
      {plan&&<div className="review"><h3>Review exact access change</h3><pre>{JSON.stringify({scope:plan.scope,...(plan.summary as object)},null,2)}</pre><p>Type: <strong>{plan.confirmation}</strong></p><label>Exact access confirmation<input value={text} onChange={e=>setText(e.target.value)}/></label><button disabled={busy||text!==plan.confirmation} onClick={()=>void run(async()=>{setReceipt(await api('access-plans/'+plan.id+'/confirm','POST',{confirmation:text}));setPlan(null);setText('');})}>Confirm access change</button></div>}
      <h3>Access receipts</h3><div aria-label="Access receipts">{history.map(r=><div className="row" key={r.id}><span>{r.state} / {r.code} / {r.evidence}</span><button disabled={busy} onClick={()=>void run(async()=>setReceipt(await api('access-operations/'+r.id)))}>Open access receipt {r.id.slice(0,8)}</button></div>)}</div>
      {receipt&&<div className="receipt"><strong data-testid="access-state">{receipt.state}</strong><p>{receipt.code} / {receipt.evidence}</p><small>{receipt.id}</small><pre>{JSON.stringify({summary:receipt.summary,result:receipt.result},null,2)}</pre>{receipt.state==='unknown'&&<><p>Outcome unknown. Observation never resubmits or authorizes a retry.</p><button disabled={busy} onClick={()=>void run(async()=>setReceipt(await api('access-operations/'+receipt.id+'/observe','POST',{})))}>Observe access state</button></>}</div>}
    </>}
  </section>;
}

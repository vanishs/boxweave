import React,{useEffect,useState} from 'react';
type Selection={client_version:string,provider:string,model:string,access_mode:string};
type State={revision:number,selection:Selection|null,status:string,evidence:string,inference:string,catalog:{client_version:string,providers:{id:string,label:string}[],models:string[],access_modes:string[]}};
type Api=(path:string,method?:string,data?:unknown)=>Promise<any>;
export function ModelWizard({api}:{api:Api}){
  const [saved,setSaved]=useState<State|null>(null),[draft,setDraft]=useState<Selection|null>(null);
  const [selected,setSelected]=useState(false),[access,setAccess]=useState(false),[error,setError]=useState(''),[busy,setBusy]=useState(false),[handoff,setHandoff]=useState('');
  function apply(value:State){setSaved(value);setDraft(value.selection||{client_version:value.catalog.client_version,provider:value.catalog.providers[0].id,model:value.catalog.models[0],access_mode:'account'});setSelected(false);setAccess(false);setHandoff('');}
  async function run(action:()=>Promise<void>){setBusy(true);setError('');try{await action();}catch(e){setError(e instanceof Error?e.message:'REQUEST_FAILED');}finally{setBusy(false);}}
  useEffect(()=>{void run(async()=>apply(await api('model-wizard')));},[]);
  if(!saved||!draft)return <section><h2>Model setup</h2>{error&&<p role="alert">{error}</p>}<button onClick={()=>void run(async()=>apply(await api('model-wizard')))}>Load model setup</button></section>;
  const dirty=JSON.stringify(draft)!==JSON.stringify(saved.selection);
  return <section aria-label="Model setup wizard"><div className="sectionhead"><h2>ZCode / GLM setup</h2><button disabled={busy} onClick={()=>void run(async()=>apply(await api('model-wizard')))}>Reload model setup</button></div>
    <p>Save your intended provider and model, then finish setup in ZCode. Enter account credentials or API keys in ZCode's native settings.</p>
    {error&&<p role="alert">{error}</p>}
    <form onSubmit={e=>{e.preventDefault();void run(async()=>apply(await api('model-wizard','PUT',{revision:saved.revision,...draft})));}}>
      <label>ZCode version<select aria-label="ZCode version" value={draft.client_version} onChange={e=>setDraft({...draft,client_version:e.target.value})}><option value={saved.catalog.client_version}>{saved.catalog.client_version}</option></select></label>
      <label>Model provider<select aria-label="Model provider" value={draft.provider} onChange={e=>setDraft({...draft,provider:e.target.value})}>{saved.catalog.providers.map(p=><option key={p.id} value={p.id}>{p.label}</option>)}</select></label>
      <label>GLM model<select aria-label="GLM model" value={draft.model} onChange={e=>setDraft({...draft,model:e.target.value})}>{saved.catalog.models.map(m=><option key={m}>{m}</option>)}</select></label>
      <label>Native access method<select aria-label="Native access method" value={draft.access_mode} onChange={e=>setDraft({...draft,access_mode:e.target.value})}><option value="native-existing">Use existing ZCode setup</option><option value="account">Account sign-in</option><option value="api-key">API key in ZCode</option></select></label>
      <button disabled={busy||!dirty}>Save model selection</button>
    </form>
    <div className="review"><h3>Saved selection</h3><p data-testid="model-status">{saved.status}</p>{saved.selection&&<><p>{saved.selection.provider} / {saved.selection.model} / {saved.selection.access_mode}</p><p>Revision {saved.revision}. Model call: {saved.inference}.</p>
      <button disabled={busy} onClick={()=>void run(async()=>setHandoff(JSON.stringify(await api('model-wizard/handoff'),null,2)))}>Show native setup steps</button>
      {handoff&&<textarea aria-label="Public model setup handoff" readOnly rows={16} value={handoff}/>}
      <label><input type="checkbox" checked={selected} onChange={e=>setSelected(e.target.checked)}/>I selected this provider and model in ZCode</label>
      <label><input type="checkbox" checked={access} onChange={e=>setAccess(e.target.checked)}/>ZCode reports my provider access is available</label>
      <button disabled={busy||dirty||!selected||!access} onClick={()=>void run(async()=>apply(await api('model-wizard/confirm','POST',{revision:saved.revision,provider_model_selected:selected,native_access_checked:access})))}>Record my native check</button>
      <p>This records your report. Model execution needs a separate test and may consume quota.</p>
    </>}</div>
  </section>;
}

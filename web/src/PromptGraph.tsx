import {useEffect,useRef,useState} from 'react';
import {ReactFlow,Background,Controls,type NodeChange,type Connection,type Node,type Edge,type ReactFlowInstance} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

type Card={id:string,title:string,body:string,x:number,y:number,context:{project:string,environment:string,tenant:string}};
type Document={name:string,prompt:string,cards:Card[],comments:{id:string,text:string,card:string|null}[],links:{id:string,source:string,target:string}[]};
type Graph={id:string,revision:number,document:Document};
type Summary={id:string,name:string,revision:number};
type Api=(path:string,method?:string,data?:unknown)=>Promise<any>;
const blank=():Document=>({name:'Untitled graph',prompt:'',cards:[],comments:[],links:[]});
const error=(e:unknown)=>e instanceof Error?e.message:'REQUEST_FAILED';

export function PromptGraph({api,onDirtyChange}:{api:Api,onDirtyChange:(dirty:boolean)=>void}){
  const [list,setList]=useState<Summary[]>([]),[graph,setGraph]=useState<Graph|null>(null);
  const [doc,setDoc]=useState<Document>(blank),[dirty,setDirty]=useState(false),[busy,setBusy]=useState(false);
  const [selected,setSelected]=useState(''),[peer,setPeer]=useState(''),[comment,setComment]=useState(''),[message,setMessage]=useState('');
  const canvas=useRef<HTMLDivElement>(null),flow=useRef<ReactFlowInstance|null>(null);
  function fitLater(){let second=0;const first=requestAnimationFrame(()=>{second=requestAnimationFrame(()=>{void flow.current?.fitView({padding:0.2});});});return()=>{cancelAnimationFrame(first);cancelAnimationFrame(second);};}
  useEffect(()=>{if(!canvas.current)return;let cancel=()=>{};const observer=new ResizeObserver(()=>{cancel();cancel=fitLater();});observer.observe(canvas.current);return()=>{observer.disconnect();cancel();};},[]);
  useEffect(()=>fitLater(),[graph?.id,doc.cards.length]);
  const card=doc.cards.find(c=>c.id===selected);
  useEffect(()=>{onDirtyChange(dirty);},[dirty]);
  useEffect(()=>{const warn=(e:BeforeUnloadEvent)=>{if(dirty){e.preventDefault();e.returnValue='';}};window.addEventListener('beforeunload',warn);return()=>window.removeEventListener('beforeunload',warn);},[dirty]);
  async function refresh(){setList((await api('graphs')).graphs);}
  useEffect(()=>{void refresh().catch(e=>setMessage(error(e)));},[]);
  function edit(next:Document){setDoc(next);setDirty(true);setMessage('Unsaved changes');}
  async function run(action:()=>Promise<void>){setBusy(true);setMessage('');try{await action();}catch(e){setMessage(error(e));}finally{setBusy(false);}}
  function adopt(g:Graph){setGraph(g);setDoc(g.document);setDirty(false);setSelected('');setPeer('');setComment('');}
  async function load(id:string){if(dirty){setMessage('Save or reload before switching graphs.');return;}adopt(await api('graphs/'+id));await refresh();}
  async function save(){const g=graph?await api('graphs/'+graph.id,'PUT',{revision:graph.revision,document:doc}):await api('graphs','POST',{document:doc});adopt(g);await refresh();setMessage('Saved revision '+g.revision);}
  function addCard(){const n=doc.cards.length;if(n>=24){setMessage('Card limit reached');return;}const c:Card={id:crypto.randomUUID(),title:'Card '+(n+1),body:'',x:(n%3)*240,y:Math.floor(n/3)*150,context:{project:'',environment:'',tenant:''}};edit({...doc,cards:[...doc.cards,c]});setSelected(c.id);}
  function changeCard(update:Partial<Card>){if(card)edit({...doc,cards:doc.cards.map(c=>c.id===card.id?{...c,...update}:c)});}
  function link(a:string,b:string){if(!a||!b||a===b||doc.links.some(l=>(l.source===a&&l.target===b)||(l.source===b&&l.target===a))){setMessage('Choose two different, unlinked cards.');return;}if(doc.links.length>=24){setMessage('Relation limit reached');return;}edit({...doc,links:[...doc.links,{id:crypto.randomUUID(),source:a,target:b}]});}
  function positions(changes:NodeChange[]){const moved=changes.filter(c=>c.type==='position'&&c.position);if(!moved.length)return;setDoc(current=>({...current,cards:current.cards.map(c=>{const change=moved.find(n=>n.type==='position'&&n.id===c.id);return change?.type==='position'&&change.position?{...c,x:Math.round(change.position.x),y:Math.round(change.position.y)}:c;})}));setDirty(true);}
  async function exported(){if(!graph||dirty)throw new Error('Save changes before sharing.');const out=await api('graphs/'+graph.id+'/markdown');if(out.revision!==graph.revision)throw new Error('REVISION_CONFLICT');return out;}
  async function copy(){const out=await exported();try{await navigator.clipboard.writeText(out.markdown);}catch{setMessage('Clipboard failed; no successful copy recorded.');return;}try{await api('graphs/'+graph!.id+'/copies','POST',{revision:out.revision});setMessage('Copied redacted Markdown; success reported.');}catch(e){setMessage('Copied locally; success report not recorded: '+error(e));}}
  async function download(){const out=await exported();const url=URL.createObjectURL(new Blob([out.markdown],{type:'text/markdown;charset=utf-8'}));const a=document.createElement('a');a.href=url;a.download='prompt-graph.md';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);setMessage('Downloaded redacted Markdown.');}
  const nodes:Node[]=doc.cards.map(c=>({id:c.id,position:{x:c.x,y:c.y},data:{label:c.title},selected:c.id===selected}));
  const edges:Edge[]=doc.links.map(l=>({...l,type:'straight'}));
  return <section className="prompt-graph">
    <div className="sectionhead"><div><h2>Prompt Graph</h2><p>Compose context. Relations never run agents or infrastructure.</p></div><button className="quiet" disabled={busy} onClick={()=>{if(dirty){setMessage('Save or reload before starting another graph.');return;}setGraph(null);setDoc(blank());setSelected('');setMessage('New unsaved graph');}}>New graph</button></div>
    <p className="muted">Prompts are not a secret store. Recognized credentials are redacted on copy/export; review before sharing.</p>
    <label>Saved graph<select aria-label="Saved graph" value={graph?.id||''} disabled={busy} onChange={e=>{if(e.target.value)void run(()=>load(e.target.value));}}><option value="">New graph</option>{list.map(g=><option key={g.id} value={g.id}>{g.name} · revision {g.revision}</option>)}</select></label>
    <div className="graph-toolbar"><button disabled={busy} onClick={()=>void run(save)}>Save graph</button><button className="quiet" disabled={busy||(!graph&&!dirty)} onClick={()=>void run(async()=>{if(dirty&&!window.confirm('Discard unsaved graph changes?'))return;if(graph){adopt(await api('graphs/'+graph.id));await refresh();setMessage('Reloaded saved graph.');}else{setDoc(blank());setDirty(false);setSelected('');setMessage('Draft discarded.');}})}>{graph?'Reload saved graph':'Discard draft'}</button><button className="quiet" disabled={busy||!graph||dirty} onClick={()=>void run(copy)}>Copy Markdown</button><button className="quiet" disabled={busy||!graph||dirty} onClick={()=>void run(download)}>Export Markdown</button><button className="quiet" disabled={busy||!graph} onClick={()=>void run(async()=>{if(!window.confirm('Delete this graph only? Environments are unchanged.'))return;await api('graphs/'+graph!.id,'DELETE',{revision:graph!.revision});setGraph(null);setDoc(blank());setDirty(false);setSelected('');await refresh();setMessage('Graph deleted. Environments unchanged.');})}>Delete graph</button></div>
    <p role="status" aria-live="polite">{message||'Explicit saves · '+(graph?'revision '+graph.revision:'not saved')}</p>
    <label>Graph name<input maxLength={80} value={doc.name} onChange={e=>edit({...doc,name:e.target.value})}/></label>
    <label>Graph prompt<textarea maxLength={2000} value={doc.prompt} onChange={e=>edit({...doc,prompt:e.target.value})}/></label>
    <div className="graph-toolbar"><button disabled={busy} onClick={addCard}>Add card</button><label>Selected card<select aria-label="Selected card" value={selected} onChange={e=>setSelected(e.target.value)}><option value="">Graph comments</option>{doc.cards.map(c=><option key={c.id} value={c.id}>{c.title}</option>)}</select></label></div>
    <div ref={canvas} className="graph-canvas" aria-label="Prompt composition canvas"><ReactFlow onInit={instance=>{flow.current=instance;}} nodes={nodes} edges={edges} onNodesChange={positions} onNodeClick={(_,node)=>setSelected(node.id)} onConnect={(c:Connection)=>link(c.source,c.target)} deleteKeyCode={null} fitView><Background/><Controls showInteractive={false}/></ReactFlow></div>
    {card&&<div className="graph-editor"><h3>Edit selected card</h3><label>Card title<input maxLength={100} value={card.title} onChange={e=>changeCard({title:e.target.value})}/></label><label>Card text<textarea maxLength={2000} value={card.body} onChange={e=>changeCard({body:e.target.value})}/></label>
      {(['project','environment','tenant'] as const).map(key=><label key={key}>{key[0].toUpperCase()+key.slice(1)+' context'}<input maxLength={120} value={card.context[key]} onChange={e=>changeCard({context:{...card.context,[key]:e.target.value}})}/></label>)}
      <div className="graph-toolbar"><label>Link selected card to<select aria-label="Link selected card to" value={peer} onChange={e=>setPeer(e.target.value)}><option value="">Choose another card</option>{doc.cards.filter(c=>c.id!==card.id).map(c=><option key={c.id} value={c.id}>{c.title}</option>)}</select></label><button disabled={busy||!peer} onClick={()=>link(card.id,peer)}>Add relation</button><button className="quiet" onClick={()=>{edit({...doc,cards:doc.cards.filter(c=>c.id!==card.id),links:doc.links.filter(l=>l.source!==card.id&&l.target!==card.id),comments:doc.comments.filter(c=>c.card!==card.id)});setSelected('');}}>Remove card</button></div></div>}
    {doc.links.length>0&&<div><h3>Undirected relations</h3>{doc.links.map(l=><p key={l.id}>{doc.cards.find(c=>c.id===l.source)?.title} ↔ {doc.cards.find(c=>c.id===l.target)?.title} <button className="quiet" onClick={()=>edit({...doc,links:doc.links.filter(x=>x.id!==l.id)})}>Remove relation</button></p>)}</div>}
    <div className="graph-editor"><h3>{card?'Comments on '+card.title:'Graph comments'}</h3><label>Comment text<textarea maxLength={1000} value={comment} onChange={e=>setComment(e.target.value)}/></label><button disabled={!comment.trim()||doc.comments.length>=24} onClick={()=>{edit({...doc,comments:[...doc.comments,{id:crypto.randomUUID(),text:comment,card:card?.id||null}]});setComment('');}}>Add comment</button>{doc.comments.map((c,index)=><div className="graph-editor" key={c.id}><label>{c.card?doc.cards.find(n=>n.id===c.card)?.title:'Graph comment'}<textarea aria-label={'Edit comment '+(index+1)} maxLength={1000} value={c.text} onChange={e=>edit({...doc,comments:doc.comments.map(x=>x.id===c.id?{...x,text:e.target.value}:x)})}/></label><button className="quiet" onClick={()=>edit({...doc,comments:doc.comments.filter(x=>x.id!==c.id)})}>Remove comment</button></div>)}</div>
  </section>;
}

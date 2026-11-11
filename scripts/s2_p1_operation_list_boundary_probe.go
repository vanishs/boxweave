//go:build ignore

// Real installed MCP operation_list mid-query revocation/expiry. A separate
// debugger halts the unchanged candidate binary on the exact source line
// AFTER state.ListOperations and BEFORE privateReadReply; no prod hook exists.
package main

import (
 "context"
 "encoding/json"
 "fmt"
 "net/http"
 "os"
 "path/filepath"
 "strings"
 "time"

 "github.com/modelcontextprotocol/go-sdk/mcp"
 "github.com/vanishs/boxweave/internal/state"
)

type authHeader struct{token string}
func (a authHeader) RoundTrip(r *http.Request)(*http.Response,error){
 cp:=r.Clone(r.Context());cp.Header.Set("Authorization","Bearer "+a.token)
 return http.DefaultTransport.RoundTrip(cp)
}
func must(ok bool,why string){if !ok{panic("S2_P1_INFLIGHT_LIST_FAIL: "+why)}}
type result struct{r *mcp.CallToolResult;e error}
func runCase(ctx context.Context,st *state.Store,project,dir,mode string){
 ttl:=time.Hour
 if mode=="expire"{ttl=6*time.Second}
 raw,token,e:=st.IssueToken(ctx,"viewer",ttl)
 must(e==nil&&len(raw)>0&&token.ID!="","fresh credential issuance")
 rec,e:=st.Admit(ctx,state.Admission{
  Layer:project,ActorID:token.ID,TokenID:token.ID,
  Key:"s2-p1-debugger-list-"+mode,RequestID:"debugger-list-"+mode,
  Action:"box_exec",SafeIntent:[]byte(`{"kind":"never-dispatched-debugger-list"}`),
  WorkRef:"lab/p1-debugger-list-"+mode,AgentRef:"source-boundary",
 })
 must(e==nil&&rec.State=="ADMITTED","original scoped list fixture not admitted")
 cli:=mcp.NewClient(&mcp.Implementation{Name:"s2-p1-real-list-boundary",Version:"1"},nil)
 s,e:=cli.Connect(ctx,&mcp.StreamableClientTransport{
  Endpoint:"http://127.0.0.1:18667/mcp",
  HTTPClient:&http.Client{Transport:authHeader{token:raw}},
  DisableStandaloneSSE:true,MaxRetries:-1,
 },nil)
 must(e==nil&&s!=nil,"cannot initialize real installed MCP")
 defer s.Close()
 arm:=filepath.Join(dir,"armed")
 held:=filepath.Join(dir,"held")
 release:=filepath.Join(dir,"release")
 for _,p:=range []string{arm,held,release}{_,err:=os.Stat(p);must(os.IsNotExist(err),"old debugger signal exists")}
 must(os.WriteFile(arm,[]byte(mode+"\n"),0600)==nil,"cannot arm exact instruction boundary")
 done:=make(chan result,1)
 go func(){
  r,e:=s.CallTool(ctx,&mcp.CallToolParams{Name:"operation_list",Arguments:map[string]any{
   "work_ref":"lab/p1-debugger-list-"+mode,"limit":10,
  }})
  done<-result{r,e}
 }()
 observed:=false
 for i:=0;i<120;i++{
  if _,err:=os.Stat(held);err==nil{observed=true;break}
  select{case <-done:panic("S2_P1_INFLIGHT_LIST_FAIL: query returned before debugger stopped final boundary")
  default:}
  time.Sleep(100*time.Millisecond)
 }
 must(observed,"original binary did not hit post-SQL pre-privateReadReply source line")
 if mode=="expire"{
  until:=time.Until(token.Expires.Add(time.Second))
  if until>0{select{
   case <-done:panic("S2_P1_INFLIGHT_LIST_FAIL: pending response unexpectedly completed before expiry")
   case <-time.After(until):
  }}
 }else{
  must(st.RevokeToken(ctx,token.ID)==nil,"could not revoke exact suspended bearer")
 }
 _,e=st.VerifyToken(ctx,raw)
 must(e!=nil,"bearer not disqualified at exact held source line")
 must(os.WriteFile(release,[]byte("credentials invalidated; resume unchanged candidate\n"),0600)==nil,
  "cannot resume exact binary")
 var out result
 select{
 case out=<-done:
 case <-time.After(14*time.Second):panic("S2_P1_INFLIGHT_LIST_FAIL: genuine MCP list never returned after debugger resume")
 }
 must(out.e!=nil||out.r!=nil&&out.r.IsError,"private historical operations leaked to invalidated token")
 if out.r!=nil{
  b,e:=json.Marshal(out.r.StructuredContent)
  must(e==nil&&!strings.Contains(string(b),rec.ID),"scoped private original disclosed after auth loss")
 }
 original,e:=st.Get(ctx,rec.ID)
 must(e==nil&&original.State=="ADMITTED"&&original.NativeOperationID=="",
  "read-disconnect changed original operation or dispatched native effect")
 for _,p:=range []string{arm,held,release}{must(os.Remove(p)==nil,"debugger source-boundary signal was not cleared")}
 label:="REVOKED"
 if mode=="expire"{label="EXPIRED"}
 fmt.Println("S2_P1_INSTALLED_OPERATION_LIST_POST_QUERY_PRE_RESPONSE_"+label+"_NO_LEAK=PASS")
}
func main(){
 dir,project:=os.Getenv("BW_S2_LIST_LAB"),os.Getenv("BW_S2_LIST_PROJECT")
 must(os.Geteuid()!=0&&os.Getenv("GITHUB_ACTIONS")=="true"&&
  os.Getenv("RUNNER_ENVIRONMENT")=="github-hosted"&&
  strings.HasPrefix(dir,"/tmp/bw-v1-s2-ghci.")&&filepath.Base(dir)=="p1-list-boundary"&&
  project=="user-"+os.Getenv("BW_S2_TEST_UID"),"only real disposable restricted CI")
 ctx,cancel:=context.WithTimeout(context.Background(),115*time.Second)
 defer cancel()
 st,e:=state.Open(filepath.Join(dir,"ledger.sqlite"))
 must(e==nil,"cannot open service's exact disposable ledger")
 defer st.Close()
 runCase(ctx,st,project,dir,"revoke")
 runCase(ctx,st,project,dir,"expire")
 fmt.Println("S2_P1_INSTALLED_OPERATION_LIST_TWO_INFLIGHT_AUTH_RACES=PASS")
}

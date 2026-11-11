//go:build ignore

// CI-only official MCP SDK proof of credential revocation/expiry during
// a private operation_get long poll. Controller-only ADMITTED rows, no Incus writes.
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

type bearer struct{ token string }
func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func check(ok bool, reason string) {
	if !ok { panic("S2_P1_TOKEN_INFLIGHT_FAIL: "+reason) }
}
func session(ctx context.Context, token string) *mcp.ClientSession {
	client := mcp.NewClient(&mcp.Implementation{Name:"s2-inflight-auth-blackbox",Version:"1"},nil)
	s,err := client.Connect(ctx,&mcp.StreamableClientTransport{
		Endpoint:"http://127.0.0.1:18659/mcp",
		HTTPClient:&http.Client{Transport:bearer{token}},
		DisableStandaloneSSE:true,MaxRetries:-1,
	},nil)
	check(err==nil && s!=nil,"authenticated MCP session not available")
	return s
}
type reply struct{ result *mcp.CallToolResult; err error }
func one(ctx context.Context,st *state.Store,project,mode string) {
	ttl:=time.Hour
	if mode=="expire" {ttl=3*time.Second}
	raw,actor,err:=st.IssueToken(ctx,"viewer",ttl)
	check(err==nil && len(raw)==68 && actor.ID!="","test viewer not created")
	original,err:=st.Admit(ctx,state.Admission{
		Layer:project,ActorID:actor.ID,TokenID:actor.ID,
		Key:"s2-p1-inflight-"+mode+"-original-key",RequestID:"initial-private-poll",
		Action:"box_exec",SafeIntent:[]byte("{\"kind\":\"never-dispatched\"}"),
		WorkRef:"lab/inflight-credential",AgentRef:mode,
	})
	check(err==nil && original.State=="ADMITTED","test-only original not journaled")
	_,err=st.VerifyToken(ctx,raw)
	check(err==nil,"test bearer was invalid before request")
	s:=session(ctx,raw)
	defer s.Close()
	done:=make(chan reply,1)
	go func(){
		out,e:=s.CallTool(ctx,&mcp.CallToolParams{
			Name:"operation_get",Arguments:map[string]any{"operation_id":original.ID,"wait_ms":5000},
		})
		done<-reply{out,e}
	}()
	select {
	case <-done: panic("S2_P1_TOKEN_INFLIGHT_FAIL: request completed before invalidation")
	case <-time.After(750*time.Millisecond):
	}
	if mode=="revoke" {
		err=st.RevokeToken(ctx,actor.ID)
		check(err==nil,"exact test credential revocation failed")
	}
	out:=<-done
	check(out.err==nil || !strings.Contains(out.err.Error(),"deadline exceeded"),
		"transport timed out instead of checking live credential")
	check(out.err!=nil || out.result!=nil && out.result.IsError,
		"invalidated credential received private operation result")
	if out.result!=nil {
		b,e:=json.Marshal(out.result.StructuredContent)
		check(e==nil && !strings.Contains(string(b),original.ID),
			"private original included in invalidated credential response")
	}
	_,err=st.VerifyToken(ctx,raw)
	check(err!=nil,"credential was not expired/revoked at response")
	saved,err:=st.Get(ctx,original.ID)
	check(err==nil && saved.ID==original.ID && saved.State=="ADMITTED" &&
		saved.NativeOperationID=="","credential read affected original operation state")
	if mode=="revoke" {fmt.Println("S2_P1_INSTALLED_MCP_REVOKED_DURING_READ_NO_LEAK=PASS")}
	if mode=="expire" {fmt.Println("S2_P1_INSTALLED_MCP_EXPIRED_DURING_READ_NO_LEAK=PASS")}
}
func main(){
	dir,project:=os.Getenv("BW_S2_AUTH_LAB"),os.Getenv("BW_S2_AUTH_PROJECT")
	check(os.Geteuid()!=0 && os.Getenv("GITHUB_ACTIONS")=="true" &&
		os.Getenv("RUNNER_ENVIRONMENT")=="github-hosted" &&
		strings.HasPrefix(dir,"/tmp/bw-v1-s2-ghci.") &&
		filepath.Base(dir)=="p1-token-inflight" &&
		strings.HasPrefix(project,"user-"),"test outside disposable confined CI")
	ctx,cancel:=context.WithTimeout(context.Background(),27*time.Second)
	defer cancel()
	st,err:=state.Open(filepath.Join(dir,"ledger.sqlite"))
	check(err==nil,"installed test SQLite unavailable")
	defer st.Close()
	one(ctx,st,project,"revoke")
	one(ctx,st,project,"expire")
	fmt.Println("S2_P1_INSTALLED_INFLIGHT_BEARER_REVOCATION_AND_EXPIRY=PASS")
}

//go:build ignore

// Single-use malformed real Incus Exec Operation readback. The HTTP 202 and
// guest effect are genuine; only an already-final native response is corrupted.
// Never install or run against an existing/shared Project.
package main

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "io"
 "log"
 "net"
 "net/http"
 "net/http/httputil"
 "net/url"
 "os"
 "path/filepath"
 "regexp"
 "strings"
 "sync/atomic"
 "time"
)

var uuidPattern=regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
func marker(path,text string) error {
 f,e:=os.OpenFile(path,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600)
 if e!=nil{return e}
 _,e=io.WriteString(f,text)
 if e==nil{e=f.Sync()}
 if closeErr:=f.Close();e==nil{e=closeErr}
 return e
}
func main(){
 listen:=flag.String("listen","","private Unix listener")
 upstream:=flag.String("upstream","","restricted upstream native socket")
 project:=flag.String("project","","unique restricted native project")
 name:=flag.String("name","","unique artificial Box name")
 lab:=flag.String("lab","","CI-only evidence directory")
 flag.Parse()
 if os.Geteuid()==0||os.Getenv("GITHUB_ACTIONS")!="true"||
  os.Getenv("RUNNER_ENVIRONMENT")!="github-hosted"||
  !strings.HasPrefix(*lab,"/tmp/bw-v1-s2-ghci.")||
  filepath.Base(*lab)!="p2-malformed-exec"||
  *listen!=filepath.Join(*lab,"proxy.sock")||
  *upstream!="/var/lib/incus/unix.socket.user"||
  *project!="user-"+os.Getenv("BW_S2_TEST_UID")||
  *name!="bw-s2-p2-malformed-exec-110v"{
  log.Fatal("S2_P2_MALFORMED_PROXY_REFUSED: only original disposable CI")
 }
 if _,e:=os.Lstat(*listen);!errors.Is(e,os.ErrNotExist){log.Fatal("existing listener refused")}
 accepted:=filepath.Join(*lab,"native-accepted.txt")
 injected:=filepath.Join(*lab,"native-malformed.txt")
 doubled:=filepath.Join(*lab,"native-duplicate.txt")
 transport:=&http.Transport{DisableKeepAlives:true,
  DialContext:func(ctx context.Context,_,_ string)(net.Conn,error){
   return (&net.Dialer{Timeout:5*time.Second}).DialContext(ctx,"unix",*upstream)
  }}
 reverse:=httputil.NewSingleHostReverseProxy(&url.URL{Scheme:"http",Host:"unix.socket"})
 reverse.Transport=transport
 reverse.ErrorHandler=func(w http.ResponseWriter,_ *http.Request,_ error){http.Error(w,"unavailable native socket",502)}
 var fired atomic.Bool
 var nativeID atomic.Value
 handler:=http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Method==http.MethodPost{
   if r.URL.Path!="/1.0/instances/"+*name+"/exec"||
    r.URL.Query().Get("project")!=*project||!fired.CompareAndSwap(false,true){
    _=marker(doubled,"SECOND_OR_FOREIGN_NATIVE_EXEC_REFUSED\n")
    http.Error(w,"unapproved native mutation",409);return
   }
   raw,e:=io.ReadAll(io.LimitReader(r.Body,1<<16))
   if e!=nil{http.Error(w,"native request unreadable",400);return}
   var spec struct{Command []string `json:"command"`;RecordOutput bool `json:"record-output"`}
   if json.Unmarshal(raw,&spec)!=nil||len(spec.Command)<2||!spec.RecordOutput{
    http.Error(w,"unexpected original guest effect",400);return
   }
   req:=r.Clone(r.Context());req.URL.Scheme,req.URL.Host,req.RequestURI="http","unix.socket",""
   req.Body=io.NopCloser(bytes.NewReader(raw));req.ContentLength=int64(len(raw))
   res,e:=transport.RoundTrip(req);if e!=nil{http.Error(w,"native exec unavailable",502);return}
   defer res.Body.Close()
   body,e:=io.ReadAll(io.LimitReader(res.Body,1<<20))
   var v struct{Operation string `json:"operation"`}
   if e!=nil||res.StatusCode!=http.StatusAccepted||json.Unmarshal(body,&v)!=nil{
    http.Error(w,"native original not accepted",502);return
   }
   u,e:=url.Parse(v.Operation)
   id:=""
   if e==nil&&u!=nil{id=strings.TrimPrefix(u.Path,"/1.0/operations/")}
   if !uuidPattern.MatchString(id){http.Error(w,"native operation ID absent",502);return}
   nativeID.Store(id)
   if e=marker(accepted,"REAL_EXEC_202_NATIVE_ID="+id+"\n");e!=nil{http.Error(w,"acceptance witness missing",502);return}
   for key,values:=range res.Header{for _,val:=range values{w.Header().Add(key,val)}}
   w.WriteHeader(res.StatusCode);_,_=w.Write(body);return
  }
  if r.Method!=http.MethodGet{
   _=marker(doubled,"UNAPPROVED_NATIVE_MUTATION_REFUSED\n")
   http.Error(w,"only original native Exec permitted",409);return
  }
  id,ok:=nativeID.Load().(string)
  if !ok||r.URL.Path!="/1.0/operations/"+id||r.URL.Query().Get("project")!=*project{
   reverse.ServeHTTP(w,r);return
  }
  req:=r.Clone(r.Context());req.URL.Scheme,req.URL.Host,req.RequestURI="http","unix.socket",""
  res,e:=transport.RoundTrip(req);if e!=nil{http.Error(w,"native read failed",502);return}
  defer res.Body.Close()
  original,e:=io.ReadAll(io.LimitReader(res.Body,1<<20))
  if e!=nil||res.StatusCode!=200{http.Error(w,"native operation read failed",502);return}
  var root map[string]any
  if json.Unmarshal(original,&root)!=nil{http.Error(w,"native operation envelope unreadable",502);return}
  op,_:=root["metadata"].(map[string]any)
  details,_:=op["metadata"].(map[string]any)
  if op!=nil&&details!=nil&&op["status_code"]==float64(200){
   if _,has:=details["return"];has {
    // Preserve the actual native success/IDs, corrupt only a final guest
    // return type. The BoxWeave worker must NOT infer successful execution.
    details["return"]="malformed-noninteger"
    changed,e:=json.Marshal(root);if e!=nil{http.Error(w,"fault not serializable",500);return}
    original=changed
    if _,e:=os.Stat(injected);errors.Is(e,os.ErrNotExist){
     if e=marker(injected,fmt.Sprintf("MALFORMED_FINAL_NATIVE_ID=%s\n",id));e!=nil{
      http.Error(w,"fault witness missing",500);return
     }
    }
   }
  }
  for k,v:=range res.Header{if strings.EqualFold(k,"Content-Length"){continue};for _,s:=range v{w.Header().Add(k,s)}}
  w.WriteHeader(res.StatusCode);_,_=w.Write(original)
 })
 listener,e:=net.Listen("unix",*listen);if e!=nil{log.Fatal(e)}
 if e=os.Chmod(*listen,0600);e!=nil{log.Fatal(e)}
 log.Println("S2_P2_REAL_NATIVE_FINAL_METADATA_PROXY_READY")
 if e=(&http.Server{Handler:handler,ReadHeaderTimeout:5*time.Second}).Serve(listener);e!=nil&&!errors.Is(e,http.ErrServerClosed){log.Fatal(e)}
}

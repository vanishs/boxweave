// BoxWeave v1 host-local CLI; no general host-shell or public REST surface.
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/vanishs/boxweave/internal/control"
    "github.com/vanishs/boxweave/internal/install"
    "github.com/vanishs/boxweave/internal/native"
    "github.com/vanishs/boxweave/internal/state"
)

var (
    version="0.0.0-s1-dev"
    commit="unverified"
)
func usage() {fmt.Fprintln(os.Stderr,"usage: boxweave serve|status|doctor|token-create|token-revoke|install-stage|install-status [flags]")}
func run()error{
    if len(os.Args)<2 {usage();return errors.New("subcommand required")}
    command:=os.Args[1]
    f:=flag.NewFlagSet(command,flag.ContinueOnError)
    db:=f.String("db","/var/lib/boxweave/ledger.sqlite","protected local SQLite path")
    socket:=f.String("incus-socket","/var/lib/incus/unix.socket.user","restricted Incus Unix socket")
    project:=f.String("project","","installer-verified confined project; mandatory for native reads")
    listener:=f.String("listen","127.0.0.1:18650","loopback management HTTP listen address")
    role:=f.String("role","viewer","token role")
    ttl:=f.Duration("ttl",30*24*time.Hour,"token validity")
    tokenID:=f.String("id","","exact local credential ID")
    serviceUser:=f.String("service-user","boxweave","dedicated nonroot incus-only service account")
    sourceBinary:=f.String("source-binary","","source executable at approved exact digest")
    digest:=f.String("expect-sha256","","locally reviewed binary sha256")
    approve:=f.Bool("approve-stage",false,"explicitly authorize staging service files; no automatic start")
    if err:=f.Parse(os.Args[2:]);err!=nil{return err}
    if f.NArg()!=0{return errors.New("unexpected arguments")}
    switch command {
    case "serve","status","doctor","token-create","token-revoke","install-stage","install-status":
    default: usage();return errors.New("unsupported command")
    }
    if command=="install-stage"||command=="install-status" {
        p:=install.Plan{User:*serviceUser,StateDir:"/var/lib/boxweave",
            BinaryPath:"/usr/local/bin/boxweave",UnitPath:"/etc/systemd/system/boxweave.service",
            SourceBinary:*sourceBinary,Project:*project,Socket:*socket,Version:version,
            ExpectedDigest:*digest}
        if command=="install-stage" {
            m,err:=install.Stage(p,*approve)
            if err!=nil{return err}
            // Staging is not install acceptance; systemd and Incus remain untouched.
            return json.NewEncoder(os.Stdout).Encode(m)
        }
        m,err:=install.Status(p)
        if err!=nil{return err}
        return json.NewEncoder(os.Stdout).Encode(m)
    }
    s,err:=state.Open(*db);if err!=nil{return err}
    defer s.Close()
    ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
    // Never blindly replay an operation possibly sent to Incus before a restart.
    if command=="serve"{
        if _,err:=s.Recover(ctx);err!=nil{return err}
    }
    svc:=control.Service{Store:s,Native:native.Reader{Socket:*socket,Project:*project},BuildVersion:version,BuildCommit:commit,Listener:*listener}
    switch command{
    case "serve":
        return svc.ListenAndServe()
    case "status","doctor":
        status:=svc.Status(ctx)
        enc:=json.NewEncoder(os.Stdout);enc.SetIndent("","  ")
        if err:=enc.Encode(status);err!=nil{return err}
        if command=="doctor" && !status.Ready{return errors.New("preflight incomplete")}
        return nil
    case "token-create":
        if os.Geteuid()==0 {return errors.New("token creation must use the dedicated local service identity")}
        raw,t,err:=s.IssueToken(ctx,*role,*ttl);if err!=nil{return err}
        // Printed once, only following an explicit host-local invocation.
        fmt.Fprintf(os.Stdout,"id=%s role=%s expires=%s\ntoken=%s\n",t.ID,t.Role,t.Expires.Format(time.RFC3339),raw)
        return nil
    case "token-revoke":
        if os.Geteuid()==0{return errors.New("token revocation must use the dedicated local service identity")}
        if *tokenID==""{return errors.New("exact token ID required")}
        return s.RevokeToken(ctx,*tokenID)
    }
    return errors.New("unreachable")
}
func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,"boxweave:",err);os.Exit(1)}}

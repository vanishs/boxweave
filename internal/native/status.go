// Package native provides read-only, project-confined official Incus SDK access.
package native

import (
    "context"
    "errors"
    "fmt"
    "os"
    "slices"

    incus "github.com/lxc/incus/v7/client"
)

type Status struct {
    Ready bool `json:"ready"`
    Version string `json:"incus_version,omitempty"`
    Project string `json:"project"`
    ProfileReady bool `json:"default_profile_ready"`
    APIExtensions map[string]bool `json:"required_extensions"`
    ErrorClass string `json:"error_class,omitempty"`
}

type Reader struct { Socket string; Project string }

var requiredExtensions=[]string{"projects","restricted","container_exec_recording"}

func(r Reader) Read(ctx context.Context)Status{
    s:=Status{Project:r.Project,APIExtensions:map[string]bool{}}
    if os.Geteuid()==0{s.ErrorClass="REFUSE_ROOT_RUNTIME";return s}
    if r.Project==""||r.Socket=="" {s.ErrorClass="MISSING_NATIVE_CONFIG";return s}
    c,err:=incus.ConnectIncusUnixWithContext(ctx,r.Socket,nil)
    if err!=nil{s.ErrorClass="INCUS_CONNECT_FAILED";return s}
    names,err:=c.GetProjectNames()
    if err!=nil{s.ErrorClass="INCUS_PROJECT_QUERY_FAILED";return s}
    if !slices.Contains(names,r.Project){s.ErrorClass="PROJECT_NOT_AUTHORIZED";return s}
    // UseProject is necessary, but native OS credentials must ALSO enforce confinement.
    scoped:=c.UseProject(r.Project)
    server,_,err:=scoped.GetServer()
    if err!=nil{s.ErrorClass="INCUS_SERVER_QUERY_FAILED";return s}
    s.Version=server.Environment.ServerVersion
    for _,ext:=range requiredExtensions {s.APIExtensions[ext]=slices.Contains(server.APIExtensions,ext)}
    profile,_,err:=scoped.GetProfile("default")
    if err!=nil{s.ErrorClass="PROFILE_UNAVAILABLE";return s}
    for _,d:=range profile.Devices {
        if d["type"]=="disk" && d["path"]=="/" && d["pool"]!="" {s.ProfileReady=true;break}
    }
    if !s.ProfileReady {s.ErrorClass="ROOT_DISK_NOT_CONFIGURED";return s}
    s.Ready=true
    return s
}

func(r Reader) Strict(ctx context.Context)error {
    v:=r.Read(ctx);if !v.Ready{return fmt.Errorf("native preflight: %s",v.ErrorClass)}
    if !v.APIExtensions["projects"] {return errors.New("native projects extension absent")}
    return nil
}

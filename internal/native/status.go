// Package native provides read-only, project-confined official Incus SDK access.
package native

import (
    "context"
    "errors"
    "fmt"
    "os"
    "os/user"
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
    current,err:=user.Current()
    if err!=nil{s.ErrorClass="OS_IDENTITY_UNAVAILABLE";return s}
    ids,err:=current.GroupIds()
    if err!=nil{s.ErrorClass="OS_GROUPS_UNAVAILABLE";return s}
    incusGroup,err:=user.LookupGroup("incus")
    if err!=nil{s.ErrorClass="INCUS_GROUP_MISSING";return s}
    if !slices.Contains(ids,incusGroup.Gid) {s.ErrorClass="INCUS_USER_GROUP_REQUIRED";return s}
    for _,forbidden:=range []string{"incus-admin","sudo","docker","lxd"}{
        g,e:=user.LookupGroup(forbidden)
        if e==nil && slices.Contains(ids,g.Gid){s.ErrorClass="UNSAFE_OS_PRIVILEGE";return s}
    }
    if r.Project==""||r.Socket=="" {s.ErrorClass="MISSING_NATIVE_CONFIG";return s}
    c,err:=incus.ConnectIncusUnixWithContext(ctx,r.Socket,nil)
    if err!=nil{s.ErrorClass="INCUS_CONNECT_FAILED";return s}
    names,err:=c.GetProjectNames()
    if err!=nil{s.ErrorClass="INCUS_PROJECT_QUERY_FAILED";return s}
    if !slices.Contains(names,r.Project)||len(names)!=1{s.ErrorClass="PROJECT_NOT_EXCLUSIVELY_CONFINED";return s}
    // UseProject is necessary, but native OS credentials must ALSO enforce confinement.
    scoped:=c.UseProject(r.Project)
    project,_,err:=scoped.GetProject(r.Project)
    if err!=nil{s.ErrorClass="PROJECT_CONFIG_UNAVAILABLE";return s}
    if project.Config["restricted"]!="true"{s.ErrorClass="PROJECT_NOT_RESTRICTED";return s}
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
    for _,required:=range requiredExtensions{
        if !s.APIExtensions[required]{s.ErrorClass="REQUIRED_EXTENSION_MISSING";return s}
    }
    s.Ready=true
    return s
}

func(r Reader) Strict(ctx context.Context)error {
    v:=r.Read(ctx);if !v.Ready{return fmt.Errorf("native preflight: %s",v.ErrorClass)}
    if !v.APIExtensions["projects"] {return errors.New("native projects extension absent")}
    return nil
}

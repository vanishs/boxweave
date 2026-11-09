package install

import (
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "os/user"
    "regexp"
    "strings"
)

// NativeBootstrap only supports a fresh, approved Ubuntu 24.04 guest. It never
// upgrades an existing daemon or changes shared Incus data.
type NativeBootstrapReport struct {
    Mode string `json:"mode"`
    PackageVersion string `json:"incus_package_version"`
    PlanSHA256 string `json:"plan_sha256,omitempty"`
    AptChanges []string `json:"apt_changes,omitempty"`
    RebootRequired bool `json:"reboot_required"`
}
var ubuntuIncusVersion = regexp.MustCompile("^[0-9A-Za-z.+:~-]{3,80}$")

func installedIncusPackage() (string,bool,error) {
    cmd:=exec.Command("dpkg-query","-W","-f=${Status} ${Version}","incus")
    b,err:=cmd.CombinedOutput()
    if err!=nil {
        if strings.Contains(string(b),"no packages found") {return "",false,nil}
        return "",false,fmt.Errorf("unknown Incus package state: %w",err)
    }
    s:=strings.TrimSpace(string(b))
    if strings.HasPrefix(s,"install ok installed "){
        return strings.TrimPrefix(s,"install ok installed "),true,nil
    }
    return "",false,errors.New("partial/residual Incus dpkg state requires manual review")
}
func ubuntuNativeBaseline() error {
    b,err:=os.ReadFile("/etc/os-release")
    if err!=nil{return err}
    s:="\n"+string(b)
    if !strings.Contains(s,"\nID=ubuntu\n") ||
        (!strings.Contains(s,"VERSION_ID=\"24.04\"") && !strings.Contains(s,"VERSION_ID=24.04")){
        return errors.New("native auto bootstrap is certified only for Ubuntu 24.04")
    }
    return nil
}
func nativePlan(packageVersion string)(NativeBootstrapReport,error) {
    p:=NativeBootstrapReport{PackageVersion:packageVersion}
    if err:=ubuntuNativeBaseline();err!=nil{return p,err}
    if !ubuntuIncusVersion.MatchString(packageVersion){
        return p,errors.New("exact pinned Incus Ubuntu package version required")
    }
    pkg,installed,err:=installedIncusPackage()
    if err!=nil{return p,err}
    if installed {
        if pkg!=packageVersion {
            return p,fmt.Errorf("existing Incus package %s differs from %s: shared daemon upgrade requires separate authorization",pkg,packageVersion)
        }
        if _,err=os.Lstat("/var/lib/incus/unix.socket.user");err!=nil{
            return p,errors.New("existing daemon is not ready: refuse automatic restart")
        }
        p.Mode="reuse-existing-daemon"
        return p,nil
    }
    if _,err=os.Lstat("/var/lib/incus");err==nil{
        return p,errors.New("existing/residual Incus state requires manual investigation")
    }else if !errors.Is(err,os.ErrNotExist){return p,err}
    if _,err=exec.LookPath("incus");err==nil{
        return p,errors.New("unmanaged Incus executable present: refuse automatic adoption")
    }
    out,err:=exec.Command("apt-get","-s","--no-install-recommends","install",
        "incus="+packageVersion,"incus-client="+packageVersion,"dnsmasq-base").CombinedOutput()
    if err!=nil{return p,fmt.Errorf("signed native apt simulation failed: %w",err)}
    simulation:=string(out)
    if strings.Contains(simulation,"Remv ") ||
        strings.Contains(simulation,"cannot be authenticated") ||
        strings.Contains(simulation,"essential packages will be removed"){
        return p,errors.New("unapproved removal or unauthenticated apt plan")
    }
    client,daemon:=false,false
    for _,line:=range strings.Split(simulation,"\n"){
        if strings.HasPrefix(line,"Inst incus-client ("+packageVersion+" "){client=true}
        if strings.HasPrefix(line,"Inst incus ("+packageVersion+" "){daemon=true}
        if strings.HasPrefix(line,"Inst "){p.AptChanges=append(p.AptChanges,strings.TrimSpace(line))}
    }
    if !client||!daemon{return p,errors.New("apt cannot provide exact signed native package version")}
    h:=sha256.Sum256([]byte("boxweave-s1-native-ubuntu2404\n"+packageVersion+"\n"+simulation))
    p.PlanSHA256=hex.EncodeToString(h[:])
    p.Mode="fresh-install-proposed"
    return p,nil
}
func NativeDependencyPlan(packageVersion string)(NativeBootstrapReport,error){
    return nativePlan(packageVersion)
}
func NativeDependencyBootstrap(version,approvedDigest string,approved bool)(NativeBootstrapReport,error){
    if os.Geteuid()!=0||!approved{return NativeBootstrapReport{},errors.New("explicit root local native-install approval required")}
    p,err:=nativePlan(version)
    if err!=nil{return p,err}
    if p.Mode=="reuse-existing-daemon"{return p,nil}
    if approvedDigest==""||approvedDigest!=p.PlanSHA256{
        return p,errors.New("approved apt plan SHA differs from current simulation")
    }
    if err:=runLocal("apt-get","install","-y","--no-install-recommends",
        "incus="+version,"incus-client="+version,"dnsmasq-base");err!=nil{
        return p,fmt.Errorf("partial apt install: stop, inspect, never blindly retry: %w",err)
    }
    if err:=runLocal("incus","admin","init","--auto","--storage-backend=dir");err!=nil{
        return p,fmt.Errorf("partial native initialization: stop, inspect, never blindly retry: %w",err)
    }
    if _,err=user.Lookup("boxweave");err!=nil{
        if err=runLocal("useradd","--create-home","--gid","incus",
            "--shell","/usr/sbin/nologin","boxweave");err!=nil{
            return p,fmt.Errorf("partial restricted identity bootstrap: %w",err)
        }
    }
    if err:=runLocal("runuser","-u","boxweave","--","incus","project","list","--format","csv");err!=nil{
        return p,fmt.Errorf("per-user restricted Project not ready: %w",err)
    }
    pkg,ok,err:=installedIncusPackage()
    if err!=nil||!ok||pkg!=version{
        return p,errors.New("exact native package postcondition was not verified")
    }
    if _,err=os.Lstat("/var/lib/incus/unix.socket.user");err!=nil{
        return p,errors.New("restricted native socket absent after bootstrap")
    }
    p.Mode="new-daemon-ready"
    return p,nil
}

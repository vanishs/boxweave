package install

import (
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
    "syscall"
    "time"
)

// Distinguish an active systemd state from a *stable and actually responding*
// instance of the exact reviewed executable. Do not accept a transient PID or
// a foreign process already bound to the loopback MCP port.
func observeInstalled(p Plan, m Manifest) error {
    if err := serviceActive(); err != nil {
        return fmt.Errorf("systemd unit inactive: %w", err)
    }
    out, err := exec.Command("systemctl","show","--property=MainPID","--value",serviceName).Output()
    if err != nil {return err}
    pid,err := strconv.Atoi(strings.TrimSpace(string(out)))
    if err != nil || pid < 2 {return errors.New("no stable service MainPID")}
    stat,err:=os.Stat(filepath.Join("/proc",strconv.Itoa(pid)))
    if err!=nil {return err}
    info,ok:=stat.Sys().(*syscall.Stat_t)
    if !ok {return errors.New("process identity unavailable")}
    uid,_,err:=ServiceIdentity(p)
    if err!=nil{return err}
    if int(info.Uid)!=uid {return errors.New("service MainPID belongs to wrong OS user")}
    h,err:=sha(filepath.Join("/proc",strconv.Itoa(pid),"exe"))
    if err!=nil{return err}
    if h!=m.BinarySHA256{return errors.New("running MainPID executable digest mismatch")}
    client:=&http.Client{Timeout:time.Second,Transport:&http.Transport{Proxy:nil}}
    req,err:=http.NewRequest(http.MethodPost,"http://127.0.0.1:18650/mcp",strings.NewReader("{}"))
    if err!=nil{return err}
    resp,err:=client.Do(req)
    if err!=nil{return err}
    _,_ = io.Copy(io.Discard,io.LimitReader(resp.Body,1024))
    _ = resp.Body.Close()
    if resp.StatusCode!=http.StatusUnauthorized{
        return fmt.Errorf("actual bound MCP unauthorized probe expected 401, got %d",resp.StatusCode)
    }
    return nil
}
func waitInstalledHealthy(p Plan,m Manifest) error {
    var last error
    streak:=0
    for i:=0;i<20;i++ {
        last=observeInstalled(p,m)
        if last==nil {
            streak++
            if streak>=3{return nil}
        }else{streak=0}
        time.Sleep(350*time.Millisecond)
    }
    if last==nil {last=errors.New("no stable runtime confirmation")}
    return fmt.Errorf("installed runtime not continuously healthy: %w",last)
}

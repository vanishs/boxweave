package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

type nativeExistingInventory struct {
	ZeroGuests bool
	OnlyDefaultProject bool
	HasReadyPool bool
	ServiceUserExists bool
}

func nativeExistingRead() (nativeExistingInventory,error){
	var v nativeExistingInventory
	if exec.Command("systemctl","is-active","--quiet","incus.service").Run()!=nil {
		return v,errors.New("installed native daemon inactive; refuse silent restart")
	}
	out,err:=exec.Command("incus","list","--all-projects","--format","json").CombinedOutput()
	if err!=nil{return v,fmt.Errorf("native guest inventory unavailable: %w",err)}
	var instances []json.RawMessage
	if err=json.Unmarshal(out,&instances);err!=nil{return v,err}
	v.ZeroGuests=len(instances)==0
	out,err=exec.Command("incus","project","list","--format","json").CombinedOutput()
	if err!=nil{return v,err}
	var projects []struct { Name string `json:"name"` }
	if err=json.Unmarshal(out,&projects);err!=nil{return v,err}
	v.OnlyDefaultProject=len(projects)==1&&projects[0].Name=="default"
	out,err=exec.Command("incus","storage","list","--format","json").CombinedOutput()
	if err!=nil{return v,err}
	var pools []struct {Name string `json:"name"`; Status string `json:"status"`}
	if err=json.Unmarshal(out,&pools);err!=nil{return v,err}
	for _,pool:=range pools{
		if pool.Name=="default" && strings.EqualFold(pool.Status,"Created"){v.HasReadyPool=true}
	}
	_,err=user.Lookup("boxweave")
	v.ServiceUserExists=err==nil
	if err!=nil {
		if _,ok:=err.(user.UnknownUserError);!ok{return v,err}
	}
	return v,nil
}

func nativeConfiguredProfile()error{
	out,err:=exec.Command("incus","profile","show","default").CombinedOutput()
	if err!=nil{return err}
	if !strings.Contains(string(out),"pool: default") || !strings.Contains(string(out),"path: /") {
		return errors.New("native default root disk/profile not configured")
	}
	return nil
}

// Verify an exact, project-confined non-root identity before treating an
// installed daemon as usable by BoxWeave. It cannot see foreign native users.
func nativeServiceActorReady()error{
	u,err:=user.Lookup("boxweave");if err!=nil{return err}
	uid,err:=strconv.Atoi(u.Uid);if err!=nil||uid==0{return errors.New("nonroot boxweave UID required")}
	g,err:=user.LookupGroup("incus")
	if err!=nil{return err}
	if u.Gid!=g.Gid{return errors.New("boxweave must have only primary incus group")}
	out,err:=exec.Command("runuser","-u","boxweave","--","id","-nG").CombinedOutput()
	if err!=nil||strings.TrimSpace(string(out))!="incus" {
		return errors.New("boxweave must have exactly the incus group")
	}
	own:="user-"+u.Uid
	out,err=exec.Command("runuser","-u","boxweave","--","incus","project","list","--format","csv").CombinedOutput()
	if err!=nil||strings.TrimSpace(string(out))==""||
		strings.Contains(strings.TrimSpace(string(out)),"\n")||
		!strings.HasPrefix(strings.TrimSpace(string(out)),own+" "){
		return errors.New("boxweave native project not exclusively confined")
	}
	out,err=exec.Command("incus","project","show",own).CombinedOutput()
	if err!=nil||!strings.Contains(string(out),"restricted: \"true\"") {
		return errors.New("service-user native project is not restricted")
	}
	return nil
}

func nativeProvisionDigest(version,mode string,v nativeExistingInventory)(string,error){
	host,err:=migrationHostDigest()
	if err!=nil{return "",err}
	h:=sha256.Sum256([]byte(fmt.Sprintf("s1-native-reuse-provision-v1|%s|%s|%s|%t|%t|%t|%t",
		host,version,mode,v.ZeroGuests,v.OnlyDefaultProject,v.HasReadyPool,v.ServiceUserExists)))
	return hex.EncodeToString(h[:]),nil
}

// Creates only the reviewed per-user confined Project; never restarts another
// daemon or grants 'incus-admin' to the runtime actor.
func nativePrepareServiceAccount()error{
	if _,err:=user.Lookup("boxweave");err!=nil{
		if _,ok:=err.(user.UnknownUserError);!ok{return err}
		if err=runLocal("useradd","--create-home","--gid","incus",
			"--shell","/usr/sbin/nologin","boxweave");err!=nil{return err}
	}
	out,err:=exec.Command("runuser","-u","boxweave","--","incus","project","list","--format","csv").CombinedOutput()
	if err!=nil{return fmt.Errorf("cannot establish restricted native user Project: %w (%s)",err,string(out))}
	return nativeServiceActorReady()
}

func nativeInstalledReview(version string,v nativeExistingInventory)(NativeBootstrapReport,error){
	p:=NativeBootstrapReport{PackageVersion:version}
	if !v.HasReadyPool {
		if !v.ZeroGuests || !v.OnlyDefaultProject || v.ServiceUserExists{
			return p,errors.New("partially initialized/shared native daemon: manual reconciliation required")
		}
		p.Mode="compatible-uninitialized-review"
		p.AptChanges=[]string{"LOCAL-ONLY: incus admin init --auto --storage-backend=dir","LOCAL-ONLY: create dedicated incus-only runtime user"}
	} else {
		if err:=nativeConfiguredProfile();err!=nil{return p,err}
		if v.ServiceUserExists {
			if err:=nativeServiceActorReady();err!=nil{return p,err}
			p.Mode="reuse-existing-daemon"
			return p,nil
		}
		p.Mode="compatible-user-provision-review"
		p.AptChanges=[]string{"LOCAL-ONLY: create dedicated incus-only runtime user; no native daemon restart"}
	}
	digest,err:=nativeProvisionDigest(version,p.Mode,v)
	if err!=nil{return p,err}
	p.PlanSHA256=digest
	return p,nil
}

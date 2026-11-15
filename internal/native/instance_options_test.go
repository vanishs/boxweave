package native

import (
 "encoding/json"
 "testing"
 "github.com/lxc/incus/v7/shared/api"
)

func boolPtr(v bool)*bool{return &v}
func strPtr(v string)*string{return &v}
func int64Ptr(v int64)*int64{return &v}

func TestBH019DottedOptionAdmissionFailsClosed(t *testing.T) {
 for _, tc:=range []struct{name string; options InstanceOptions}{
  {"unconfined", InstanceOptions{AppArmorProfile:strPtr("unconfined")}},
  {"unchanged needs trusted adapter", InstanceOptions{AppArmorProfile:strPtr("unchanged")}},
  {"generated needs trusted adapter", InstanceOptions{AppArmorProfile:strPtr("generated")}},
  {"invalid AppArmor", InstanceOptions{AppArmorProfile:strPtr("anything")}},
  {"low level allow nesting", InstanceOptions{AppArmorAllowNesting:boolPtr(true)}},
  {"low level allow nesting false", InstanceOptions{AppArmorAllowNesting:boolPtr(false)}},
  {"privileged", InstanceOptions{Privileged:boolPtr(true)}},
  {"explicit idmap base", InstanceOptions{IDMapBase:int64Ptr(65536)}},
  {"explicit idmap size", InstanceOptions{IDMapSize:int64Ptr(65536)}},
  {"unreserved isolated range", InstanceOptions{IDMapIsolated:boolPtr(true)}},
  {"device bpf without bpf", InstanceOptions{InterceptBPFDevices:boolPtr(true)}},
 } {
  t.Run(tc.name,func(t *testing.T) {
   if err:=tc.options.Apply(api.ConfigMap{},nil); err==nil {t.Fatal("unsafe option admitted")}
  })
 }
}
func TestBH019PresenceAndNoRawPassthrough(t *testing.T) {
 var o InstanceOptions
 if err:=json.Unmarshal([]byte(`{"security.protection.delete":false,"boot.autostart":true,"security.guestapi":false,"lxc.apparmor.profile":"unset","security.idmap.isolated":false}`),&o);err!=nil{t.Fatal(err)}
 if o.IsZero()||o.ProtectionDelete==nil||*o.ProtectionDelete||o.GuestAPI==nil||*o.GuestAPI||o.BootAutostart==nil||!*o.BootAutostart{t.Fatal("lost explicit false/true")}
 c:=api.ConfigMap{}
 if err:=o.Apply(c,nil);err!=nil{t.Fatal(err)}
 if c["security.protection.delete"]!="false"||c["boot.autostart"]!="true"||c["security.guestapi"]!="false"||c["security.idmap.isolated"]!="false"{t.Fatal(c)}
 if _,found:=c["raw.lxc"];found{t.Fatal("raw.lxc escaped")}
 if !o.Matches(observedNativeOptions(c)){t.Fatal("observed options did not match")}
 delete(c,"security.guestapi")
 if o.Matches(observedNativeOptions(c)){t.Fatal("missing option treated as explicit false")}
}
func TestBH019BPFDependenciesAndProjectGate(t *testing.T) {
 o:=InstanceOptions{InterceptBPF:boolPtr(true),InterceptBPFDevices:boolPtr(true)}
 c:=api.ConfigMap{}
 if err:=o.Apply(c,nil);err==nil{t.Fatal("interception admitted in unapproved Project")}
 if err:=o.Apply(c,map[string]string{"restricted.containers.interception":"allow"});err!=nil{t.Fatal(err)}
 if !o.Matches(observedNativeOptions(c)){t.Fatal("BPF readback lost")}
}

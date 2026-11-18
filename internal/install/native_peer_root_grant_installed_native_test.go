package install

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"reflect"
	"strings"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

func s4b3InstalledOneUse(t *testing.T, asRoot bool) {
	t.Helper()
	if os.Getenv("GITHUB_ACTIONS")!="true" ||
		os.Getenv("BOXWEAVE_S4B3_ONE_USE")!="20261118-disposable-root-grant" {
		t.Skip("only fresh independently provisioned Incus 7.6 GitHub host")
	}
	if (os.Geteuid()==0)!=asRoot {
		t.Fatal("B3 fixture must run at requested exact operating-system authority")
	}
}

func TestS4B3NativeRootSeedBaselineAndExactScope(t *testing.T) {
	s4b3InstalledOneUse(t,true)
	path:=os.Getenv("BOXWEAVE_S4B3_SCOPE")
	if path==""{t.Fatal("missing one-use fixture output")}
	ctx,cancel:=context.WithTimeout(context.Background(),120*time.Second)
	defer cancel()
	root,err:=incus.ConnectIncusUnixWithContext(ctx,"/var/lib/incus/unix.socket",nil)
	if err!=nil{t.Fatal(err)}
	u,err:=user.Lookup("boxweave")
	if err!=nil{t.Fatal(err)}
	scope:=s4b2InstalledScope{Project:"user-"+u.Uid,Bridge:"bwbr0",ACL:"bw-s4b2-native"}
	owned:=root.UseProject(scope.Project)
	for _,name:=range []string{"s4b2a","s4b2b","s4b2c"}{
		inst,_,err:=owned.GetInstance(name)
		if err!=nil || (inst.Status!="Running" && inst.Status!="Ready"){
			t.Fatalf("missing running NIC %s: %v",name,err)
		}
		dev:=inst.Devices["eth0"]
		mac:=strings.ToLower(inst.Config["volatile.eth0.hwaddr"])
		if mac==""{mac=strings.ToLower(inst.ExpandedDevices["eth0"]["hwaddr"])}
		if mac=="" || inst.Config["volatile.uuid"]=="" || dev["ipv4.address"]=="" || dev["ipv6.address"]==""{
			t.Fatal("missing pinned native UUID/MAC/IPv4/IPv6")
		}
		scope.Boxes=append(scope.Boxes,native.PeerACLBox{
			Name:name,UUID:inst.Config["volatile.uuid"],MAC:mac,
			IPv4:dev["ipv4.address"],IPv6:dev["ipv6.address"]})
	}
	nw,_,err:=root.GetNetwork(scope.Bridge)
	if err!=nil{t.Fatal(err)}
	got,tag,err:=root.GetNetworkACL(scope.ACL)
	if err!=nil || got==nil {t.Fatal("ACL baseline missing",err)}
	base,err:=NativePeerACLFromGrantSnapshot(NativePrepareTarget{
		Project:scope.Project,ACL:scope.ACL},nw,nil)
	if err!=nil{t.Fatal(err)}
	if err:=root.UpdateNetworkACL(scope.ACL,base.NetworkACLPut,tag);err!=nil{t.Fatal(err)}
	got,_,err=root.GetNetworkACL(scope.ACL)
	if err!=nil || !reflect.DeepEqual(got.Ingress,base.Ingress) ||
		!reflect.DeepEqual(got.Egress,base.Egress){
		t.Fatal("B3 native no-grant base ACL GET/PUT/readback is not exact",err)
	}
	payload,err:=json.Marshal(scope)
	if err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(path,payload,0644);err!=nil{t.Fatal(err)}
	t.Log("S4_B3_NATIVE_GENUINE_BASE_ACL_AND_THREE_OWNED_DUALSTACK_NICS=PASS")
}

func TestS4B3NativeNonrootSeedClaimedGrantOriginal(t *testing.T){
	s4b3InstalledOneUse(t,false)
	scopePath,dbPath:=os.Getenv("BOXWEAVE_S4B3_SCOPE"),os.Getenv("BOXWEAVE_S4B3_DB")
	if scopePath=="" || dbPath==""{t.Fatal("one-use fixture location unavailable")}
	raw,err:=os.ReadFile(scopePath)
	if err!=nil{t.Fatal(err)}
	var scope s4b2InstalledScope
	if err:=json.Unmarshal(raw,&scope);err!=nil{t.Fatal(err)}
	u,err:=user.Lookup("boxweave")
	if err!=nil || u.Uid!=fmt.Sprint(os.Geteuid()) || scope.Project!="user-"+u.Uid ||
		scope.Bridge!="bwbr0" || scope.ACL!="bw-s4b2-native" || len(scope.Boxes)!=3{
		t.Fatal("wrong Project/service uid or incomplete native scope")
	}
	s,err:=state.Open(dbPath)
	if err!=nil{t.Fatal(err)}
	defer s.Close()
	db,err:=sql.Open("sqlite3",dbPath)
	if err!=nil{t.Fatal(err)}
	defer db.Close()
	ctx:=context.Background()
	now:=time.Now().UnixNano()
	for _,b:=range scope.Boxes{
		if _,err:=db.ExecContext(ctx,`INSERT INTO owned_boxes
 (uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,'OWNED',?)`,
			b.UUID,scope.Project,b.Name,b.UUID,now);err!=nil{t.Fatal(err)}
		if _,err:=db.ExecContext(ctx,`INSERT INTO network_bindings
 (project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at)
 VALUES(?,?,?,?,?,?,?,?,?,'PINNED',?)`,
			scope.Project,b.UUID,b.UUID,"eth0",scope.Bridge,scope.ACL,b.MAC,b.IPv4,b.IPv6,now);err!=nil{t.Fatal(err)}
		for _,ip:=range []struct{family,addr string}{{"ipv4",b.IPv4},{"ipv6",b.IPv6}}{
			if _,err:=db.ExecContext(ctx,`INSERT INTO network_address_fences
 (bridge,family,address,project,box_uuid,incarnation,state,observed_at)
 VALUES(?,?,?,?,?,?,'RESERVED',?)`,
				scope.Bridge,ip.family,ip.addr,scope.Project,b.UUID,b.UUID,now);err!=nil{t.Fatal(err)}
		}
	}
	_,tok,err:=s.IssueToken(ctx,"admin",time.Hour)
	if err!=nil{t.Fatal(err)}
	a,b:=scope.Boxes[0],scope.Boxes[1]
	intent:=state.RootPeerRevokeSafeIntent{
		Project:scope.Project,Bridge:scope.Bridge,ACL:scope.ACL,
		SourceUUID:a.UUID,SourceIncarnation:a.UUID,
		TargetUUID:b.UUID,TargetIncarnation:b.UUID,Allow:true}
	safe,err:=json.Marshal(intent)
	if err!=nil{t.Fatal(err)}
	op,err:=s.Admit(ctx,state.Admission{
		Layer:scope.Project,ActorID:tok.ID,TokenID:tok.ID,
		Key:"s4b3-native-installed-grant-1118-unique",RequestID:"s4b3-one-use-root-grant",
		Action:"box_peer_set",BoxUUID:a.UUID,Incarnation:a.UUID,SafeIntent:safe,
		ResourceKey:"acl:default/"+scope.ACL,ResourceKeys:[]string{"box:"+a.UUID,"box:"+b.UUID}})
	if err!=nil{t.Fatal(err)}
	if err:=s.StagePeerGrant(ctx,op.ID);err!=nil{t.Fatal(err)}
	if err:=s.Claim(ctx,op.ID);err!=nil{t.Fatal(err)}
	fmt.Printf("S4_B3_NATIVE_GRANT_ORIGINAL=%s\n",op.ID)
	t.Log("S4_B3_NATIVE_NONROOT_ADMIN_CLAIM_AND_PENDING_GRANT=PASS")
}

func TestS4B3NativeInstalledRootGrantActualEffect(t *testing.T){
	s4b3InstalledOneUse(t,true)
	scopePath,dbPath:=os.Getenv("BOXWEAVE_S4B3_SCOPE"),os.Getenv("BOXWEAVE_S4B3_DB")
	original,scenario:=os.Getenv("BOXWEAVE_S4B3_ORIGINAL"),os.Getenv("BOXWEAVE_S4B3_SCENARIO")
	if scopePath=="" || dbPath=="" || len(original)!=32 ||
		(scenario!="normal" && scenario!="after_acl_fault"){
		t.Fatal("missing exact disposable native grant original or scenario")
	}
	raw,err:=os.ReadFile(scopePath)
	if err!=nil{t.Fatal(err)}
	var scope s4b2InstalledScope
	if err:=json.Unmarshal(raw,&scope);err!=nil || len(scope.Boxes)!=3{t.Fatal("native scope invalid",err)}
	s,err:=state.OpenNetworkAuditReadOnly(dbPath)
	if err!=nil{t.Fatal(err)}
	defer s.Close()
	ctx,cancel:=context.WithTimeout(context.Background(),180*time.Second)
	defer cancel()
	plan,err:=NativePeerRootGrantReviewOperation(ctx,s,scope.Project,scope.Bridge,scope.ACL,original)
	if err!=nil || len(plan.PlanSHA256)!=64{
		t.Fatalf("native root review did not seal claimed original: %+v %v",plan,err)
	}
	var receipt NativePeerConntrackJournalReceipt
	if scenario=="after_acl_fault"{
		faults:=0
		receipt,err=nativePeerRootGrantApplyWithTestFault(ctx,s,scope.Project,scope.Bridge,scope.ACL,
			original,plan.PlanSHA256,true,func()error{faults++;return errors.New("injected after REAL ACL PUT")})
		if err==nil || faults!=1 || receipt.Status!="DISPATCH_UNKNOWN" || receipt.MutationComplete{
			t.Fatalf("uncertain real ACL PUT was reported as granted: %+v faults=%d err=%v",receipt,faults,err)
		}
		journalPath,err:=nativePeerConntrackOperationJournalPath(original)
		if err!=nil{t.Fatal(err)}
		journal,err:=nativePeerConntrackLoadJournal(journalPath)
		if err!=nil || journal.Status!="DISPATCH_UNKNOWN" || journal.Plan.PlanSHA256!=plan.PlanSHA256{
			t.Fatalf("root original is not frozen UNKNOWN: %+v %v",journal,err)
		}
		root,err:=incus.ConnectIncusUnixWithContext(ctx,"/var/lib/incus/unix.socket",nil)
		if err!=nil{t.Fatal(err)}
		for _,b:=range scope.Boxes{
			inst,_,err:=root.UseProject(scope.Project).GetInstance(b.Name)
			if err!=nil || inst.Status!="Stopped" || inst.Config["boot.autostart"]!="false"{
				t.Fatalf("post-ACL uncertainty released a native sibling: %s %v",b.Name,err)
			}
		}
		if err:=nativePeerBridgeKernelNoLivePorts(ctx,scope.Bridge);err!=nil{t.Fatal(err)}
		t.Log("S4_B3_REAL_ACL_PUT_UNKNOWN_ROOT_JOURNAL_ALL_OWNED_NICS_STOPPED=PASS")
		return
	}
	receipt,err=NativePeerRootGrantApplyOperation(ctx,s,scope.Project,scope.Bridge,scope.ACL,
		original,plan.PlanSHA256,true)
	if err!=nil || receipt.Status!="COMPLETE_VERIFIED" || !receipt.KernelVerified ||
		receipt.VerifiedNICs!=3 || receipt.ProbedScopes!=4 || receipt.ReadyForPeer ||
		receipt.PostNativeDigestSHA256==""{
		t.Fatalf("native grant kernel/all-NIC completion not verified: %+v %v",receipt,err)
	}
	if _,err:=NativePeerRootGrantApplyOperation(ctx,s,scope.Project,scope.Bridge,scope.ACL,
		original,plan.PlanSHA256,true);err==nil{
		t.Fatal("same root grant original was redispatched")
	}
	t.Log("S4_B3_REAL_ROOT_GRANT_EXACT_ACL_NFT_ALL_NICS_AND_NO_REPLAY=PASS")
}

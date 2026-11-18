package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// NativePeerRootGrantApplyOperation is the B3 grant branch of the B2 root
// original journal. A failed or interrupted native effect remains UNKNOWN;
// it never returns a speculative grant or exposes a general ACL/root API.
func NativePeerRootGrantApplyOperation(ctx context.Context,s *state.Store,
	project,bridge,acl,original,approvedSHA string,approved bool)(NativePeerConntrackJournalReceipt,error){
	return nativePeerRootGrantApplyWithTestFault(ctx,s,project,bridge,acl,original,approvedSHA,approved,nil)
}

// Fault seam is package-private and is never available from IPC/production.
// It lets native tests prove that a real completed ETag PUT followed by an
// uncertain response retains stopped NICs and the DISPATCH_UNKNOWN journal.
func nativePeerRootGrantApplyWithTestFault(ctx context.Context,s *state.Store,
	project,bridge,acl,original,approvedSHA string,approved bool,afterACL func()error)(NativePeerConntrackJournalReceipt,error){
	var zero NativePeerConntrackJournalReceipt
	if !approved || os.Geteuid()!=0 || s==nil || !s.IsNetworkAuditReadOnly(){
		return zero,errors.New("grant root requires operator approval and read-only state")
	}
	path,err:=nativePeerConntrackOperationJournalPath(original)
	if err!=nil{return zero,err}
	unlock,err:=nativeScriptletAcquireLock(nativePeerConntrackLockPath)
	if err!=nil{return zero,err}
	defer unlock()
	pre,err:=nativePeerRootGrantReview(ctx,s,project,bridge,acl,original)
	if err!=nil{return zero,err}
	if approvedSHA=="" || approvedSHA!=pre.Plan.PlanSHA256 {
		return zero,errors.New("grant root operator approval pre-isolation digest mismatch")
	}
	if err:=nativePeerConntrackPrepareOriginalJournalDirectory();err!=nil{return zero,err}
	return nativePeerConntrackDispatchOnce(ctx,path,pre.Plan,approvedSHA,approved,
		func(ctx context.Context)(NativePeerConntrackReport,error){
			root,err:=incus.ConnectIncusUnixWithContext(ctx,"/var/lib/incus/unix.socket",nil)
			if err!=nil{return NativePeerConntrackReport{},err}
			check:=func(ctx context.Context)error{
				if _,err:=nativePeerBridgeAuditFromStoreWithClient(ctx,s,root,project,bridge,acl);err!=nil{return err}
				global,err:=nativePeerGlobalNetworkScopeSnapshot(ctx,root,pre.Target)
				if err!=nil || global!=pre.GlobalSHA{return errors.New("global grant native scope drift")}
				routes,err:=nativePeerHostRoutes(ctx,pre.Target)
				if err!=nil || routes!=pre.RoutesSHA{return errors.New("grant host routes changed")}
				proof,err:=s.ReadRootPeerGrant(ctx,original,project,bridge,acl)
				if err!=nil || !reflect.DeepEqual(proof,pre.Proof){
					return errors.New("grant owner/claims/whole allowlist changed during execution")
				}
				return nil
			}
			if err:=native.IsolateSharedPeerACLWithClient(ctx,root,project,bridge,acl,pre.Target.Boxes,check);err!=nil {
				return NativePeerConntrackReport{},err
			}
			if err:=nativePeerConntrackAllStopped(ctx,root,pre.Target);err!=nil{return NativePeerConntrackReport{},err}
			if err:=nativePeerBridgeKernelNoLivePorts(ctx,bridge);err!=nil{return NativePeerConntrackReport{},err}
			current,etag,err:=root.GetNetworkACL(acl)
			if err!=nil || current==nil || etag!=pre.ACLTag ||
				!reflect.DeepEqual(current.Ingress,pre.Before.Ingress) ||
				!reflect.DeepEqual(current.Egress,pre.Before.Egress){
				return NativePeerConntrackReport{},errors.New("grant ACL baseline drifted across stop barrier")
			}
			nw,_,err:=root.GetNetwork(bridge)
			if err!=nil{return NativePeerConntrackReport{},err}
			want,err:=NativePeerACLFromGrantSnapshot(NativePrepareTarget{Project:project,ACL:acl},nw,pre.Proof.Target)
			if err!=nil || !reflect.DeepEqual(want,pre.After){
				return NativePeerConntrackReport{},errors.New("bridge config or grant compiler target drifted")
			}
			if err:=check(ctx);err!=nil{return NativePeerConntrackReport{},err}
			if err:=root.UpdateNetworkACL(acl,pre.After.NetworkACLPut,etag);err!=nil {
				return NativePeerConntrackReport{},fmt.Errorf("native grant ETag fanout uncertain: %w",err)
			}
			updated,_,err:=root.GetNetworkACL(acl)
			if err!=nil || updated==nil || !reflect.DeepEqual(updated.Ingress,pre.After.Ingress) ||
				!reflect.DeepEqual(updated.Egress,pre.After.Egress){
				return NativePeerConntrackReport{},errors.New("native grant ACL whole-policy readback mismatch")
			}
			if err:=check(ctx);err!=nil{return NativePeerConntrackReport{},err}
			if afterACL!=nil {
				if err:=afterACL();err!=nil{return NativePeerConntrackReport{},fmt.Errorf("post-grant ACL result uncertain: %w",err)}
			}
			// The B2 revoke conntrack helper deliberately requires a DROP.
			// Instead purge only four original pinned tuples while the entire
			// protected bridge has no live ports. This must never be global -F.
			report,err:=nativePeerConntrackPurgeExact(ctx,nativePeerConntrackOS,pre.Source,pre.Destination)
			report.NativeDigestSHA256=pre.Plan.NativeDigestSHA256
			if err!=nil{return report,err}
			if err:=nativePeerConntrackAllStopped(ctx,root,pre.Target);err!=nil{return report,err}
			if err:=nativePeerBridgeKernelNoLivePorts(ctx,bridge);err!=nil{return report,err}
			post,err:=nativePeerBridgeAuditFromStoreWithClient(ctx,s,root,project,bridge,acl)
			if err!=nil{return report,err}
			report.PostNativeDigestSHA256=post.NativeDigestSHA256
			if err:=check(ctx);err!=nil{return report,err}
			count,err:=nativePeerRootResumeGrantVerified(ctx,root,pre)
			if err!=nil{return report,err}
			report.VerifiedNICs=count
			report.KernelVerified=count>=2
			if !report.KernelVerified{return report,errors.New("grant native per-NIC kernel evidence incomplete")}
			if err:=check(ctx);err!=nil{return report,err}
			return report,nil
		})
}

// A Start SDK failure can mean that a guest actually resumed. On ANY
// uncertainty, attempt to STOP every formerly protected NIC, including the
// member whose start was ambiguous. Root journal and SQLite remain UNKNOWN.
func nativePeerRootResumeGrantVerified(ctx context.Context,root incus.InstanceServer,
	pre nativePeerGrantPreflight)(int,error){
	owned:=root.UseProject(pre.Target.Project)
	var resumed []string
	success:=false
	defer func(){
		if success{return}
		stopCtx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
		defer cancel()
		for _,box:=range pre.Target.Boxes {
			op,e:=owned.UpdateInstanceState(box.Name,api.InstanceStatePut{Action:"stop",Timeout:20},"")
			if e==nil && op!=nil{_ = op.WaitContext(stopCtx)}
		}
	}()
	for _,name:=range pre.Running {
		if err:=ctx.Err();err!=nil{return 0,err}
		op,err:=owned.UpdateInstanceState(name,api.InstanceStatePut{Action:"start",Timeout:30},"")
		if err!=nil || op==nil{return 0,fmt.Errorf("native grant Start uncertain for %q: %w",name,err)}
		resumed=append(resumed,name)
		if err:=op.WaitContext(ctx);err!=nil{return 0,err}
		var box native.PeerACLBox
		for _,b:=range pre.Target.Boxes {if b.Name==name{box=b;break}}
		inst,_,err:=owned.GetInstance(name)
		if err!=nil{return 0,err}
		if err:=native.ValidatePeerACLBox(inst,pre.Target.Project,pre.Target.Bridge,pre.Target.ACL,box,false);err!=nil{return 0,err}
		if (inst.Status!="Running" && inst.Status!="Ready") || inst.Config["boot.autostart"]!="false"{
			return 0,errors.New("native grant NIC resume or autostart ownership drift")
		}
		rules,err:=nativePeerRootKernelRules(ctx)
		if err!=nil{return 0,err}
		if err:=nativePeerRootVerifyGrantNFT(rules,pre.Target.Project,pre.Target.Boxes,resumed,pre.Proof.Target);err!=nil{return 0,err}
	}
	success=true
	return len(resumed),nil
}

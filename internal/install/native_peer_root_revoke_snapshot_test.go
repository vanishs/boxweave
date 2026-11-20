package install

import (
	"reflect"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func TestS4B3RevokeSnapshotCompilerPreservesUnrelatedAllowAndExactDrops(t *testing.T) {
	tgt := NativePrepareTarget{ACL: "bw-test-drop", Project: "bwowned"}
	n := revocationTestNetwork()
	revoked := revocationTestPair("uuid-b", "10.248.83.12", "fd83:248::12")
	remaining := []state.PeerGrantBinding{{
		SourceUUID: "uuid-a", SourceIncarnation: "uuid-a",
		TargetUUID: "uuid-c", TargetIncarnation: "uuid-c",
		SourceIPv4: "10.248.83.11", TargetIPv4: "10.248.83.13",
		SourceIPv6: "fd83:248::11", TargetIPv6: "fd83:248::13",
	}}
	got, err := nativeACLRevokeOneFromSnapshot(tgt, n, remaining, revoked)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := nativeACLWithDeniedPeers(tgt, n,
		[]nativePeerBinding{revocationTestPair("uuid-c", "10.248.83.13", "fd83:248::13")},
		[]nativePeerBinding{revoked})
	if err != nil || !reflect.DeepEqual(got, direct) {
		t.Fatal("whole-policy preserve/deny compiler disagreement", err)
	}
	reverse := []state.PeerGrantBinding{{
		SourceUUID: "uuid-b", SourceIncarnation: "uuid-b",
		TargetUUID: "uuid-a", TargetIncarnation: "uuid-a",
		SourceIPv4: "10.248.83.12", TargetIPv4: "10.248.83.11",
		SourceIPv6: "fd83:248::12", TargetIPv6: "fd83:248::11",
	}}
	if _, err := nativeACLRevokeOneFromSnapshot(tgt, n, reverse, revoked); err == nil {
		t.Fatal("revocation silently dropped an independent reverse stateful grant")
	}
	remaining[0].TargetIPv6 = "fd83:248::99"
	remaining[0].TargetIPv4 = "127.0.0.1"
	if _, err := nativeACLRevokeOneFromSnapshot(tgt, n, remaining, revoked); err == nil {
		t.Fatal("untrusted remaining address admitted")
	}
	t.Log("S4_B3_REVOKE_FULL_ACL_OTHER_PEER_PRESERVED_REVERSE_REFUSED=PASS")
}

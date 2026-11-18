package state

import (
	"context"
	"os"
	"testing"
)

func s4b3RootGrantFixture(t *testing.T) (*Store, *Store, string) {
	t.Helper()
	s, id := s4b3GrantFixture(t)
	if err := s.StagePeerGrant(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	var seq int
	var name, path string
	if err := s.db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func(){ _ = r.Close() })
	return s, r, id
}

func TestS4B3RootGrantOnlyExactClaimedOriginal(t *testing.T) {
	s, r, id := s4b3RootGrantFixture(t)
	ctx := context.Background()
	proof := func() (RootPeerGrant,error) {
		return r.ReadRootPeerGrant(ctx,id,"user-1001","bwbr0","bw-s4b2")
	}
	if _, err := proof(); err == nil {
		t.Fatal("unclaimed grant was root-authorized")
	}
	if err := s.Claim(ctx,id); err != nil { t.Fatal(err) }
	g,err := proof()
	if err != nil || g.OperationID != id || !g.Intent.Allow || len(g.Confirmed)!=0 ||
		len(g.Target)!=1 || g.Target[0].SourceUUID!=s4b2RootSrc {
		t.Fatalf("claimed grant denied: %+v %v",g,err)
	}
	if _,err:=r.ReadRootPeerRevocation(ctx,id,"user-1001","bwbr0","bw-s4b2");err==nil {
		t.Fatal("B3 grant passed B2 revoke-only proof")
	}
	t.Log("S4_B3_ROOT_GRANT_CLAIMED_ORIGINAL_NOT_REVOKE=PASS")
}

func TestS4B3RootGrantRejectsPoststageDrift(t *testing.T) {
	cases:=[]struct{name,sql string}{
		{"unknown","UPDATE operations SET state='UNKNOWN'"},
		{"wrong-action","UPDATE operations SET action='box_start'"},
		{"wrong-layer","UPDATE operations SET layer='foreign'"},
		{"wrong-safe-intent","UPDATE operations SET safe_intent='{}'"},
		{"viewer","UPDATE credentials SET role='viewer'"},
		{"revoked","UPDATE credentials SET revoked=1"},
		{"missing-claim","DELETE FROM operation_resource_claims WHERE resource_key LIKE 'acl:%'"},
		{"foreign-target","UPDATE peer_grants SET target_uuid='foreign'"},
		{"stale-lease","UPDATE network_address_fences SET state='RELEASED' WHERE address='fd43:18::12'"},
		{"lost-sibling","DELETE FROM network_bindings WHERE box_uuid='"+s4b2RootThird+"'"},
		{"bad-grant","UPDATE peer_grants SET desired=0"},
		{"unknown-grant","UPDATE peer_grants SET status='UNKNOWN'"},
	}
	for _,tc:=range cases {t.Run(tc.name,func(t *testing.T){
		s,r,id:=s4b3RootGrantFixture(t)
		if err:=s.Claim(context.Background(),id);err!=nil{t.Fatal(err)}
		if _,err:=s.db.Exec(tc.sql);err!=nil{t.Fatal(err)}
		if got,err:=r.ReadRootPeerGrant(context.Background(),id,"user-1001","bwbr0","bw-s4b2");err==nil {
			t.Fatalf("drift accepted: %+v",got)
		}
	})}
	t.Log("S4_B3_ROOT_GRANT_UNKNOWN_CLAIMS_FOREIGN_FENCES_FAIL_CLOSED=PASS")
}

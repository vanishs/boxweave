package state

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/peerscope"
)

func TestPR178N3LifecycleWithTerminalPeerPolicy(t *testing.T) {
	ctx := context.Background()
	s, _, _ := n3RetiredFixture(t)
	const project, bridge = "user-1001", "incusbr-1001"
	a, b := "11111111-1111-4111-8111-000000000002", "11111111-1111-4111-8111-000000000003"
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	peer := func(allow bool, key string) {
		t.Helper()
		i := ScopedPeerOriginalIntent{Schema: 2, Project: project, Bridge: bridge, SourceUUID: a, SourceIncarnation: a, TargetUUID: b, TargetIncarnation: b, Allow: allow}
		raw, _ := json.Marshal(i)
		claims, e := scopedPeerResourceClaims(i)
		if e != nil {
			t.Fatal(e)
		}
		rec, e := s.AdmitScoped(ctx, Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: key, RequestID: key, Action: "box_peer_set", SafeIntent: raw, BoxUUID: a, Incarnation: a, ResourceKey: "peer-guard:" + bridge, ResourceKeys: claims[:4]})
		if e != nil {
			t.Fatal(e)
		}
		e = s.StageAndClaimScopedPeerOriginal(ctx, rec.ID)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.ReadScopedTerminalPeerPolicy(ctx, project, bridge); e == nil {
			t.Fatal("pending grant admitted as terminal")
		}
		tx, e := s.db.BeginTx(ctx, nil)
		if e != nil {
			t.Fatal(e)
		}
		p, e := readScopedPeerOriginalTx(ctx, tx, rec.ID, project, bridge, "DISPATCHING")
		tx.Rollback()
		if e != nil {
			t.Fatal(e)
		}
		// Test-only synthetic Root receipt; no claim of native verification.
		e = s.FinalizeScopedPeerOriginal(ctx, ScopedPeerRootReceipt{OriginalID: rec.ID, Project: project, Bridge: bridge, Status: "COMPLETE_VERIFIED", ProofSHA256: ScopedPeerProofSHA256(p), PlanSHA256: strings.Repeat("a", 64), FinalSHA256: strings.Repeat("b", 64), VerifiedSteps: 4})
		if e != nil {
			t.Fatal(e)
		}
	}
	peer(true, "n3-terminal-lifecycle-grant")
	pol, e := s.ReadScopedTerminalPeerPolicy(ctx, project, bridge)
	if e != nil || len(pol.Active) != 1 || len(pol.Revoked) != 0 {
		t.Fatal(pol, e)
	}
	name, uuid := "n3-extra", "11111111-1111-4111-8111-000000000004"
	raw, _ := json.Marshal(ScopedCreateIntent{Schema: 2, Project: project, Bridge: bridge, Name: name, CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Options: json.RawMessage("{}")})
	rec, e := s.AdmitScoped(ctx, Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "n3-terminal-extra-create", RequestID: "extra-create", Action: "box_create", ResourceKey: "box:" + name, SafeIntent: raw})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, rec.ID); e != nil {
		t.Fatal(e)
	}
	pin := VerifiedCreateBinding{bridge, peerscope.QuarantineACL, "02:16:3e:11:22:04", "10.43.18.14", "fd43:18::14"}
	if ok, e := s.StageCreateBinding(ctx, rec.ID, project, name, pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = s.NativeAck(ctx, rec.ID, "native-extra-create"); e != nil {
		t.Fatal(e)
	}
	pin.ACL, e = peerscope.ACLName(project, bridge, uuid)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedScopedCreate(ctx, rec.ID, project, name, uuid, pin); e != nil {
		t.Fatal(e)
	}
	del := func(id, key string) {
		t.Helper()
		box, e := s.OwnedBox(ctx, project, id)
		if e != nil {
			t.Fatal(e)
		}
		raw, _ := json.Marshal(struct {
			BoxID string `json:"box_id"`
		}{id})
		rec, e := s.AdmitScoped(ctx, Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: key, RequestID: key, Action: "box_delete", CleanupDependencies: true, ResourceKey: "box:" + box.Name, BoxUUID: id, Incarnation: box.Incarnation, SafeIntent: raw})
		if e != nil {
			t.Fatal(e)
		}
		if e = s.Claim(ctx, rec.ID); e != nil {
			t.Fatal(e)
		}
		if e = s.NativeAck(ctx, rec.ID, "native-"+key); e != nil {
			t.Fatal(e)
		}
		if e = s.FinalizeVerifiedScopedDelete(ctx, rec.ID, box); e != nil {
			t.Fatal(e)
		}
	}
	del(uuid, "n3-terminal-extra-delete")
	peer(false, "n3-terminal-lifecycle-revoke")
	del(b, "n3-terminal-revoked-delete")
	pol, e = s.ReadScopedTerminalPeerPolicy(ctx, project, bridge)
	if e != nil || len(pol.Active) != 0 || len(pol.Revoked) != 1 || pol.Revoked[0].TargetUUID != b {
		t.Fatal(pol, e)
	}
}

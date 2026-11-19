package state

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestS4B2NativeNetworkDependenciesBlockAllRecyclingLifecycle(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, tok, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	const project = "user-1001"
	grantSQL := "INSERT INTO peer_grants(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status) VALUES (?,?,?,?,?,1,'UNKNOWN')"
	if _, err := s.db.ExecContext(ctx, grantSQL, project, "uuid-a", "uuid-a", "uuid-b", "uuid-b"); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"box_delete", "box_update", "box_stop", "box_start"} {
		for _, who := range []string{"uuid-a", "uuid-b"} {
			a := Admission{
				Layer: project, ActorID: tok.ID, TokenID: tok.ID,
				Key:       fmt.Sprintf("s4b2-guard-%s-%s-1116", action, who),
				RequestID: "lifecycle-no-native", Action: action, SafeIntent: []byte("{}"),
				ResourceKey: "box:" + who, BoxUUID: who, Incarnation: who,
			}
			if _, err := s.Admit(ctx, a); !errors.Is(err, ErrResourceLocked) {
				t.Fatalf("unsafe %s of %s with UNKNOWN peer admitted: %v", action, who, err)
			}
		}
	}
	// One unrelated Box must continue independently.
	other := Admission{
		Layer: project, ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b2-independent-native-lifecycle-1116", RequestID: "independent",
		Action: "box_stop", SafeIntent: []byte("{}"),
		ResourceKey: "box:uuid-c", BoxUUID: "uuid-c", Incarnation: "uuid-c",
	}
	if _, err := s.Admit(ctx, other); err != nil {
		t.Fatalf("foreign independent Box was locked: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE peer_grants SET status='REVOKED' WHERE project=?", project); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO forward_claims(forward_id,project,box_uuid,incarnation,native_device,protocol,host_bind,host_port,guest_port,guest_ip,state) VALUES(?,?,?,?,?,'tcp','127.0.0.1',8089,8080,'10.0.0.3','UNKNOWN')",
		"fwd-1", project, "uuid-b", "uuid-b", "proxy-1"); err != nil {
		t.Fatal(err)
	}
	b := Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b2-unknown-native-forward-1116", RequestID: "forward",
		Action: "box_delete", SafeIntent: []byte("{}"), ResourceKey: "box:uuid-b",
		BoxUUID: "uuid-b", Incarnation: "uuid-b",
	}
	if _, err := s.Admit(ctx, b); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("UNKNOWN forward failed to fence Box deletion: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE forward_claims SET state='REVOKED' WHERE forward_id='fwd-1'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES(?,?,?, ?,?,?,'QUARANTINED',?)",
		"bw-bridge", "ipv4", "10.81.0.3", project, "uuid-b", "uuid-b", time.Now().UnixNano()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Admit(ctx, b); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("QUARANTINED lease failed to fence Box deletion: %v", err)
	}
	// This SQL mutation represents independent reviewed/native proof in a
	// *test fixture*, NOT an exposed product release path.
	if _, err := s.db.ExecContext(ctx,
		"UPDATE network_address_fences SET state='RELEASED' WHERE bridge='bw-bridge'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Admit(ctx, b); err != nil {
		t.Fatalf("fully cleared dependencies permanently blocked delete: %v", err)
	}
}

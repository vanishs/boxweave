//go:build ignore

// The one-use CI probe reads a REAL original ACK-lost native Create.
// In an isolated SQLite COPY, legacy identity-only binding MUST fail closed;
// exact staged pin, native leases and the original marker MUST allow the
// modern read-only-native recovery path to settle atomically.
// The original controller ledger and native instance are NEVER modified.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/recovery"
	"github.com/vanishs/boxweave/internal/state"
)

type originalCheckpoint struct {
	OriginalID string `json:"original_id"`
}

func check(ok bool, msg string) {
	if !ok {
		panic("S2_UNKNOWN_CREATE_NATIVE_BIND_FAIL: " + msg)
	}
}

func main() {
	check(len(os.Args) == 5, "expected exact checkpoint / copied database / Project / native UUID")
	checkpoint, copiedDB, project, uuid := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	check(os.Geteuid() != 0 && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		strings.HasPrefix(checkpoint, "/tmp/bw-v1-s2-ghci.") &&
		strings.HasSuffix(checkpoint, "/ack-original.json") &&
		strings.HasPrefix(copiedDB, "/tmp/bw-v1-s2-ghci.") &&
		filepath.Base(copiedDB) == "ledger.sqlite" &&
		filepath.Base(filepath.Dir(copiedDB)) == "unknown-create-bind-copy" &&
		strings.HasPrefix(project, "user-") && len(uuid) == 36,
		"not a one-use isolated restricted CI native identity")
	data, err := os.ReadFile(checkpoint)
	check(err == nil, "checkpoint missing")
	var original originalCheckpoint
	check(json.Unmarshal(data, &original) == nil && len(original.OriginalID) == 32,
		"no durable original Operation ID")

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	nativeReader := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	name := "bw-s2-ack-native-110d"
	observed, err := nativeReader.ObserveCreatedByOperation(ctx, name, original.OriginalID)
	check(err == nil && observed.UUID == uuid && observed.Name == name && observed.Project == project,
		"native marker/UUID/Project not bound to original")
	_, err = nativeReader.ObserveCreatedByOperation(ctx, name, "00000000000000000000000000000000")
	check(errors.Is(err, native.ErrNotOwned), "foreign original marker accepted")

	nativeRead, err := nativeReader.ObserveOwnedBox(ctx, name, uuid)
	check(err == nil && nativeRead.State == "PRESENT_UUID_CONFIRMED" && nativeRead.Fingerprint != "",
		"second exact native UUID readback missing")

	ledger, err := state.Open(copiedDB)
	check(err == nil, "cannot open isolated SQLite backup")
	defer ledger.Close()
	snap, err := ledger.CreateRecoveryInspect(ctx, original.OriginalID)
	check(err == nil && snap.NativeOperationID == "" && snap.Name == name && snap.Project == project,
		"original lost ACK no longer unbound and immutable")
	challenge, err := ledger.IssueCreateBindingChallenge(ctx, snap, "ci-independent-verifier", nativeRead.Fingerprint)
	check(err == nil, "reviewer challenge not issued")
	err = ledger.BindUnknownCreatedBox(ctx, snap, "ci-independent-verifier", challenge, uuid,
		"fake-changed-native-observation", "reviewed exact native UUID and unchanged original", true)
	check(err != nil, "forged native readback accepted")
	err = ledger.BindUnknownCreatedBox(ctx, snap, "ci-independent-verifier", challenge, uuid,
		nativeRead.Fingerprint, "reviewed exact native UUID and unchanged original", true)
	check(errors.Is(err, state.ErrResourceLocked), "modern pinned Create accepted legacy identity-only binding")
	again, err := ledger.Get(ctx, original.OriginalID)
	check(err == nil && again.State == "UNKNOWN" && again.NativeOperationID == "" &&
		again.BoxUUID == "" && again.NextAction.Kind == "HOST_LOCAL_REVIEW",
		"legacy binding refusal altered original UNKNOWN or its lock")
	fmt.Println("S2_NATIVE_CREATE_REAL_LOST_ACK_LEGACY_IDENTITY_BIND_REFUSED=PASS")

	db, err := sql.Open("sqlite3", "file:"+copiedDB+"?mode=ro&_foreign_keys=on")
	check(err == nil, "copy read-only inspection unavailable")
	defer db.Close()
	var actor, tokenID string
	var intent []byte
	err = db.QueryRowContext(ctx, `SELECT actor_id,token_id,safe_intent FROM operations WHERE id=?`, original.OriginalID).Scan(
		&actor, &tokenID, &intent)
	check(err == nil && actor == tokenID, "original owner credential unknown")
	var pins int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_resource_claims
		WHERE operation_id=? AND resource_key LIKE 'create-pin:%'`, original.OriginalID).Scan(&pins)
	check(err == nil && pins == 4, "original modern Create is missing one or more address reservations")

	db, err := sql.Open("sqlite3", "file:"+copiedDB+"?mode=ro&_foreign_keys=on")
	check(err == nil, "copy read-only inspection unavailable")
	defer db.Close()
	var actor, tokenID string
	var intent []byte
	err = db.QueryRowContext(ctx, `SELECT actor_id,token_id,safe_intent FROM operations WHERE id=?`, original.OriginalID).Scan(
		&actor, &tokenID, &intent)
	check(err == nil && actor == tokenID, "original owner credential unknown")
	var pins int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_resource_claims
		WHERE operation_id=? AND resource_key LIKE 'create-pin:%'`, original.OriginalID).Scan(&pins)
	check(err == nil && pins == 4, "original modern Create is missing one or more address reservations")

	// Exercise the actual N3 staged recovery path, not a fixture-only UUID
	// insert. Native operations are only OBSERVED; all writes go to the copy.
	svc := recovery.Service{Store: ledger, Native: nativeReader, Layer: project}
	diagnosis, err := svc.Inspect(ctx, original.OriginalID, "")
	check(err == nil && diagnosis.View.Disposition == "SAFE_RECOVER" &&
		diagnosis.View.ProposedAction == state.RecoverySettle &&
		diagnosis.View.Reason == "CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED" &&
		diagnosis.View.ObservedNativeUUID == uuid &&
		diagnosis.View.ClaimProtection == "RETAINED",
		"exact native marker, address pin, lease or poststate not proved")
	request := state.RecoveryRequest{
		Layer: project, ActorID: actor, TokenID: tokenID,
		OriginalID: original.OriginalID, Key: "s2-native-staged-ack-loss-reconcile-copy-001",
		WorkRef: "lab/lost-native-ack", AgentRef: "ack-verifier",
	}
	settled, err := svc.Reconcile(ctx, request)
	check(err == nil && settled.Recovery != nil &&
		settled.Recovery.Disposition == "SETTLED" &&
		settled.Recovery.Receipt.State == "SUCCEEDED" && !settled.NativeRetried,
		"staged native Create recovery failed to settle copied journal atomically")
	again, err = ledger.Get(ctx, original.OriginalID)
	check(err == nil && again.State == "SUCCEEDED" && again.BoxUUID == uuid &&
		again.NativeOperationID == "", "original receipt not settled with exact UUID and no invented native ACK")
	var owned, bindings, fences int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM owned_boxes WHERE uuid=? AND project=? AND name=? AND state='OWNED'`,
		uuid, project, name).Scan(&owned)
	check(err == nil && owned == 1, "staged recovery did not atomically establish ownership")
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_bindings WHERE project=? AND box_uuid=? AND state='PINNED'`,
		project, uuid).Scan(&bindings)
	check(err == nil && bindings == 1, "staged recovery omitted the complete original static binding")
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_address_fences
		WHERE project=? AND box_uuid=? AND state='RESERVED' AND operation_id=?`,
		project, uuid, original.OriginalID).Scan(&fences)
	check(err == nil && fences == 2, "staged recovery omitted either original address fence")
	repeated, err := svc.Reconcile(ctx, request)
	check(err == nil && repeated.Recovery != nil &&
		repeated.Recovery.Receipt.Duplicate &&
		repeated.Recovery.Receipt.ID == settled.Recovery.Receipt.ID && !repeated.NativeRetried,
		"recovery idempotency re-executed the original effect")
	fmt.Println("S2_NATIVE_CREATE_REAL_LOST_ACK_PINNED_SQLITE_COPY_RECOVERY_NO_RETRY=PASS")
	same := state.Admission{
		Layer: project, ActorID: actor, TokenID: tokenID,
		Key:    "native-s2-create-ack-dropped-once-001",
		Action: "box_create", ResourceKey: "box:" + name,
		SafeIntent: intent, WorkRef: "lab/lost-native-ack", AgentRef: "ack-verifier",
		RequestID: "ci-original-replay",
	}
	replay, err := ledger.Replay(ctx, same)
	check(err == nil && replay.Duplicate && replay.ID == original.OriginalID &&
		replay.State == "SUCCEEDED" && replay.BoxUUID == uuid, "same key no longer preserved as settled original")
	same.Key = "different-key-native-create-must-not-dispatch"
	_, err = ledger.Admit(ctx, same)
	check(errors.Is(err, state.ErrResourceLocked), "existing recovered Box admitted duplicate native Create")
	fmt.Println("S2_NATIVE_CREATE_REAL_LOST_ACK_RECOVERED_ORIGINAL_REPLAY_AND_LOCK=PASS")
}

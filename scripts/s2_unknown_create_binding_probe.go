//go:build ignore

// The one-use CI probe reads a REAL original ACK-lost native Create and
// proves that a COPY of its UNKNOWN journal can bind the exact native UUID.
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

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
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
	check(err == nil, "exact original UNKNOWN identity binding refused")
	again, err := ledger.Get(ctx, original.OriginalID)
	check(err == nil && again.State == "UNKNOWN" && again.NativeOperationID == "" &&
		again.BoxUUID == uuid && again.NextAction.Kind == "HOST_LOCAL_REVIEW",
		"original UNKNOWN falsely treated as success or native ACK invented")
	fmt.Println("S2_NATIVE_CREATE_REAL_LOST_ACK_MARKER_UUID_COPIED_SQLITE_BIND_NO_RETRY=PASS")

	err = ledger.BindUnknownCreatedBox(ctx, snap, "ci-independent-verifier", challenge, uuid,
		nativeRead.Fingerprint, "repeated original challenge must be refused", true)
	check(err != nil, "replayed original binding accepted")
	db, err := sql.Open("sqlite3", "file:"+copiedDB+"?mode=ro&_foreign_keys=on")
	check(err == nil, "copy read-only inspection unavailable")
	defer db.Close()
	var actor, tokenID string
	var intent []byte
	err = db.QueryRowContext(ctx, `SELECT actor_id,token_id,safe_intent FROM operations WHERE id=?`, original.OriginalID).Scan(
		&actor, &tokenID, &intent)
	check(err == nil && actor == tokenID, "original owner credential unknown")
	same := state.Admission{
		Layer: project, ActorID: actor, TokenID: tokenID,
		Key:    "native-s2-create-ack-dropped-once-001",
		Action: "box_create", ResourceKey: "box:" + name,
		SafeIntent: intent, WorkRef: "lab/lost-native-ack", AgentRef: "ack-verifier",
		RequestID: "ci-original-replay",
	}
	replay, err := ledger.Replay(ctx, same)
	check(err == nil && replay.Duplicate && replay.ID == original.OriginalID &&
		replay.State == "UNKNOWN", "same key no longer preserved as original")
	same.Key = "different-key-native-create-must-not-dispatch"
	_, err = ledger.Admit(ctx, same)
	check(errors.Is(err, state.ErrResourceLocked), "locked UNKNOWN admitted duplicate native Create")
	fmt.Println("S2_NATIVE_CREATE_REAL_LOST_ACK_BIND_ORIGINAL_REPLAY_AND_LOCK=PASS")
}

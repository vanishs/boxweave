//go:build ignore

// Test-only, exact-actor SQLite WAL fsync crash witness. It is not a BoxWeave
// command or a second native transport. The native effect has already been
// acknowledged as potentially accepted; it MUST never be dispatched here.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vanishs/boxweave/internal/state"
)

const fixtureNativeID = "a0200000-0000-4000-8000-00000000f51c"

func require(cond bool, msg string) {
	if !cond {
		panic("S2_FSYNC_FAIL: " + msg)
	}
}

func admission(project, actorID string) state.Admission {
	return state.Admission{
		Layer: project, ActorID: actorID, TokenID: actorID,
		Key:         "s2-wal-committing-native-ack-immutable-key-01",
		Action:      "box_update",
		SafeIntent:  []byte("{\"fixture\":\"original-native-already-dispatched\"}"),
		ResourceKey: "s2-fsync-lab:original-native-dispatched",
		RequestID:   "s2-fsync-original-request",
		WorkRef:     "lab/sqlite-fsync", AgentRef: "one-use-fault",
	}
}

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "prepare" && os.Args[1] != "ack" && os.Args[1] != "verify") {
		panic("explicit test-only mode required")
	}
	db, token, project := os.Getenv("BW_S2_FSYNC_DB"), os.Getenv("BW_S2_FSYNC_TOKEN"), os.Getenv("BW_S2_FSYNC_PROJECT")
	marker := os.Getenv("BW_S2_FSYNC_ARM_FILE")
	if os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" || os.Geteuid() == 0 ||
		!strings.HasPrefix(db, "/tmp/bw-v1-s2-ghci.") ||
		!strings.HasSuffix(db, "/fsync-ledger/ledger.sqlite") ||
		filepath.Clean(db) != db ||
		!strings.HasPrefix(token, "bw1_") || !strings.HasPrefix(project, "user-") ||
		!strings.HasPrefix(marker, filepath.Dir(db)+"/") {
		panic("only nonroot disposable GH WAL sync lab permitted")
	}
	ctx := context.Background()
	s, err := state.Open(db)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	principal, err := s.VerifyToken(ctx, token)
	if err != nil || principal.Role != "admin" {
		panic("one-use exact admin token required")
	}
	a := admission(project, principal.ID)
	switch os.Args[1] {
	case "prepare":
		r, err := s.Admit(ctx, a)
		if err != nil || r.State != "ADMITTED" || r.NativeOperationID != "" {
			panic(fmt.Sprintf("initial original admission unexpectedly native: %v", err))
		}
		if err = s.Claim(ctx, r.ID); err != nil {
			panic(err)
		}
		claimed, err := s.Get(ctx, r.ID)
		if err != nil || claimed.State != "DISPATCHING" || claimed.NativeOperationID != "" {
			panic("original not persistently claimed for simulated already-accepted native")
		}
		fmt.Println("S2_FSYNC_ORIGINAL_OPERATION_ID=" + r.ID)
	case "ack":
		r, err := s.Replay(ctx, a)
		require(err == nil && r.State == "DISPATCHING" &&
			r.NativeOperationID == "", "incorrect original before NativeAck COMMIT")
		if err := os.WriteFile(marker, []byte("armed-for-original-native-ack\n"), 0600); err != nil {
			panic(err)
		}
		// The LD_PRELOAD test-only hook kills THIS process at a real
		// SQLite WAL fsync/fdatasync during this exact transaction commit.
		// An untriggered native acknowledgment is a deliberate CI FAILURE.
		err = s.NativeAck(ctx, r.ID, fixtureNativeID)
		panic(fmt.Sprintf("WAL fsync kill was not injected; NativeAck err=%v", err))
	case "verify":
		before, err := s.Replay(ctx, a)
		require(err == nil && (before.State == "DISPATCHING" || before.State == "VERIFYING"),
			"post-crash original was neither intact pre-COMMIT nor intact post-COMMIT")
		require(before.NativeOperationID == "" || before.NativeOperationID == fixtureNativeID,
			"unknown native identity introduced by fsync crash")
		if before.State == "VERIFYING" {
			require(before.NativeOperationID == fixtureNativeID, "committed native ACK lost its original ID")
		}
		n, err := s.Recover(ctx)
		require(err == nil && n == 1, "recovery did not lock exactly one in-flight original")
		after, err := s.Replay(ctx, a)
		require(err == nil && after.Duplicate && after.State == "UNKNOWN" &&
			after.ID == before.ID && after.TrackingID == before.TrackingID &&
			after.NativeOperationID == before.NativeOperationID &&
			after.NextAction.Kind == "HOST_LOCAL_REVIEW",
			"torn commit recovery changed original or permitted re-dispatch")
		if n, err := s.Recover(ctx); err != nil || n != 0 {
			panic("second recovery changed the already locked original")
		}
		fmt.Println("S2_SQLITE_WAL_FSYNC_REAL_KILL_UNKNOWN_NO_REPLAY=PASS")
	}
}

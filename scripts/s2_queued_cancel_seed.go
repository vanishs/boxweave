//go:build ignore

// A deterministic one-use journal admission only, not a shipped CLI or
// native mutation. The actual cancellation is performed by the installed
// candidate through authenticated official MCP SDK in s2_native_probe.go.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vanishs/boxweave/internal/state"
)

func main() {
	db := os.Getenv("BW_S2_SEED_DB")
	project := os.Getenv("BW_S2_SEED_PROJECT")
	token := os.Getenv("BW_S2_SEED_ADMIN_TOKEN")
	if !strings.HasPrefix(db, "/tmp/bw-v1-s2-ghci.") ||
		!strings.HasSuffix(db, "/ledger/ledger.sqlite") ||
		!strings.HasPrefix(project, "user-") || !strings.HasPrefix(token, "bw1_") ||
		os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" || os.Geteuid() == 0 {
		panic("queued cancel seed limited to disposable nonroot GH Incus lab")
	}
	ctx := context.Background()
	store, err := state.Open(db)
	if err != nil {
		panic(err)
	}
	defer store.Close()
	actor, err := store.VerifyToken(ctx, token)
	if err != nil || actor.Role != "admin" {
		panic("one-use seed requires original authenticated administrator")
	}
	r, err := store.Admit(ctx, state.Admission{
		Layer: project, ActorID: actor.ID, TokenID: actor.ID,
		Key:         "native-s2-queued-undispatched-cancel-original-01",
		Action:      "box_exec",
		SafeIntent:  []byte("{\"fixture\":\"admitted-not-dispatched\"}"),
		ResourceKey: "s2-ci:queued-undispatched-cancel",
		RequestID:   "s2-ci-queued-cancel-seed",
		WorkRef:     "lab/queued-cancel", AgentRef: "native-seed",
	})
	if err != nil || r.State != "ADMITTED" || r.NativeOperationID != "" {
		panic(fmt.Sprintf("queued native-free fixture admission not safe: %v", err))
	}

	// Entirely controller-local history, never an Incus native effect. These
	// records belong to the same disposable authenticated admin actor but do
	// not claim/lock an owned Box. Exercise real SDK indexed pagination,
	// actor/label filters and bounds with >2 full 50-row pages.
	for i := 0; i < 120; i++ {
		item, err := store.Admit(ctx, state.Admission{
			Layer: project, ActorID: actor.ID, TokenID: actor.ID,
			Key: fmt.Sprintf("native-s2-history-indexed-seed-%03d", i),
			Action: "box_exec",
			SafeIntent: []byte(fmt.Sprintf("{\"fixture\":\"history-only\",\"i\":%d}", i)),
			RequestID: fmt.Sprintf("s2-native-history-%03d", i),
			WorkRef: "lab/history-120", AgentRef: "seed-only",
		})
		if err != nil || item.State != "ADMITTED" || item.NativeOperationID != "" {
			panic(fmt.Sprintf("refusing fake/native history fixture %03d: %v", i, err))
		}
		if err := store.CancelQueued(ctx, project, item.ID); err != nil {
			panic(fmt.Sprintf("history fixture cancellation %03d: %v", i, err))
		}
	}
	// Keep stdout one exact original queued ID for the caller's existing
	// safe cancellation witness; history totals are verified over real MCP.
	fmt.Println(r.ID)
}

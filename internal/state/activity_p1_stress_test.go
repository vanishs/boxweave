package state

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// Explicit opt-in: run only in disposable Box, not every normal CI job.
func TestActivityP1History100kOptIn(t *testing.T) {
	if os.Getenv("BOXWEAVE_ACTIVITY_P1_STRESS") != "1" {
		t.Skip("100k history requires dedicated isolated disposable environment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	s := newStore(t)
	tok := activityTestToken(t, s)
	seedStarted := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { t.Fatal(err) }
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO operations(id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,safe_intent,work_ref,agent_ref,resource_key,box_uuid,incarnation,state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil { t.Fatal(err) }
	startNS := time.Now().Add(-350 * 24 * time.Hour).UnixNano()
	const records = 100000
	for i := 0; i < records; i++ {
		id := fmt.Sprintf("%032x", i+1)
		layer := "user-1001"
		if i%2 == 0 { layer = "foreign-layer" }
		state := "SUCCEEDED"
		if i%333 == 1 { state = "UNKNOWN" }
		_, err = stmt.ExecContext(ctx, id, id, id, layer, tok.ID, tok.ID,
			"key-"+id, "box_exec", id, []byte("{}"),
			"p1/stress-20261119", "p1/long-history", "box:synthetic",
			"synthetic-box", "synthetic-gen", state, startNS+int64(i), startNS+int64(i))
		if err != nil { t.Fatalf("seed record %d: %v", i, err) }
	}
	if err := stmt.Close(); err != nil { t.Fatal(err) }
	if err := tx.Commit(); err != nil { t.Fatal(err) }
	t.Logf("P1_HISTORY_SEED_100000_MS=%d", time.Since(seedStarted).Milliseconds())

	firstIssues := time.Now()
	issuesCount := 0
	issueSeen := map[string]bool{}
	cursor := ""
	pages := 0
	for {
		got, err := s.ReadActivity(ctx, ActivityQuery{
			Layer: "user-1001", Requester: tok.ID, View: "issues",
			WorkRef: "p1/stress-20261119", Limit: 50, Cursor: cursor,
		})
		if err != nil { t.Fatalf("issues after page %d: %v", pages, err) }
		for _, issue := range got.Issues {
			if issue.Reason != "UNRESOLVED_OPERATION" || issue.State != "UNKNOWN" {
				t.Fatalf("wrong SQLite unresolved issue: %+v", issue)
			}
			if issueSeen[issue.OperationID] { t.Fatalf("duplicate issue in stable 100k history: %s", issue.OperationID) }
			issueSeen[issue.OperationID] = true
			issuesCount++
		}
		pages++
		if !got.Truncated {
			if got.NextCursor != "" { t.Fatal("nontruncated issues result has misleading cursor") }
			break
		}
		if got.NextCursor == "" || pages > 1000 || got.AttentionComplete {
			t.Fatalf("incorrect limited issue page: %+v", got)
		}
		cursor = got.NextCursor
	}
	if issuesCount < 100 { t.Fatalf("failed to recover old UNKNOWN originals: only %d", issuesCount) }
	t.Logf("P1_ISSUES_100K_COUNT=%d PAGES=%d TOTAL_MS=%d", issuesCount, pages, time.Since(firstIssues).Milliseconds())

	overviewAt := time.Now()
	cursor = ""
	overviewSeen := map[string]bool{}
	for page := 0; page < 40; page++ {
		got, err := s.ReadActivity(ctx, ActivityQuery{
			Layer: "user-1001", Requester: tok.ID, View: "overview",
			Limit: 50, Cursor: cursor,
		})
		if err != nil || len(got.Operations) != 50 {
			t.Fatalf("overview page=%d result=%+v err=%v", page, got, err)
		}
		for _, op := range got.Operations {
			if overviewSeen[op.ID] { t.Fatalf("overview cursor duplicated %s", op.ID) }
			overviewSeen[op.ID] = true
		}
		if !got.Truncated || got.NextCursor == "" { t.Fatalf("history page %d ended early", page) }
		cursor = got.NextCursor
	}
	if len(overviewSeen) != 2000 { t.Fatal("2000-result history walk count drift") }
	t.Logf("P1_OVERVIEW_100K_2000_RESULTS_TOTAL_MS=%d", time.Since(overviewAt).Milliseconds())

	if _, err := s.ReadActivity(ctx, ActivityQuery{
		Layer: "foreign-layer", Requester: tok.ID, View: "overview",
		Limit: 50, Cursor: cursor,
	}); err != ErrInvalid { t.Fatalf("foreign Layer cursor replay: %v", err) }
	_, other, err := s.IssueToken(ctx, time.Hour)
	if err != nil { t.Fatal(err) }
	if _, err := s.ReadActivity(ctx, ActivityQuery{
		Layer: "user-1001", Requester: other.ID, View: "overview",
		Limit: 50, Cursor: cursor,
	}); err != ErrInvalid { t.Fatalf("different Token cursor replay: %v", err) }

	concurrentAt := time.Now()
	const workers, calls = 8, 8
	results := make(chan error, workers*calls)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < calls; j++ {
				view := "overview"
				if (n+j)%2 == 0 { view = "issues" }
				got, err := s.ReadActivity(ctx, ActivityQuery{
					Layer: "user-1001", Requester: tok.ID, View: view, Limit: 50,
				})
				if err == nil && got.Scope != "user-1001" {
					err = fmt.Errorf("scope drift: %s", got.Scope)
				}
				results <- err
			}
		}(w)
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil { t.Fatalf("concurrent SQLite query: %v", err) }
	}
	t.Logf("P1_CONCURRENT_100K_READS=%d TOTAL_MS=%d", workers*calls, time.Since(concurrentAt).Milliseconds())
}

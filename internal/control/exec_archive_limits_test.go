package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

func TestExecArchiveCapacityRefusesCaptureWithoutLosingExitIdentity(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	st, err := state.Open(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := &Service{Store: st}
	root, err := execArchiveDirectory(st, true)
	if err != nil {
		t.Fatal(err)
	}
	// Sparse files count toward the *logical* archive budget: no actual
	// 256 MiB write or host-disk pressure is needed for this negative test.
	for i := 1; i <= int(execArchiveTotalLimit/execArchiveStreamLimit); i++ {
		path := filepath.Join(root, fmt.Sprintf("%032x.stdout", i))
		f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e = f.Truncate(execArchiveStreamLimit); e != nil {
			t.Fatal(e)
		}
		if e = f.Close(); e != nil {
			t.Fatal(e)
		}
	}
	id := strings.Repeat("f", 32)
	called := false
	err = svc.captureExecStream(context.Background(), id, "stdout", "trusted-native",
		func(context.Context, string, int64, int64) ([]byte, bool, error) {
			called = true
			return []byte("not-forced"), true, nil
		})
	if err == nil || called {
		t.Fatalf("aggregate archive cap not enforced: err=%v native_read=%v", err, called)
	}
	if _, _, _, found, readErr := svc.readExecArchive(id, "stdout", 0, 32); readErr != nil || found {
		t.Fatalf("spool cap created an unverified stream: found=%v err=%v", found, readErr)
	}
}

func TestExecArchiveMissingDeletedBoxIsStructuredAndForeignScopeDenied(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	st, err := state.Open(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	raw, tok, err := st.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ownedBox := strings.Repeat("b", 32)
	foreignBox := strings.Repeat("c", 32)
	admit := func(layer, box, key string) state.Receipt {
		r, e := st.Admit(ctx, state.Admission{Layer: layer, ActorID: tok.ActorID, TokenID: tok.ID,
			Key: key, RequestID: key, Action: "box_exec", SafeIntent: []byte("{}"),
			BoxUUID: box, Incarnation: box, ResourceKey: "box:" + box})
		if e != nil {
			t.Fatal(e)
		}
		exit := 23
		p := "/1.0/instances/removed/logs/exec-output/exec_11111111-2222-3333-4444-555555555555.stdout"
		if e = st.SaveExecOutput(ctx, r.ID, p, "", &exit); e != nil {
			t.Fatal(e)
		}
		return r
	}
	owned := admit("bw-test", ownedBox, "missing-archive-owned-1")
	foreign := admit("foreign-project", foreignBox, "missing-archive-foreign-1")
	server := activityMCPServer(st)
	defer server.Close()
	client, err := activityMCPClient(ctx, server.URL,
		&http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: raw}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	call := func(id string) (*mcp.CallToolResult, error) {
		return client.CallTool(ctx, &mcp.CallToolParams{Name: "operation_output_get",
			Arguments: map[string]any{"operation_id": id, "stream": "stdout"}})
	}
	reply, err := call(owned.ID)
	if err != nil || reply == nil || reply.IsError {
		t.Fatalf("missing archive was a misleading MCP error: %+v %v", reply, err)
	}
	data, err := json.Marshal(reply.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var got operationOutputResult
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Availability != "unavailable" || got.UnavailableReason != "BOX_REMOVED_OUTPUT_NOT_ARCHIVED" ||
		got.ExitCode == nil || *got.ExitCode != 23 || got.CommandSuccess == nil || *got.CommandSuccess ||
		got.ContentBase64 != "" {
		t.Fatalf("deleted original was not honestly classified: %+v", got)
	}
	refused, err := call(foreign.ID)
	if err == nil && refused != nil && !refused.IsError {
		t.Fatalf("foreign layer original leaked stdout/exit: %+v", refused)
	}
}

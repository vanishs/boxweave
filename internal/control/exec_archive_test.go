package control

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

func TestExecArchiveSurvivesDeletedBoxAndExpiresOverRealMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "store.sqlite")
	st, err := state.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	raw, token, err := st.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	boxID := "b2c122c0-a9f4-4a2f-a63e-3982247e359e"
	rec, err := st.AdmitScoped(ctx, state.Admission{
		Layer: "bw-test", ActorID: token.ActorID, TokenID: token.ID,
		Key: "archive-output-live-mcp-001", RequestID: "archive-output-request",
		Action: "box_exec", SafeIntent: []byte("{}"),
		BoxUUID: boxID, Incarnation: boxID,
		ResourceKey: "box:already-removed", WorkRef: "archive-tests",
	})
	if err != nil {
		t.Fatal(err)
	}
	// There is deliberately NO owned Box: a delete/tombstone must not
	// make the authorized original Exec output impossible to recover.
	nativePath := "/1.0/instances/removed/logs/exec-output/exec_11111111-2222-3333-4444-555555555555.stdout"
	exit := 0
	if err := st.SaveExecOutput(ctx, rec.ID, nativePath, "", &exit); err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: st}
	content := []byte("retained-result-after-box-delete\n")
	reader := func(_ context.Context, _ string, offset, limit int64) ([]byte, bool, error) {
		if offset >= int64(len(content)) {
			return nil, true, nil
		}
		end := offset + limit
		if end > int64(len(content)) {
			end = int64(len(content))
		}
		return content[offset:end], end == int64(len(content)), nil
	}
	if err := svc.captureExecStream(ctx, rec.ID, "stdout", nativePath, reader); err != nil {
		t.Fatal(err)
	}
	server := activityMCPServer(st)
	defer server.Close()
	client, err := activityMCPClient(ctx, server.URL,
		&http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: raw}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	read := func() operationOutputResult {
		t.Helper()
		reply, err := client.CallTool(ctx, &mcp.CallToolParams{Name: "operation_output_get",
			Arguments: map[string]any{"operation_id": rec.ID, "stream": "stdout"}})
		if err != nil || reply == nil || reply.IsError {
			t.Fatalf("archive MCP read: %+v %v", reply, err)
		}
		raw, err := json.Marshal(reply.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		var value operationOutputResult
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	got := read()
	data, err := base64.StdEncoding.DecodeString(got.ContentBase64)
	if err != nil || !bytes.Equal(data, content) || got.Availability != "available" ||
		got.ExitCode == nil || *got.ExitCode != 0 ||
		got.CommandSuccess == nil || !*got.CommandSuccess || !got.Ended {
		t.Fatalf("deleted Box archive bytes/exit missing: %+v bytes=%q %v", got, data, err)
	}
	database, err := sql.Open("sqlite3", "file:"+dbPath+"?_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	old := time.Now().Add(-8 * 24 * time.Hour)
	if _, err = database.ExecContext(ctx, "UPDATE exec_outputs SET observed_at=? WHERE operation_id=?", old.UnixNano(), rec.ID); err != nil {
		t.Fatal(err)
	}
	expired := read()
	if expired.Availability != "expired" || expired.UnavailableReason != "RETENTION_WINDOW_ELAPSED" ||
		expired.ContentBase64 != "" || expired.ExitCode == nil {
		t.Fatalf("expired archive leaked retained bytes or lost exit: %+v", expired)
	}
	n, err := svc.SweepExecArchive(ctx, time.Now())
	if err != nil || n != 1 {
		t.Fatalf("7-day GC did not remove archived file: %d %v", n, err)
	}
	if _, _, _, found, err := svc.readExecArchive(rec.ID, "stdout", 0, 100); err != nil || found {
		t.Fatalf("GC archive still readable: found=%v err=%v", found, err)
	}
}

func TestExecArchiveCapsOutputAndRejectsSymlink(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	st, err := state.Open(filepath.Join(dir, "store.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := &Service{Store: st}
	id := strings.Repeat("a", 32)
	payload := bytes.Repeat([]byte("B"), int(execArchiveStreamLimit)+17)
	fake := func(_ context.Context, _ string, offset, limit int64) ([]byte, bool, error) {
		end := offset + limit
		if end > int64(len(payload)) {
			end = int64(len(payload))
		}
		if offset >= int64(len(payload)) {
			return []byte{}, true, nil
		}
		return payload[offset:end], end == int64(len(payload)), nil
	}
	if err := svc.captureExecStream(ctx, id, "stderr", "native-path-only-in-test", fake); err != nil {
		t.Fatal(err)
	}
	out, ended, truncated, found, err := svc.readExecArchive(id, "stderr", execArchiveStreamLimit-8, 64)
	if err != nil || !found || !ended || !truncated || !bytes.Equal(out, bytes.Repeat([]byte("B"), 8)) {
		t.Fatalf("bounded output not safely truncated: n=%d ended=%v trunc=%v found=%v err=%v", len(out), ended, truncated, found, err)
	}
	root, err := execArchiveDirectory(st, false)
	if err != nil {
		t.Fatal(err)
	}
	evilID := strings.Repeat("b", 32)
	target, err := execArchiveFile(root, evilID, "stdout")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "store.sqlite"), target); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, err = svc.readExecArchive(evilID, "stdout", 0, 30)
	if err == nil {
		t.Fatal("archive read followed untrusted symlink")
	}
	if err := svc.captureExecStream(ctx, evilID, "stdout", "native-path-only-in-test", fake); err == nil {
		t.Fatal("archive capture used untrusted symlink")
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatal("symlink confusion")
	}
}

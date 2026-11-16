//go:build ignore

// One-use, supervised S3 installed HTTP MCP File Tool black-box probe.
// It uses only the public official MCP client; native Guest poststate is
// independently checked by the disposable outer runner, not this process.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fileBearerTransport struct{ token string }

func (t fileBearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copyReq := req.Clone(req.Context())
	copyReq.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(copyReq)
}

func need(ok bool, reason string) {
	if !ok {
		panic("S3_FILE_INSTALLED_FAIL: " + reason)
	}
}

func requiredEnv(name string) string {
	s := os.Getenv(name)
	need(s != "", "missing isolated setup "+name)
	return s
}

func token(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "token=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "token="))
		}
	}
	panic("CI-local bearer file has no token field")
}

func connect(ctx context.Context, name, tokenPath string) *mcp.ClientSession {
	client := mcp.NewClient(&mcp.Implementation{Name: "s3-file-phase2-" + name, Version: "1"}, nil)
	s, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: requiredEnv("BW_S3_MCP_URL"),
		HTTPClient: &http.Client{Transport: fileBearerTransport{token: token(tokenPath)}},
		DisableStandaloneSSE: true,
		MaxRetries: -1,
	}, nil)
	if err != nil {
		panic(fmt.Errorf("isolated MCP %s connect: %w", name, err))
	}
	return s
}

func call(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) map[string]any {
	result, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil || result == nil || result.IsError {
		panic(fmt.Errorf("original MCP tool %s failed: %v; no retry", name, err))
	}
	b, err := json.Marshal(result.StructuredContent)
	if err != nil {
		panic(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}

func rejects(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) bool {
	result, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	return err != nil || (result != nil && result.IsError)
}

func field(o map[string]any, name string) string {
	s, _ := o[name].(string)
	return s
}

func object(o map[string]any, name string) map[string]any {
	v, _ := o[name].(map[string]any)
	need(v != nil, "missing object "+name)
	return v
}

func originalID(o map[string]any) string {
	s := field(o, "operation_id")
	need(len(s) == 32, "original durable operation ID absent")
	return s
}

func await(ctx context.Context, s *mcp.ClientSession, id, want string) map[string]any {
	for i := 0; i < 80; i++ {
		r := object(call(ctx, s, "operation_get", map[string]any{
			"operation_id": id, "wait_ms": 3500,
		}), "operation")
		state := field(r, "state")
		if state == want {
			return r
		}
		if state == "FAILED" || state == "UNKNOWN" || state == "CANCELLED" ||
			state == "SUCCEEDED" {
			panic(fmt.Sprintf("original %s finished %s but expected %s (no native retry)", id, state, want))
		}
	}
	panic("original did not complete in bounded wait, preserve original no redispatch")
}

func fileBytes(ctx context.Context, s *mcp.ClientSession, uuid, file string) []byte {
	v := call(ctx, s, "box_file_read", map[string]any{
		"box_id": uuid, "path": file, "offset": 0, "length": 256,
	})
	need(v["whole_file"] == true && v["ended"] == true, "bounded full file read incomplete")
	b, err := base64.StdEncoding.DecodeString(field(v, "data_base64"))
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(b)
	need(field(v, "page_sha256") == hex.EncodeToString(sum[:]), "page digest differs from read bytes")
	return b
}

func main() {
	if os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("BW_S3_FILE_DISPOSABLE") != "approved-ephemeral-vm" ||
		os.Geteuid() == 0 {
		panic("refuse non-CI or privileged installed MCP test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 16*time.Minute)
	defer cancel()
	admin := connect(ctx, "admin", requiredEnv("BW_S3_ADMIN_TOKEN_FILE"))
	defer admin.Close()
	viewer := connect(ctx, "viewer", requiredEnv("BW_S3_VIEWER_TOKEN_FILE"))
	defer viewer.Close()
	t, err := admin.ListTools(ctx, nil)
	if err != nil {
		panic(err)
	}
	names := map[string]bool{}
	for _, tool := range t.Tools {
		names[tool.Name] = true
	}
	for _, n := range []string{"box_file_list", "box_file_read", "box_file_apply"} {
		need(names[n], "missing File Tool "+n)
	}
	need(len(names) == 16, "installed catalog is not exactly 16 after S3 phase1")
	fmt.Println("S3_INSTALLED_16_TOOL_OFFICIAL_SDK_CATALOG=PASS")
	// One and only one new disposable owned Box. No shared user Boxes,
	// L0-L5 historical resources, or parent Incus credential delegation.
	const name = "s3-file-phase2-ci"
	create := map[string]any{
		"name": name, "cpu": 1, "memory_mib": 1024, "disk_gib": 2,
		"start": true, "internet": false, "idempotency_key": "s3-file-installed-box-create-1115",
	}
	created := call(ctx, admin, "box_create", create)
	createID := originalID(created)
	done := await(ctx, admin, createID, "SUCCEEDED")
	uuid := field(done, "box_uuid")
	need(len(uuid) >= 32, "new Box native UUID missing")
	state := object(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	need(field(state, "box_id") == uuid && field(state, "native_state") == "Running",
		"native owned Guest is not running")
	fmt.Println("S3_INSTALLED_BOX_CREATE_ORIGINAL_UUID=PASS")
	base := "/tmp/s3-file-phase2"
	fixture := "mkdir -p " + base + "; " +
		"printf S3_PHASE2_VICTIM_LOCKED > " + base + "/victim; " +
		"ln -s " + base + "/victim " + base + "/sym; " +
		"ln " + base + "/victim " + base + "/hard; " +
		"mkfifo " + base + "/fifo; " +
		"ln -s " + base + " /tmp/s3-file-phase2-link; echo S3_FIXTURE_READY"
	e := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-ec", fixture},
		"idempotency_key": "s3-installed-fixture-one-native-exec-1115",
	})
	await(ctx, admin, originalID(e), "SUCCEEDED")
	fmt.Println("S3_INSTALLED_GUEST_FIXTURE_ORIGINAL_EXEC=PASS")
	listing := call(ctx, admin, "box_file_list", map[string]any{
		"box_id": uuid, "path": base, "limit": 2,
	})
	entries, ok := listing["entries"].([]any)
	need(ok && len(entries) == 2 && field(listing, "next_cursor") != "",
		"bounded non-snapshot File listing did not paginate")
	bytesRead := fileBytes(ctx, admin, uuid, base+"/victim")
	need(bytes.Equal(bytesRead, []byte("S3_PHASE2_VICTIM_LOCKED")), "native original victim mismatch")
	need(bytes.Equal(fileBytes(ctx, viewer, uuid, base+"/victim"), bytesRead),
		"authorized Viewer read did not return original Guest bytes")
	fmt.Println("S3_INSTALLED_FILE_LIST_PAGINATION_ADMIN_VIEWER_READ=PASS")
	for _, bad := range []string{"/proc/self/fd/1", base+"/../victim", base+"/sym"} {
		need(rejects(ctx, admin, "box_file_read", map[string]any{
			"box_id": uuid, "path": bad,
		}), "read followed forbidden Guest path "+bad)
	}
	need(rejects(ctx, admin, "box_file_read", map[string]any{
		"box_id": "00000000-0000-4000-8000-000000000000", "path": base+"/victim",
	}), "foreign Box UUID authorized")
	fmt.Println("S3_INSTALLED_MCP_FOREIGN_SYMLINK_MAGIC_PATH_DENIAL=PASS")
	key := "s3-file-installed-one-exclusive-create-1115"
	data := []byte("S3_EXCLUSIVE_CREATED_OK")
	apply := map[string]any{
		"box_id": uuid, "path": base+"/created", "mode": "create",
		"content_base64": base64.StdEncoding.EncodeToString(data), "idempotency_key": key,
	}
	receipt := call(ctx, admin, "box_file_apply", apply)
	operation := originalID(receipt)
	terminal := await(ctx, admin, operation, "SUCCEEDED")
	need(field(terminal, "native_operation_id") == "", "fabricated native Operation ID for synchronous SFTP")
	need(bytes.Equal(fileBytes(ctx, admin, uuid, base+"/created"), data),
		"original created bytes don't match native SFTP")
	replay := call(ctx, admin, "box_file_apply", apply)
	need(originalID(replay) == operation && replay["duplicate"] == true,
		"same original idempotency key dispatched again")
	changed := map[string]any{}
	for k, v := range apply {
		changed[k] = v
	}
	changed["content_base64"] = base64.StdEncoding.EncodeToString([]byte("CHANGED"))
	need(rejects(ctx, admin, "box_file_apply", changed), "same key accepted different content")
	fmt.Println("S3_INSTALLED_MCP_FILE_EXCLUSIVE_CREATE_ORIGINAL_ONCE=PASS")
	need(rejects(ctx, viewer, "box_file_apply", map[string]any{
		"box_id": uuid, "path": base+"/viewer-forbidden", "mode": "create",
		"idempotency_key": "s3-viewer-denied-file-create",
	}), "Viewer performed a native write")
	for _, mode := range []string{"replace", "delete"} {
		need(rejects(ctx, admin, "box_file_apply", map[string]any{
			"box_id": uuid, "path": base+"/victim", "mode": mode,
			"idempotency_key": "s3-native-cas-unsupported-" + mode,
		}), "unsafe native conditional "+mode+" was accepted")
	}
	need(bytes.Equal(fileBytes(ctx, admin, uuid, base+"/victim"), bytesRead),
		"unsafe file mode modified original Guest bytes")
	fmt.Println("S3_INSTALLED_VIEWER_AND_UNSUPPORTED_CAS_DENIAL=PASS")
	for i, path := range []string{base+"/victim", base+"/sym", base+"/hard", base+"/fifo", "/tmp/s3-file-phase2-link/new"} {
		input := map[string]any{
			"box_id": uuid, "path": path, "mode": "create",
			"content_base64": base64.StdEncoding.EncodeToString([]byte("FORBIDDEN")),
			"idempotency_key": fmt.Sprintf("s3-file-denied-existing-%d", i),
		}
		r := call(ctx, admin, "box_file_apply", input)
		await(ctx, admin, originalID(r), "FAILED")
		need(bytes.Equal(fileBytes(ctx, admin, uuid, base+"/victim"), bytesRead),
			"denied SFTP create changed existing original bytes")
	}
	fmt.Println("S3_INSTALLED_MCP_SFTP_LINK_FIFO_PREOPEN_DENIAL_NO_UNKNOWN=PASS")
	// A proved refused native create must not leave an UNKNOWN resource lock.
	safe := map[string]any{
		"box_id": uuid, "path": base+"/post-denial", "mode": "create",
		"content_base64": base64.StdEncoding.EncodeToString([]byte("POST_DENIAL_OK")),
		"idempotency_key": "s3-post-denial-resource-lock-check",
	}
	rr := call(ctx, admin, "box_file_apply", safe)
	await(ctx, admin, originalID(rr), "SUCCEEDED")
	need(bytes.Equal(fileBytes(ctx, admin, uuid, base+"/post-denial"), []byte("POST_DENIAL_OK")),
		"denied Create retained false resource lock")
	fmt.Println("S3_INSTALLED_MCP_POST_DENIAL_LOCK_RELEASE=PASS")
	stop := call(ctx, admin, "box_stop", map[string]any{
		"box_id": uuid, "idempotency_key": "s3-stopped-after-verified-files",
	})
	await(ctx, admin, originalID(stop), "SUCCEEDED")
	del := call(ctx, admin, "box_delete", map[string]any{
		"box_id": uuid, "idempotency_key": "s3-deleted-after-verified-files",
	})
	await(ctx, admin, originalID(del), "SUCCEEDED")
	need(rejects(ctx, admin, "box_get", map[string]any{"box_id": uuid}),
		"deleted original UUID still visible as owned")
	fmt.Println("S3_INSTALLED_MCP_ORIGINAL_BOX_CLEANUP_VERIFIED=PASS")
	fmt.Println("S3_INSTALLED_REAL_HTTP_MCP_FILE_NEGATIVE_AND_IDEMPOTENCY=PASS")
}

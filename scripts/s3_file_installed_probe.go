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
	"path/filepath"
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
		Endpoint:             requiredEnv("BW_S3_MCP_URL"),
		HTTPClient:           &http.Client{Transport: fileBearerTransport{token: token(tokenPath)}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	if err != nil {
		panic(fmt.Errorf("isolated MCP %s connect: %w", name, err))
	}
	return s
}

func call(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) map[string]any {
	result, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil || result == nil {
		panic(fmt.Errorf("original MCP tool %s transport failed: %v; no retry", name, err))
	}
	if result.IsError {
		// Only locally-created disposable Guest fixture paths and typed
		// errors are involved. Trim diagnostics, never log bearer tokens
		// or arbitrary successful Guest file contents.
		b, _ := json.Marshal(result.Content)
		detail := string(b)
		if len(detail) > 400 {
			detail = detail[:400]
		}
		panic(fmt.Errorf("S3_INSTALLED_MCP_TOOL_ERROR name=%s reason=%s", name, detail))
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

type s3GitCheckpoint struct {
	BoxUUID          string `json:"box_uuid"`
	NativeBoxName    string `json:"native_box_name"`
	SetupOp          string `json:"setup_op"`
	FailOp           string `json:"fail_op"`
	FixOp            string `json:"fix_op"`
	PassOp           string `json:"pass_op"`
	FileCreateOp     string `json:"file_create_op"`
	InitialGitCommit string `json:"initial_git_commit"`
	FinalGitCommit   string `json:"final_git_commit"`
	SourceSHA256     string `json:"source_sha256"`
}

func s3GitCheckpointFile() string {
	return requiredEnv("BW_S3_PHASE3_CHECKPOINT")
}

func persistS3GitCheckpoint(data s3GitCheckpoint) {
	filename := s3GitCheckpointFile()
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}
	// Refuse to overwrite a prior incomplete native attempt or adopt a
	// previous Agent's Box UUID from a different CI run.
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		panic(err)
	}
	if _, err := f.Write(append(out, byte(10))); err != nil {
		panic(err)
	}
	if err := f.Sync(); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	dir, err := os.Open(filepath.Dir(filename))
	if err != nil {
		panic(err)
	}
	if err := dir.Sync(); err != nil {
		panic(err)
	}
	_ = dir.Close()
}

func loadS3GitCheckpoint() s3GitCheckpoint {
	b, err := os.ReadFile(s3GitCheckpointFile())
	if err != nil {
		panic(err)
	}
	var cp s3GitCheckpoint
	if err := json.Unmarshal(b, &cp); err != nil {
		panic(err)
	}
	need(cp.NativeBoxName == "s3-file-phase2-ci" && len(cp.BoxUUID) >= 32 &&
		len(cp.SetupOp) == 32 && len(cp.FailOp) == 32 && len(cp.FixOp) == 32 &&
		len(cp.PassOp) == 32 && len(cp.FileCreateOp) == 32 &&
		len(cp.InitialGitCommit) == 40 && len(cp.FinalGitCommit) == 40 &&
		len(cp.SourceSHA256) == 64, "invalid original Git+native UUID checkpoint")
	return cp
}

func s3GuestExec(ctx context.Context, admin *mcp.ClientSession, uuid, key, script string) (string, int, string, string) {
	r := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c", script},
		"timeout_ms": 180000, "idempotency_key": key,
		"work_ref": "lab/s3/one-agent-git", "agent_ref": "s3-supervised-single-agent",
	})
	id := originalID(r)
	// Operation state SUCCEEDED means native Exec completed, NOT guest exit 0.
	await(ctx, admin, id, "SUCCEEDED")
	var stdout, stderr string
	exitCode := -9999
	for _, stream := range []string{"stdout", "stderr"} {
		o := call(ctx, admin, "operation_output_get", map[string]any{
			"operation_id": id, "stream": stream, "limit": 4096,
		})
		exit, known := o["exit_code"].(float64)
		need(known, "original Guest Exec has no independently read exit code")
		if exitCode != -9999 {
			need(exitCode == int(exit), "inconsistent original Guest output exit codes")
		}
		exitCode = int(exit)
		b, err := base64.StdEncoding.DecodeString(field(o, "content_base64"))
		if err != nil {
			panic(err)
		}
		// Hard-bound and never log arbitrary source or secrets.
		need(len(b) <= 4096, "Guest Exec output exceeded verifier bound")
		if stream == "stdout" {
			stdout = string(b)
		} else {
			stderr = string(b)
		}
	}
	return id, exitCode, stdout, stderr
}

func gitCommitLine(text, prefix string) string {
	for _, line := range strings.Split(text, string(byte(10))) {
		if strings.HasPrefix(line, prefix) {
			out := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if len(out) == 40 && strings.IndexFunc(out, func(c rune) bool {
				return !strings.ContainsRune("0123456789abcdef", c)
			}) == -1 {
				return out
			}
		}
	}
	panic("original Guest Git commit is absent or invalid: " + prefix)
}

func runS3GuestGitPhase3(ctx context.Context, admin, viewer *mcp.ClientSession, uuid string) {
	// One trusted attended Agent; no remote shared repo, network peer Tool
	// or unsafe File replace. Guest has a native Git working tree.
	const dir = "/root/s3-guest-git"
	const wrongSource = `def double(value):
    return value * 2 + 1
`
	const rightSource = `def double(value):
    return value * 2
`
	const unitTest = `import unittest
from calc import double

class TestCalc(unittest.TestCase):
    def test_double(self):
        self.assertEqual(double(2), 4)
`
	installScript := "set -eu; if ! command -v git >/dev/null 2>&1 || ! command -v python3 >/dev/null 2>&1; then " +
		"DEBIAN_FRONTEND=noninteractive apt-get update -qq && " +
		"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends git python3; fi; " +
		"command -v git; command -v python3; mkdir -p " + dir
	installID, code, _, _ := s3GuestExec(ctx, admin, uuid, "s3-guest-git-dependencies-once", installScript)
	need(code == 0, "Guest git/python prerequisite failed, preserve original "+installID)
	fmt.Println("S3_PHASE3_GUEST_GIT_PYTHON_NATIVE_PREREQUISITES=PASS")
	var created []string
	for i, v := range []struct {
		path string
		data string
	}{{dir + "/calc.py", wrongSource}, {dir + "/test_calc.py", unitTest}} {
		r := call(ctx, admin, "box_file_apply", map[string]any{
			"box_id": uuid, "path": v.path, "mode": "create",
			"content_base64":  base64.StdEncoding.EncodeToString([]byte(v.data)),
			"idempotency_key": fmt.Sprintf("s3-guest-git-file-create-%d-once", i),
			"work_ref":        "lab/s3/one-agent-git", "agent_ref": "s3-supervised-single-agent",
		})
		id := originalID(r)
		await(ctx, admin, id, "SUCCEEDED")
		created = append(created, id)
		need(bytes.Equal(fileBytes(ctx, viewer, uuid, v.path), []byte(v.data)),
			"Viewer could not independently read original Git source")
	}
	fmt.Println("S3_PHASE3_MCP_FILE_EXCLUSIVE_CREATE_TWO_SOURCE_FILES=PASS")
	setupScript := "set -eu; cd " + dir +
		"; git init -q; git config user.name 'S3 CI Agent'; " +
		"git config user.email 's3-agent@example.invalid'; " +
		"git add calc.py test_calc.py; " +
		"git commit -q -m 'test: intentionally failing double'; " +
		"echo S3_GIT_INITIAL_COMMIT=$(git rev-parse HEAD)"
	setupID, setupExit, setupOut, _ := s3GuestExec(ctx, admin, uuid, "s3-git-initial-commit-once", setupScript)
	need(setupExit == 0, "original Git commit setup exited nonzero")
	firstCommit := gitCommitLine(setupOut, "S3_GIT_INITIAL_COMMIT=")
	fmt.Println("S3_PHASE3_GIT_ORIGINAL_COMMIT=" + firstCommit)
	failScript := "cd " + dir + "; python3 -B -m unittest -v test_calc"
	failID, failExit, _, failErr := s3GuestExec(ctx, admin, uuid, "s3-git-deliberate-failing-test-once", failScript)
	need(failExit == 1, "deliberate regression must exit exactly 1; native terminal alone is insufficient")
	need(strings.Contains(failErr, "AssertionError: 5 != 4") &&
		strings.Contains(failErr, "FAILED"), "original persisted stderr has no actual unittest failure")
	fmt.Println("S3_PHASE3_GUEST_TEST_FAILED_ORIGINAL_EXIT_ONE=PASS")
	fmt.Println("S3_PHASE3_GUEST_FAILED_TEST_ORIGINAL_OPERATION_ID=" + failID)
	// File replace is intentionally UNSUPPORTED; authorized Guest Exec/Git
	// is the verified safe route to an edit in this trusted supervised Box.
	fixScript := "set -eu; cd " + dir +
		"; sed -i 's/ + 1$//' calc.py; " +
		"git add calc.py; git commit -q -m 'fix: double returns exact value'; " +
		"test $(git rev-list --count HEAD) = 2; " +
		"test $(git status --porcelain | wc -l) -eq 0; " +
		"echo S3_GIT_REPAIRED_COMMIT=$(git rev-parse HEAD)"
	fixID, fixExit, fixOut, _ := s3GuestExec(ctx, admin, uuid, "s3-git-guest-fix-and-commit-once", fixScript)
	need(fixExit == 0, "Guest authorized repair or Git second commit failed")
	fixedCommit := gitCommitLine(fixOut, "S3_GIT_REPAIRED_COMMIT=")
	need(fixedCommit != firstCommit, "Git final commit matches deliberately failing original")
	need(bytes.Equal(fileBytes(ctx, viewer, uuid, dir+"/calc.py"), []byte(rightSource)),
		"File tool did not return corrected Guest source byte-for-byte")
	passScript := "cd " + dir + "; python3 -B -m unittest -v test_calc"
	passID, passExit, _, passErr := s3GuestExec(ctx, admin, uuid, "s3-git-passing-test-original-once", passScript)
	need(passExit == 0 && strings.Contains(passErr, "Ran 1 test") &&
		strings.Contains(passErr, "OK"), "original Guest regression was not repaired")
	fmt.Println("S3_PHASE3_GUEST_REPAIR_GIT_COMMIT_AND_TEST_PASS=PASS")
	sum := sha256.Sum256([]byte(rightSource))
	persistS3GitCheckpoint(s3GitCheckpoint{
		BoxUUID: uuid, NativeBoxName: "s3-file-phase2-ci",
		SetupOp: setupID, FailOp: failID, FixOp: fixID, PassOp: passID,
		FileCreateOp: created[0], InitialGitCommit: firstCommit,
		FinalGitCommit: fixedCommit, SourceSHA256: hex.EncodeToString(sum[:]),
	})
	fmt.Println("S3_PHASE3_SUPERVISED_SINGLE_AGENT_EXTERNAL_CHECKPOINT_FSYNC=PASS")
	fmt.Println("S3_PHASE3_GUEST_GIT_FAIL_FIX_PASS_STAGE=PASS")
}

func cleanupS3GuestGitPhase3(ctx context.Context, admin *mcp.ClientSession) {
	cp := loadS3GitCheckpoint()
	// The CI native verifier independently read and compared this original
	// UUID and source *before* cleanup. No cleanup of unknown old names.
	original := object(call(ctx, admin, "box_get", map[string]any{"box_id": cp.BoxUUID}), "box")
	need(field(original, "box_id") == cp.BoxUUID, "cleanup cannot adopt replacement Box")
	stop := call(ctx, admin, "box_stop", map[string]any{
		"box_id": cp.BoxUUID, "idempotency_key": "s3-phase3-verified-native-stop-once",
	})
	await(ctx, admin, originalID(stop), "SUCCEEDED")
	del := call(ctx, admin, "box_delete", map[string]any{
		"box_id": cp.BoxUUID, "idempotency_key": "s3-phase3-verified-native-delete-once",
	})
	await(ctx, admin, originalID(del), "SUCCEEDED")
	need(rejects(ctx, admin, "box_get", map[string]any{"box_id": cp.BoxUUID}),
		"original UUID still visible after S3 phase3 cleanup")
	fmt.Println("S3_PHASE3_MCP_ORIGINAL_GIT_BOX_STOP_DELETE_VERIFIED=PASS")
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
	if os.Getenv("BW_S3_PHASE3_MODE") == "cleanup" {
		agent := connect(ctx, "admin", requiredEnv("BW_S3_ADMIN_TOKEN_FILE"))
		defer agent.Close()
		cleanupS3GuestGitPhase3(ctx, agent)
		return
	}
	need(os.Getenv("BW_S3_PHASE3_MODE") == "" ||
		os.Getenv("BW_S3_PHASE3_MODE") == "stage", "unsupported S3 phase3 verifier mode")
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
		"start": true, "internet": os.Getenv("BW_S3_PHASE3_MODE") == "stage", "idempotency_key": "s3-file-installed-box-create-1115",
	}
	created := call(ctx, admin, "box_create", create)
	createID := originalID(created)
	done := await(ctx, admin, createID, "SUCCEEDED")
	uuid := field(done, "box_uuid")
	need(len(uuid) >= 32, "new Box native UUID missing")
	fmt.Println("S3_INSTALLED_BOX_ORIGINAL_OPERATION_ID=" + createID)
	fmt.Println("S3_INSTALLED_EXACT_NATIVE_BOX_UUID=" + uuid)
	state := object(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	need(field(state, "box_id") == uuid && field(state, "native_state") == "Running",
		"native owned Guest is not running")
	fmt.Println("S3_INSTALLED_BOX_CREATE_ORIGINAL_UUID=PASS")
	base := "/root/s3-file-phase2"
	fixture := "mkdir -p " + base + "; " +
		"printf S3_PHASE2_VICTIM_LOCKED > " + base + "/victim; " +
		"ln -s " + base + "/victim " + base + "/sym; " +
		"ln " + base + "/victim " + base + "/hard; " +
		"mkfifo " + base + "/fifo; " +
		"ln -s " + base + " /root/s3-file-phase2-link; echo S3_FIXTURE_READY"
	e := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-ec", fixture},
		"idempotency_key": "s3-installed-fixture-one-native-exec-1115",
	})
	fixtureID := originalID(e)
	await(ctx, admin, fixtureID, "SUCCEEDED")
	out := call(ctx, admin, "operation_output_get", map[string]any{
		"operation_id": fixtureID, "stream": "stdout", "limit": 4096,
	})
	exit, observed := out["exit_code"].(float64)
	need(observed && exit == 0, "fixture Guest native Exec did not exit 0")
	output, err := base64.StdEncoding.DecodeString(field(out, "content_base64"))
	if err != nil {
		panic(err)
	}
	need(bytes.Contains(output, []byte("S3_FIXTURE_READY")),
		"Guest stdout missing fixture marker despite native exit=0")
	fmt.Println("S3_INSTALLED_GUEST_FIXTURE_ORIGINAL_EXEC_EXIT_ZERO=PASS")
	// Read the fixed native victim before directory enumeration to separate
	// native file/path visibility from SFTP readdir transport problems.
	bytesRead := fileBytes(ctx, admin, uuid, base+"/victim")
	need(bytes.Equal(bytesRead, []byte("S3_PHASE2_VICTIM_LOCKED")),
		"native original Guest fixture missing on first MCP file read")
	fmt.Println("S3_INSTALLED_FIRST_REGULAR_FILE_SFTP_READ_BEFORE_LIST=PASS")
	listing := call(ctx, admin, "box_file_list", map[string]any{
		"box_id": uuid, "path": base, "limit": 2,
	})
	entries, ok := listing["entries"].([]any)
	need(ok && len(entries) == 2 && field(listing, "next_cursor") != "",
		"bounded non-snapshot File listing did not paginate")
	need(bytes.Equal(bytesRead, []byte("S3_PHASE2_VICTIM_LOCKED")), "native original victim mismatch")
	need(bytes.Equal(fileBytes(ctx, viewer, uuid, base+"/victim"), bytesRead),
		"authorized Viewer read did not return original Guest bytes")
	fmt.Println("S3_INSTALLED_FILE_LIST_PAGINATION_ADMIN_VIEWER_READ=PASS")
	hugeFixture := "mkdir -p /root/s3-file-phase2/many; " +
		"i=0; while [ \"$i\" -lt 6000 ]; do " +
		": > /root/s3-file-phase2/many/entry-$i; i=$((i+1)); done"
	huge := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-ec", hugeFixture},
		"idempotency_key": "s3-installed-huge-native-dir-fixture-1115",
	})
	hugeID := originalID(huge)
	await(ctx, admin, hugeID, "SUCCEEDED")
	hugeExit := call(ctx, admin, "operation_output_get", map[string]any{
		"operation_id": hugeID, "stream": "stdout", "limit": 512,
	})
	hugeCode, hugeKnown := hugeExit["exit_code"].(float64)
	need(hugeKnown && hugeCode == 0, "large-directory fixture failed in original Guest")
	need(rejects(ctx, admin, "box_file_list", map[string]any{
		"box_id": uuid, "path": base + "/many", "limit": 20,
	}), "SFTP v1 eager huge directory exceeded safety budget but was accepted")
	need(bytes.Equal(fileBytes(ctx, admin, uuid, base+"/victim"), bytesRead),
		"large-directory refusal changed native victim bytes")
	fmt.Println("S3_INSTALLED_MCP_HUGE_DIRECTORY_READ_BUDGET_FAIL_CLOSED=PASS")
	for _, bad := range []string{"/proc/self/fd/1", base + "/../victim", base + "/sym"} {
		need(rejects(ctx, admin, "box_file_read", map[string]any{
			"box_id": uuid, "path": bad,
		}), "read followed forbidden Guest path "+bad)
	}
	need(rejects(ctx, admin, "box_file_read", map[string]any{
		"box_id": "00000000-0000-4000-8000-000000000000", "path": base + "/victim",
	}), "foreign Box UUID authorized")
	fmt.Println("S3_INSTALLED_MCP_FOREIGN_SYMLINK_MAGIC_PATH_DENIAL=PASS")
	key := "s3-file-installed-one-exclusive-create-1115"
	data := []byte("S3_EXCLUSIVE_CREATED_OK")
	apply := map[string]any{
		"box_id": uuid, "path": base + "/created", "mode": "create",
		"content_base64": base64.StdEncoding.EncodeToString(data), "idempotency_key": key,
	}
	receipt := call(ctx, admin, "box_file_apply", apply)
	operation := originalID(receipt)
	fmt.Println("S3_INSTALLED_FILE_CREATE_ORIGINAL_OPERATION_ID=" + operation)
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
		"box_id": uuid, "path": base + "/viewer-forbidden", "mode": "create",
		"idempotency_key": "s3-viewer-denied-file-create",
	}), "Viewer performed a native write")
	for _, mode := range []string{"replace", "delete"} {
		need(rejects(ctx, admin, "box_file_apply", map[string]any{
			"box_id": uuid, "path": base + "/victim", "mode": mode,
			"idempotency_key": "s3-native-cas-unsupported-" + mode,
		}), "unsafe native conditional "+mode+" was accepted")
	}
	need(bytes.Equal(fileBytes(ctx, admin, uuid, base+"/victim"), bytesRead),
		"unsafe file mode modified original Guest bytes")
	fmt.Println("S3_INSTALLED_VIEWER_AND_UNSUPPORTED_CAS_DENIAL=PASS")
	for i, path := range []string{base + "/victim", base + "/sym", base + "/hard", base + "/fifo", "/root/s3-file-phase2-link/new"} {
		input := map[string]any{
			"box_id": uuid, "path": path, "mode": "create",
			"content_base64":  base64.StdEncoding.EncodeToString([]byte("FORBIDDEN")),
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
		"box_id": uuid, "path": base + "/post-denial", "mode": "create",
		"content_base64":  base64.StdEncoding.EncodeToString([]byte("POST_DENIAL_OK")),
		"idempotency_key": "s3-post-denial-resource-lock-check",
	}
	rr := call(ctx, admin, "box_file_apply", safe)
	await(ctx, admin, originalID(rr), "SUCCEEDED")
	need(bytes.Equal(fileBytes(ctx, admin, uuid, base+"/post-denial"), []byte("POST_DENIAL_OK")),
		"denied Create retained false resource lock")
	fmt.Println("S3_INSTALLED_MCP_POST_DENIAL_LOCK_RELEASE=PASS")
	if os.Getenv("BW_S3_PHASE3_MODE") == "stage" {
		runS3GuestGitPhase3(ctx, admin, viewer, uuid)
		return
	}
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

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
	"net"
	"net/http"
	"net/url"
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
	response, err := http.DefaultTransport.RoundTrip(copyReq)
	if err == nil && response != nil && os.Getenv("BW_S5B1_EXACT_TOOLS") == "true" && req.Method == http.MethodPost {
		cache := strings.ToLower(response.Header.Get("Cache-Control"))
		if !strings.Contains(cache, "private") || !strings.Contains(cache, "no-store") || !strings.Contains(strings.Join(response.Header.Values("Vary"), ","), "Authorization") {
			response.Body.Close()
			return nil, fmt.Errorf("S5B1 private response cache boundary missing")
		}
	}
	return response, err
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
	FinalGitClean    bool   `json:"final_git_clean"`
	NativeSetupOp    string `json:"native_setup_op"`
	NativeFailOp     string `json:"native_fail_op"`
	NativeFixOp      string `json:"native_fix_op"`
	NativePassOp     string `json:"native_pass_op"`
	SourceSHA256     string `json:"source_sha256"`
}

func s3GitCheckpointFile() string {
	return requiredEnv("BW_S3_PHASE3_CHECKPOINT")
}

func persistS3GitCheckpoint(data s3GitCheckpoint) {
	persistS3GitCheckpointAt(s3GitCheckpointFile(), data)
}

func persistS3GitCheckpointAt(filename string, data s3GitCheckpoint) {
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
		cp.FinalGitClean && len(cp.NativeSetupOp) == 36 && len(cp.NativeFailOp) == 36 &&
		len(cp.NativeFixOp) == 36 && len(cp.NativePassOp) == 36 &&
		len(cp.SourceSHA256) == 64, "invalid original Git+native UUID checkpoint")
	return cp
}

func s3GuestExec(ctx context.Context, admin *mcp.ClientSession, uuid, key, script string) (string, int, string, string) {
	workRef, agentRef := "lab/s3/one-agent-git", "s3-supervised-single-agent"
	if os.Getenv("BW_S5B_TC015") == "true" {
		workRef, agentRef = "lab/s5b/two-actor-git", "s5b-actor-a"
		if os.Getenv("BW_S3_PHASE3_MODE") == "handoff" {
			agentRef = "s5b-actor-b"
		}
	}
	r := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c", script},
		"timeout_ms": 300000, "idempotency_key": key,
		"work_ref": workRef, "agent_ref": agentRef,
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

// Read the original durable Incus ID while the supervised source process
// still holds the original BoxWeave operation ID. No native redispatch.
func s3OriginalNativeID(ctx context.Context, admin *mcp.ClientSession, uuid, opID string) string {
	op := object(call(ctx, admin, "operation_get", map[string]any{
		"operation_id": opID,
	}), "operation")
	need(field(op, "state") == "SUCCEEDED" && field(op, "action") == "box_exec" &&
		field(op, "box_uuid") == uuid,
		"initial original Guest Exec receipt lost its Box and action identity")
	nativeID := field(op, "native_operation_id")
	need(len(nativeID) == 36, "initial original Guest Exec has no Incus Operation UUID")
	return nativeID
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

func s3ApprovedPackageProxy() string {
	raw := requiredEnv("BW_S3_PACKAGE_PROXY_URL")
	u, err := url.Parse(raw)
	need(err == nil && u != nil && u.Scheme == "http" &&
		u.User == nil && u.Port() == "18466" && u.Path == "" && u.RawQuery == "" &&
		u.Fragment == "", "unapproved S3 one-use package relay")
	ip := net.ParseIP(u.Hostname())
	need(ip != nil && ip.IsPrivate() && ip.To4() != nil,
		"package relay must be private IPv4")
	return raw
}

func runS3GuestGitPhase3(ctx context.Context, admin, peer *mcp.ClientSession, uuid string) {
	// One trusted attended Agent; no remote shared repo, network peer Tool
	// or unsafe File replace. Guest has a native Git working tree.
	const dir = "/root/s3-guest-git"
	const wrongSource = `def double(value):
    return value * 2 + 1
`
	const unitTest = `import unittest
from calc import double

class TestCalc(unittest.TestCase):
    def test_double(self):
        self.assertEqual(double(2), 4)
`
	installScript := `set -eu
if ! command -v git >/dev/null 2>&1 || ! command -v python3 >/dev/null 2>&1; then
  echo 'Acquire::ForceIPv4 "true";' > /etc/apt/apt.conf.d/99-s3-ipv4
  printf 'Acquire::https::Proxy "%s";' '@S3_SCOPED_PROXY@' > /etc/apt/apt.conf.d/98-s3-relay
  for sources in /etc/apt/sources.list /etc/apt/sources.list.d/ubuntu.sources; do
    if test -f "$sources"; then
      sed -i -e 's#http://archive.ubuntu.com#https://archive.ubuntu.com#g'         -e 's#http://security.ubuntu.com#https://security.ubuntu.com#g' "$sources"
    fi
  done
  if grep -ER '^(deb |URIs:).*http://' /etc/apt/sources.list /etc/apt/sources.list.d 2>/dev/null; then
    echo S3_GUEST_PLAINTEXT_APT_REFUSED >&2
    exit 72
  fi
  DEBIAN_FRONTEND=noninteractive apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends git python3
fi
command -v git
command -v python3
mkdir -p /root/s3-guest-git
`
	installScript = strings.ReplaceAll(installScript, "@S3_SCOPED_PROXY@", s3ApprovedPackageProxy())
	installID, code, stdout, stderr := s3GuestExec(ctx, admin, uuid, "s3-guest-git-dependencies-once", installScript)
	if code != 0 {
		// CI-owned disposable guest only. Keep a bounded diagnostic from
		// the original native Exec; never redispatch an uncertain command.
		if len(stdout) > 700 {
			stdout = stdout[len(stdout)-700:]
		}
		if len(stderr) > 700 {
			stderr = stderr[len(stderr)-700:]
		}
		fmt.Printf("S3_PHASE3_GIT_PREREQUISITE_ORIGINAL_EXIT=%d OP=%s STDOUT=%q STDERR=%q\n",
			code, installID, stdout, stderr)
	}
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
		need(bytes.Equal(fileBytes(ctx, peer, uuid, v.path), []byte(v.data)),
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

	cp := s3GitCheckpoint{
		BoxUUID: uuid, NativeBoxName: "s3-file-phase2-ci",
		SetupOp: setupID, FailOp: failID, FileCreateOp: created[0],
		InitialGitCommit: firstCommit,
		NativeSetupOp:    s3OriginalNativeID(ctx, admin, uuid, setupID),
		NativeFailOp:     s3OriginalNativeID(ctx, admin, uuid, failID),
	}
	if os.Getenv("BW_S5B_TC015") == "true" {
		persistS3GitCheckpointAt(requiredEnv("BW_S5B_HANDOFF_CHECKPOINT"), cp)
		fmt.Printf("S5B_TC015_ACTOR_A_FAILED_CHECKPOINT=PASS BW=%s NATIVE=%s\n", cp.FailOp, cp.NativeFailOp)
		return // Process A exits before the repair. The runner commits this external checkpoint.
	}
	completeS3GuestGitPhase3(ctx, admin, peer, cp)
}

func completeS3GuestGitPhase3(ctx context.Context, actor, observer *mcp.ClientSession, cp s3GitCheckpoint) {
	const dir = "/root/s3-guest-git"
	const rightSource = "def double(value):\n    return value * 2\n"
	uuid := cp.BoxUUID
	// Native conditional File replace remains unsupported. Trusted guest
	// Exec/Git is the normal approved editing path for this disposable Box.
	fixScript := "set -eu; cd " + dir +
		"; sed -i 's/ + 1$//' calc.py; " +
		"git add calc.py; git commit -q -m 'fix: double returns exact value'; " +
		"test $(git rev-list --count HEAD) = 2; " +
		"test $(git status --porcelain | wc -l) -eq 0; " +
		"echo S3_GIT_REPAIRED_COMMIT=$(git rev-parse HEAD)"
	fixID, fixExit, fixOut, _ := s3GuestExec(ctx, actor, uuid, "s3-git-guest-fix-and-commit-once", fixScript)
	need(fixExit == 0, "Guest authorized repair or Git second commit failed")
	fixedCommit := gitCommitLine(fixOut, "S3_GIT_REPAIRED_COMMIT=")
	need(fixedCommit != cp.InitialGitCommit, "Git final commit matches deliberately failing original")
	need(bytes.Equal(fileBytes(ctx, observer, uuid, dir+"/calc.py"), []byte(rightSource)),
		"File tool did not return corrected Guest source byte-for-byte")
	passID, passExit, _, passErr := s3GuestExec(ctx, actor, uuid, "s3-git-passing-test-original-once", "cd "+dir+"; python3 -B -m unittest -v test_calc")
	need(passExit == 0 && strings.Contains(passErr, "Ran 1 test") &&
		strings.Contains(passErr, "OK"), "original Guest regression was not repaired")
	fmt.Println("S3_PHASE3_GUEST_REPAIR_GIT_COMMIT_AND_TEST_PASS=PASS")
	sum := sha256.Sum256([]byte(rightSource))
	cp.FixOp, cp.PassOp = fixID, passID
	cp.FinalGitCommit, cp.FinalGitClean = fixedCommit, true
	cp.NativeFixOp = s3OriginalNativeID(ctx, actor, uuid, fixID)
	cp.NativePassOp = s3OriginalNativeID(ctx, actor, uuid, passID)
	cp.SourceSHA256 = hex.EncodeToString(sum[:])
	persistS3GitCheckpoint(cp)
	if os.Getenv("BW_S5B_TC015") == "true" {
		fmt.Printf("S5B_TC015_ACTOR_B_REPAIRED_CHECKPOINT=PASS BW=%s NATIVE=%s\n", cp.PassOp, cp.NativePassOp)
	} else {
		fmt.Println("S3_PHASE3_SUPERVISED_SINGLE_AGENT_EXTERNAL_CHECKPOINT_FSYNC=PASS")
	}
	fmt.Println("S3_PHASE3_GUEST_GIT_FAIL_FIX_PASS_STAGE=PASS")
}

func resumeS5BGitHandoff(ctx context.Context, actor *mcp.ClientSession) {
	const dir = "/root/s3-guest-git"
	b, err := os.ReadFile(requiredEnv("BW_S5B_HANDOFF_CHECKPOINT"))
	need(err == nil && len(b) <= 8192, "missing or oversized external handoff")
	var cp s3GitCheckpoint
	need(json.Unmarshal(b, &cp) == nil && cp.NativeBoxName == "s3-file-phase2-ci" &&
		len(cp.BoxUUID) == 36 && len(cp.SetupOp) == 32 && len(cp.FailOp) == 32 && len(cp.FileCreateOp) == 32 &&
		len(cp.NativeSetupOp) == 36 && len(cp.NativeFailOp) == 36 && len(cp.InitialGitCommit) == 40 &&
		cp.FixOp == "" && cp.PassOp == "" && cp.FinalGitCommit == "" && !cp.FinalGitClean &&
		cp.NativeFixOp == "" && cp.NativePassOp == "" && cp.SourceSHA256 == "", "invalid partial handoff checkpoint")
	box := object(call(ctx, actor, "box_get", map[string]any{"box_id": cp.BoxUUID}), "box")
	need(field(box, "box_id") == cp.BoxUUID && field(box, "native_state") == "Running", "handoff original Box not verified")
	for _, original := range []struct {
		id, native, marker, stream string
		exit                       float64
	}{
		{cp.SetupOp, cp.NativeSetupOp, "S3_GIT_INITIAL_COMMIT=" + cp.InitialGitCommit, "stdout", 0},
		{cp.FailOp, cp.NativeFailOp, "AssertionError: 5 != 4", "stderr", 1},
	} {
		need(s3OriginalNativeID(ctx, actor, cp.BoxUUID, original.id) == original.native, "handoff native receipt mismatch")
		out := call(ctx, actor, "operation_output_get", map[string]any{"operation_id": original.id, "stream": original.stream, "limit": 4096})
		decoded, e := base64.StdEncoding.DecodeString(field(out, "content_base64"))
		need(e == nil && out["exit_code"] == original.exit && bytes.Contains(decoded, []byte(original.marker)), "handoff original output mismatch")
	}
	fileOp := object(call(ctx, actor, "operation_get", map[string]any{"operation_id": cp.FileCreateOp}), "operation")
	need(field(fileOp, "state") == "SUCCEEDED" && field(fileOp, "box_uuid") == cp.BoxUUID && field(fileOp, "action") == "box_file_apply", "handoff original file receipt mismatch")
	need(bytes.Equal(fileBytes(ctx, actor, cp.BoxUUID, dir+"/calc.py"), []byte("def double(value):\n    return value * 2 + 1\n")), "handoff original source mismatch")
	_, code, head, stderr := s3GuestExec(ctx, actor, cp.BoxUUID, "s5b-b-independent-git-head-read", "set -eu; cd "+dir+"; test -z \"$(git status --porcelain)\"; git rev-parse HEAD")
	need(code == 0 && strings.TrimSpace(head) == cp.InitialGitCommit && stderr == "", "handoff stale HEAD or dirty working tree")
	repro, code, _, stderr := s3GuestExec(ctx, actor, cp.BoxUUID, "s5b-b-reproduce-failure-once", "cd "+dir+"; python3 -B -m unittest -v test_calc")
	need(code == 1 && strings.Contains(stderr, "AssertionError: 5 != 4"), "second Actor did not independently reproduce failure")
	fmt.Printf("S5B_TC015_SECOND_ACTOR_REPRODUCED_ORIGINAL_FAIL=PASS BW=%s NATIVE=%s\n", repro, s3OriginalNativeID(ctx, actor, cp.BoxUUID, repro))
	completeS3GuestGitPhase3(ctx, actor, actor, cp)
}

func verifyS3GuestGitOriginals(ctx context.Context, admin *mcp.ClientSession) {
	cp := loadS3GitCheckpoint()
	box := object(call(ctx, admin, "box_get", map[string]any{"box_id": cp.BoxUUID}), "box")
	need(field(box, "box_id") == cp.BoxUUID, "fresh MCP client cannot confirm the original native Box")
	checks := []struct {
		label, id, nativeID        string
		wantExit                   int
		stdoutMarker, stderrMarker string
	}{
		{"setup", cp.SetupOp, cp.NativeSetupOp, 0, "S3_GIT_INITIAL_COMMIT=" + cp.InitialGitCommit, ""},
		{"deliberate-failure", cp.FailOp, cp.NativeFailOp, 1, "", "AssertionError: 5 != 4"},
		{"guest-fix", cp.FixOp, cp.NativeFixOp, 0, "S3_GIT_REPAIRED_COMMIT=" + cp.FinalGitCommit, ""},
		{"passing-test", cp.PassOp, cp.NativePassOp, 0, "", "Ran 1 test"},
	}
	for _, check := range checks {
		op := object(call(ctx, admin, "operation_get", map[string]any{"operation_id": check.id}), "operation")
		need(field(op, "state") == "SUCCEEDED" &&
			field(op, "native_operation_id") == check.nativeID &&
			field(op, "box_uuid") == cp.BoxUUID && field(op, "action") == "box_exec",
			"fresh original Exec does not match its prereboot native ID/Box/action: "+check.label)
		for _, stream := range []string{"stdout", "stderr"} {
			out := call(ctx, admin, "operation_output_get", map[string]any{
				"operation_id": check.id, "stream": stream, "limit": 4096,
			})
			exit, known := out["exit_code"].(float64)
			need(known && int(exit) == check.wantExit,
				"fresh original Guest exit differs from the original terminal: "+check.label)
			content, err := base64.StdEncoding.DecodeString(field(out, "content_base64"))
			need(err == nil && len(content) <= 4096,
				"fresh original Guest output is unverified: "+check.label)
			if stream == "stdout" && check.stdoutMarker != "" {
				need(bytes.Contains(content, []byte(check.stdoutMarker)),
					"fresh original Git stdout commit differs: "+check.label)
			}
			if stream == "stderr" && check.stderrMarker != "" {
				need(bytes.Contains(content, []byte(check.stderrMarker)),
					"fresh original Guest test stderr differs: "+check.label)
			}
		}
		fmt.Printf("S3_PHASE3_FRESH_ORIGINAL_EXEC=%s BW=%s NATIVE=%s EXIT=%d\n",
			check.label, check.id, field(op, "native_operation_id"), check.wantExit)
	}
	fileOp := object(call(ctx, admin, "operation_get", map[string]any{
		"operation_id": cp.FileCreateOp,
	}), "operation")
	need(field(fileOp, "state") == "SUCCEEDED" && field(fileOp, "native_operation_id") == "",
		"fresh SFTP exclusive create receipt fabricated or lost its original status")
	finalBytes := fileBytes(ctx, admin, cp.BoxUUID, "/root/s3-guest-git/calc.py")
	finalSHA := sha256.Sum256(finalBytes)
	need(hex.EncodeToString(finalSHA[:]) == cp.SourceSHA256,
		"fresh official MCP file bytes do not match the original durable checkpoint")
	fmt.Println("S3_PHASE3_FRESH_CLIENT_ORIGINAL_OPERATIONS_AND_FILE=PASS")
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

func s5b1ExactCatalog(ctx context.Context, admin *mcp.ClientSession, tools []*mcp.Tool) {
	want := map[string]bool{}
	for _, name := range strings.Fields("host_status box_create box_list box_get box_update box_start box_stop box_delete box_exec operation_get operation_list operation_output_get operation_cancel operation_recover box_file_list box_file_read box_file_apply box_peer_set box_activity_get port_forward_list port_forward_create port_forward_delete box_wait_ready") {
		want[name] = true
	}
	need(len(tools) == len(want), "S5B1 exact catalog mismatch")
	status := call(ctx, admin, "host_status", map[string]any{})
	need(field(status, "commit") == "a3e02b4750edcabb80427e3311a19614d82b0029" && status["mcp_ready"] == true, "S5B1 installed identity")
	need(field(status, "binary_sha256") == "a7a6a9fb7f32886cad8a891581cdc2eb5a8938200f48ea35b3a8cee3fed3fbf6", "S5B1 exact installed binary mismatch")
	for _, tool := range tools {
		need(want[tool.Name] && tool.Description != "" && tool.Annotations != nil, "S5B1 missing actual metadata "+tool.Name)
		delete(want, tool.Name)
		raw, e := json.Marshal(tool.InputSchema)
		need(e == nil && strings.Contains(string(raw), "\"additionalProperties\":false"), "S5B1 input schema not closed "+tool.Name)
		need(rejects(ctx, admin, tool.Name, map[string]any{"s5b1_nonexistent_field": true}), "S5B1 invalid arguments accepted "+tool.Name)
		// A rejected call must not be counted from a disconnected client.
		need(field(call(ctx, admin, "host_status", map[string]any{}), "commit") == field(status, "commit"), "S5B1 transport not connected after refusal")
		fmt.Println("S5B1_EXACT_MCP_SCHEMA_PREFLIGHT=" + tool.Name)
	}
	need(len(want) == 0, "S5B1 missing catalog tools")
	fmt.Println("S5B1_STAGE=initialize")
	init := admin.InitializeResult()
	need(init != nil && strings.Contains(init.Instructions, "UNKNOWN") && strings.Contains(init.Instructions, "idempotency_key"), "S5B1 initialize instructions missing")
	for _, name := range []string{"box_list", "operation_list", "box_activity_get", "port_forward_list"} {
		fmt.Println("S5B1_STAGE=typed_text_" + name)
		result, e := admin.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: map[string]any{}})
		need(e == nil && result != nil && !result.IsError && result.StructuredContent != nil && len(result.Content) > 0, "S5B1 typed and text result missing "+name)
		raw, e := json.Marshal(result.StructuredContent)
		need(e == nil, "S5B1 typed result marshal")
		var a map[string]any
		need(json.Unmarshal(raw, &a) == nil, "S5B1 typed result parse")
		compatible := false
		for _, part := range result.Content {
			if t, ok := part.(*mcp.TextContent); ok {
				var b map[string]any
				if json.Unmarshal([]byte(t.Text), &b) == nil {
					canonical, _ := json.Marshal(a)
					legacy, _ := json.Marshal(b)
					compatible = bytes.Equal(canonical, legacy)
				}
			}
		}
		need(compatible, "S5B1 structured/text mismatch "+name)
	}
	fmt.Println("S5B1_STAGE=missing_targets")
	const missingBox = "00000000-0000-4000-8000-000000000001"
	need(rejects(ctx, admin, "box_peer_set", map[string]any{"source_box_id": missingBox, "target_box_id": "00000000-0000-4000-8000-000000000002", "allow": false, "idempotency_key": "s5b1-exact-missing-peer"}), "S5B1 missing peer unexpectedly admitted")
	need(rejects(ctx, admin, "port_forward_create", map[string]any{"box_id": missingBox, "protocol": "tcp", "guest_port": 19091, "host_port": 23093, "host_bind": "127.0.0.1", "idempotency_key": "s5b1-exact-missing-forward"}), "S5B1 missing forward owner unexpectedly admitted")
	need(rejects(ctx, admin, "port_forward_delete", map[string]any{"forward_id": "00000000000000000000000000000001", "idempotency_key": "s5b1-exact-missing-proxy"}), "S5B1 missing forward unexpectedly deleted")
	need(rejects(ctx, admin, "operation_cancel", map[string]any{"target_operation_id": "00000000000000000000000000000001", "idempotency_key": "s5b1-exact-missing-cancel"}), "S5B1 missing cancellation unexpectedly admitted")
	fmt.Println("S5B1_EXACT_23_SCHEMA_CACHE_TYPED_TEXT_AND_REFUSALS=PASS")
}

func s5b1ExactOwned(ctx context.Context, admin *mcp.ClientSession, uuid, createID string) {
	fmt.Println("S5B1_STAGE=readiness")
	ready := call(ctx, admin, "box_wait_ready", map[string]any{"box_id": uuid, "checks": []string{"running", "exec"}, "timeout_seconds": 60, "idempotency_key": "s5b1-exact-owned-ready"})
	readyID := originalID(ready)
	await(ctx, admin, readyID, "SUCCEEDED")
	fmt.Println("S5B1_STAGE=recovery")
	diagnosis := call(ctx, admin, "operation_recover", map[string]any{"operation_id": createID, "mode": "inspect"})
	need(diagnosis["native_retried"] == false, "S5B1 recovery inspect unexpectedly replayed")
	need(field(object(diagnosis, "diagnosis"), "original_operation_id") == createID, "S5B1 recovery original identity mismatch")
	fmt.Println("S5B1_STAGE=activity")
	evidence := call(ctx, admin, "box_activity_get", map[string]any{"view": "evidence", "operation_id": readyID})
	raw, _ := json.Marshal(evidence)
	need(bytes.Contains(raw, []byte(readyID)), "S5B1 Activity original absent")
	// Only a localhost listener within the one-use runner. No external
	// bind, proxy permission expansion, PRS grant or unowned resource.
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	need(e == nil, "S5B1 fresh loopback port unavailable")
	port := listener.Addr().(*net.TCPAddr).Port
	need(listener.Close() == nil, "S5B1 loopback reservation release")
	fwd := call(ctx, admin, "port_forward_create", map[string]any{"box_id": uuid, "protocol": "tcp", "guest_port": 19091, "host_port": port, "host_bind": "127.0.0.1", "idempotency_key": "s5b1-exact-loopback-forward"})
	original := originalID(fwd)
	fmt.Println("S5B1_STAGE=forward_terminal")
	policy := requiredEnv("BW_S5B1_PROXY_POLICY")
	need(policy == "deny" || policy == "allow", "S5B1 missing explicit native proxy policy witness")
	if policy == "deny" {
		op := await(ctx, admin, original, "FAILED")
		status := call(ctx, admin, "host_status", map[string]any{})
		nativeStatus := object(status, "native")
		wantClass := "SAFE_NATIVE_PREFLIGHT_DENIED" // immutable older releases
		if permissions, ok := nativeStatus["project_permissions"].(map[string]any); ok {
			need(permissions["restricted.devices.proxy"] != "allow", "S5B1 reported proxy permission differs from independent host witness")
			wantClass = "PROJECT_PROXY_DENIED"
		}
		need(field(op, "error_class") == wantClass && field(op, "native_operation_id") == "", "S5B1 default proxy refusal lacks exact predispatch proof")
		listed := call(ctx, admin, "port_forward_list", map[string]any{"box_id": uuid})
		items, ok := listed["forwards"].([]any)
		need(ok && len(items) == 0, "S5B1 refused proxy retained live claim")
		fmt.Printf("S5B1_DEFAULT_PROXY_POLICY_REFUSAL=PASS original=%s ready=%s\n", original, readyID)
		fmt.Printf("S5B1_EXACT_READINESS_RECOVERY_ACTIVITY_FORWARD=PASS ready=%s forward_refused=%s\n", readyID, original)
		return
	}
	op := await(ctx, admin, original, "SUCCEEDED")
	need(field(op, "native_operation_id") != "", "S5B1 native forward receipt missing")
	listed := call(ctx, admin, "port_forward_list", map[string]any{"box_id": uuid})
	items, ok := listed["forwards"].([]any)
	need(ok && len(items) == 1, "S5B1 forward inventory mismatch")
	entry, ok := items[0].(map[string]any)
	need(ok && field(entry, "forward_id") == original && field(entry, "state") == "CONFIRMED" && entry["native_verified"] == true, "S5B1 forward exact device not verified")
	del := call(ctx, admin, "port_forward_delete", map[string]any{"forward_id": original, "idempotency_key": "s5b1-exact-loopback-forward-delete"})
	delID := originalID(del)
	removed := await(ctx, admin, delID, "SUCCEEDED")
	need(field(removed, "native_operation_id") != "", "S5B1 native forward delete receipt missing")
	after := call(ctx, admin, "port_forward_list", map[string]any{"box_id": uuid})
	empty, ok := after["forwards"].([]any)
	need(ok && len(empty) == 0, "S5B1 removed forward remains")
	fmt.Printf("S5B1_EXACT_READINESS_RECOVERY_ACTIVITY_FORWARD=PASS ready=%s forward=%s delete=%s\n", readyID, original, delID)
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
	if os.Getenv("BW_S3_PHASE3_MODE") == "handoff" {
		need(os.Getenv("BW_S5B_TC015") == "true", "handoff mode not explicitly enabled")
		actorB := connect(ctx, "s5b-actor-b", requiredEnv("BW_S3_PEER_TOKEN_FILE"))
		defer actorB.Close()
		resumeS5BGitHandoff(ctx, actorB)
		return
	}
	if os.Getenv("BW_S3_PHASE3_MODE") == "verify" {
		independent := connect(ctx, "fresh-restart-verifier", requiredEnv("BW_S3_ADMIN_TOKEN_FILE"))
		defer independent.Close()
		verifyS3GuestGitOriginals(ctx, independent)
		return
	}
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
	peer := connect(ctx, "peer", requiredEnv("BW_S3_PEER_TOKEN_FILE"))
	defer peer.Close()
	t, err := admin.ListTools(ctx, nil)
	if err != nil {
		panic(err)
	}
	names := map[string]bool{}
	for _, tool := range t.Tools {
		names[tool.Name] = true
	}
	for _, n := range []string{"box_file_list", "box_file_read", "box_file_apply", "box_activity_get", "box_peer_set", "operation_recover"} {
		need(names[n], "missing RC8 Tool "+n)
	}
	need(len(names) == 23, "S4-B4 candidate must expose 23 Tools including public Activity and Forward")
	fmt.Println("S3_INSTALLED_23_TOOL_OFFICIAL_SDK_CATALOG=PASS")
	if os.Getenv("BW_S5B1_EXACT_TOOLS") == "true" {
		s5b1ExactCatalog(ctx, admin, t.Tools)
	}
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
	if os.Getenv("BW_S5B1_EXACT_TOOLS") == "true" {
		s5b1ExactOwned(ctx, admin, uuid, createID)
	}
	base := "/root/s3-file-phase2"
	fixture := "mkdir -p " + base + "; " +
		"printf S3_PHASE2_VICTIM_LOCKED > " + base + "/victim; " +
		"ln -s " + base + "/victim " + base + "/sym; " +
		"ln " + base + "/victim " + base + "/hard; " +
		"mkfifo " + base + "/fifo; " +
		"ln -s " + base + " /root/s3-file-phase2-link; " +
		"test -f " + base + "/victim; sync; echo S3_FIXTURE_READY"
	e := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-ec", fixture},
		"idempotency_key": "s3-installed-fixture-one-native-exec-1115",
	})
	fixtureID := originalID(e)
	fixtureReceipt := await(ctx, admin, fixtureID, "SUCCEEDED")
	fixtureNativeID := field(fixtureReceipt, "native_operation_id")
	need(fixtureNativeID != "", "original fixture Exec is missing Incus operation ID")
	fmt.Println("S3_INSTALLED_GUEST_FIXTURE_ORIGINAL_BW_OPERATION_ID=" + fixtureID)
	fmt.Println("S3_INSTALLED_GUEST_FIXTURE_ORIGINAL_NATIVE_OPERATION_ID=" + fixtureNativeID)
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
	need(bytes.Equal(fileBytes(ctx, peer, uuid, base+"/victim"), bytesRead),
		"authorized Viewer read did not return original Guest bytes")
	fmt.Println("S3_INSTALLED_FILE_LIST_PAGINATION_ADMIN_PEER_READ=PASS")
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
	// A second approved Actor may perform an owned guest-file mutation.
	peerBytes := []byte("S3_INDEPENDENT_ACTOR_CREATE_OK")
	peerCreate := call(ctx, peer, "box_file_apply", map[string]any{
		"box_id": uuid, "path": base + "/peer-created", "mode": "create",
		"content_base64":  base64.StdEncoding.EncodeToString(peerBytes),
		"idempotency_key": "s3-peer-file-create-verified-01",
	})
	await(ctx, peer, originalID(peerCreate), "SUCCEEDED")
	need(bytes.Equal(fileBytes(ctx, admin, uuid, base+"/peer-created"), peerBytes),
		"independent authorized Actor's guest-file Create not readable")
	for _, mode := range []string{"replace", "delete"} {
		need(rejects(ctx, admin, "box_file_apply", map[string]any{
			"box_id": uuid, "path": base + "/victim", "mode": mode,
			"idempotency_key": "s3-native-cas-unsupported-" + mode,
		}), "unsafe native conditional "+mode+" was accepted")
	}
	need(bytes.Equal(fileBytes(ctx, admin, uuid, base+"/victim"), bytesRead),
		"unsafe file mode modified original Guest bytes")
	fmt.Println("S3_INSTALLED_PEER_FILE_CREATE_AND_UNSUPPORTED_CAS_DENIAL=PASS")
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
		runS3GuestGitPhase3(ctx, admin, peer, uuid)
		// The same negative and idempotency matrix already passed before
		// staging the supervised Git fixture. Emit its normal CI marker.
		fmt.Println("S3_INSTALLED_REAL_HTTP_MCP_FILE_NEGATIVE_AND_IDEMPOTENCY=PASS")
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

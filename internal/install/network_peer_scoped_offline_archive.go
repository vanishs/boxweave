package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type scopedOfflineArchivedFile struct {
	Name string `json:"name"`
	Info struct {
		SHA string `json:"sha256"`
	} `json:"info"`
}

// The persistent serve-lock is approved and archived alongside SQLite.
// It is not a SQLite sidecar. Preserve and verify its exact bytes too;
// reject any other ledger-prefixed entry, including rollback journals.
func scopedOfflineLedgerDigests(files []scopedOfflineArchivedFile) (map[string]string, error) {
	expected := map[string]string{}
	for _, f := range files {
		if strings.HasPrefix(f.Name, "ledger.sqlite") {
			switch f.Name {
			case "ledger.sqlite", "ledger.sqlite-wal", "ledger.sqlite-shm", "ledger.sqlite.serve.lock":
			default:
				return nil, errors.New("offline archive ledger inventory invalid")
			}
			if expected[f.Name] != "" || !nativePeerGuardSHA256Valid(f.Info.SHA) {
				return nil, errors.New("offline archive ledger inventory invalid")
			}
			expected[f.Name] = f.Info.SHA
		}
	}
	if expected["ledger.sqlite"] == "" || expected["ledger.sqlite.serve.lock"] == "" {
		return nil, errors.New("offline archive original ledger or serve lock missing")
	}
	return expected, nil
}

// The root-private archived review is the sole source of the exceptional
// effect set. No caller-supplied path, UUID, ACL or replacement ledger.
func scopedOfflineArchiveBinding(project, digest string) (ScopedOfflineReview, string, error) {
	var zero ScopedOfflineReview
	if os.Geteuid() != 0 || !nativePeerGuardSHA256Valid(digest) {
		return zero, "", errors.New("exact local archive review required")
	}
	if e := scopedRetirementFenceMatches(project, digest); e != nil {
		return zero, "", e
	}
	b, _, e := diagnosticReadFile(scopedRetirementFence, 0, true, 8192)
	if e != nil {
		return zero, "", e
	}
	var fence struct {
		Archive string `json:"archive"`
		Review  string `json:"review_sha256"`
	}
	if json.Unmarshal(b, &fence) != nil {
		return zero, "", errors.New("offline archive fence invalid")
	}
	raw, _, e := diagnosticReadFile("/var/lib/boxweave-uninstall-v1-audit.json", 0, true, 1024*1024)
	if e != nil {
		return zero, "", e
	}
	var audit struct {
		Status  string `json:"status"`
		Mode    string `json:"mode"`
		Phase   string `json:"phase"`
		Archive string `json:"archive"`
		Review  string `json:"review_sha256"`
	}
	if json.Unmarshal(raw, &audit) != nil || audit.Status != "STARTED" || audit.Mode != "SCOPED_PEER_FRESH_V2" || audit.Archive != fence.Archive || audit.Review != fence.Review ||
		(audit.Phase != "ARCHIVED" && audit.Phase != "NATIVE_RESTORED" && audit.Phase != "GUARD_REMOVED") {
		return zero, "", errors.New("offline original audit phase unverified")
	}

	b, _, e = diagnosticReadFile(filepath.Join(fence.Archive, "review.json"), 0, true, 1024*1024)
	if e != nil {
		return zero, "", e
	}
	var plan struct {
		Project string                      `json:"project"`
		Review  string                      `json:"review_sha256"`
		Offline bool                        `json:"offline_entry"`
		Archive *ScopedOfflineReview        `json:"offline_archive"`
		UID     uint32                      `json:"service_user_uid"`
		Files   []scopedOfflineArchivedFile `json:"state_files"`
	}
	if json.Unmarshal(b, &plan) != nil || !plan.Offline || plan.Project != project || plan.Review != fence.Review || plan.Archive == nil ||
		plan.Archive.Project != project || plan.Archive.ReviewSHA256 != digest || len(plan.Archive.Targets) == 0 || len(plan.Archive.Originals) == 0 {
		return zero, "", errors.New("offline archive reviewed original mismatch")
	}
	u, e := user.Lookup("boxweave")
	if e != nil || u.Uid != strconv.FormatUint(uint64(plan.UID), 10) || plan.UID == 0 || project != "user-"+u.Uid {
		return zero, "", errors.New("offline archive service identity changed")
	}
	expected, e := scopedOfflineLedgerDigests(plan.Files)
	if e != nil {
		return zero, "", e
	}
	for _, name := range []string{"ledger.sqlite", "ledger.sqlite-wal", "ledger.sqlite-shm", "ledger.sqlite-journal", "ledger.sqlite.serve.lock"} {
		src := filepath.Join("/var/lib/boxweave", name)
		dst := filepath.Join(fence.Archive, "state-"+name)
		if expected[name] == "" {
			for _, path := range []string{src, dst} {
				if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
					return zero, "", errors.New("offline archive unexpected ledger sidecar")
				}
			}
			continue
		}
		for _, v := range []struct {
			Path string
			UID  uint32
		}{{src, plan.UID}, {dst, 0}} {
			data, _, e := diagnosticReadFile(v.Path, v.UID, true, 32<<20)
			if e != nil || diagnosticHash(data) != expected[name] {
				return zero, "", errors.New("offline archive original ledger bytes changed")
			}
		}
	}
	return *plan.Archive, fence.Archive, nil
}

type scopedOfflineStepReceipt struct {
	Schema                       int
	Review                       string
	Index                        int
	Kind, UUID, Status, NativeID string
}

func scopedOfflineReceipt(review ScopedOfflineReview, index int, status, nativeID string) scopedOfflineStepReceipt {
	kind := "INSTANCE"
	if index%2 == 1 {
		kind = "ACL"
	}
	return scopedOfflineStepReceipt{1, review.ReviewSHA256, index, kind, review.Targets[index/2].UUID, status, nativeID}
}
func scopedOfflineReceiptName(index int, status string) string {
	return fmt.Sprintf("offline-effect-%04d-%s.json", index, strings.ToLower(status))
}
func scopedOfflineReceiptRead(archive string, expected scopedOfflineStepReceipt) (bool, error) {
	data, _, e := diagnosticReadFile(filepath.Join(archive, scopedOfflineReceiptName(expected.Index, expected.Status)), uint32(os.Geteuid()), true, 8192)
	if errors.Is(e, os.ErrNotExist) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	var actual scopedOfflineStepReceipt
	if json.Unmarshal(data, &actual) != nil {
		return false, errors.New("invalid offline effect receipt")
	}
	if expected.Status == "ACK" {
		if actual.NativeID == "" || !scopedPeerUUID.MatchString(actual.NativeID) {
			return false, errors.New("offline native ACK invalid")
		}
		expected.NativeID = actual.NativeID
	}
	canonical, e := json.Marshal(actual)
	if e != nil || !bytes.Equal(data, append(canonical, '\n')) || actual != expected {
		return false, errors.New("offline effect receipt identity mismatch")
	}
	return true, nil
}
func scopedOfflineReceiptWrite(archive string, r scopedOfflineStepReceipt) error {
	data, e := json.Marshal(r)
	if e != nil {
		return e
	}
	f, e := os.OpenFile(filepath.Join(archive, scopedOfflineReceiptName(r.Index, r.Status)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(append(data, '\n'))
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	d, e := os.Open(archive)
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}

func scopedOfflineJournal(archive string, review ScopedOfflineReview) (int, bool, error) {
	files, e := os.ReadDir(archive)
	if e != nil {
		return 0, false, e
	}
	seen := map[string]bool{}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "offline-effect-") {
			seen[f.Name()] = true
		}
	}
	completed, pending := 0, false
	for i := 0; i < 2*len(review.Targets); i++ {
		if i != completed || pending {
			break
		}
		for _, status := range []string{"INTENT", "ACK", "VERIFIED"} {
			found, e := scopedOfflineReceiptRead(archive, scopedOfflineReceipt(review, i, status, ""))
			if e != nil {
				return 0, false, e
			}
			if !found {
				continue
			}
			delete(seen, scopedOfflineReceiptName(i, status))
			switch status {
			case "INTENT":
				pending = true
			case "ACK":
				if !pending || i%2 != 0 {
					return 0, false, errors.New("offline ACK without instance intent")
				}
			case "VERIFIED":
				if !pending {
					return 0, false, errors.New("offline verification without intent")
				}
				pending = false
				completed++
			}
		}
		if !pending && completed == i {
			break
		}
	}
	if len(seen) != 0 {
		return 0, false, errors.New("offline effect journal not a contiguous prefix")
	}
	return completed, pending, nil
}

// Only an independently observed one-step absence may consume a prior
// intent. An unobserved or ambiguous dispatch is never repeated.
func scopedOfflineNext(completed int, pending bool, observed, total int) (string, error) {
	if completed < 0 || observed < 0 || completed > total || observed > total {
		return "", errors.New("offline prefix bound invalid")
	}
	if pending {
		if completed < total && observed == completed+1 {
			return "VERIFY_ONLY", nil
		}
		return "", errors.New("offline original outcome unverified; preserve intent")
	}
	if observed != completed {
		return "", errors.New("unrecorded native retirement change")
	}
	if completed == total {
		return "COMPLETE", nil
	}
	return "DISPATCH", nil
}

type ScopedOfflineProgress struct {
	ReviewSHA256 string `json:"review_sha256"`
	Completed    int    `json:"completed"`
	Total        int    `json:"total"`
	Next         string `json:"next"`
}

func scopedOfflineObserveArchive(ctx context.Context, project, digest string) (ScopedOfflineReview, string, int, error) {
	var zero ScopedOfflineReview
	approved, archive, e := scopedOfflineArchiveBinding(project, digest)
	if e != nil {
		return zero, "", 0, e
	}
	a, e := scopedOfflineReviewRead(ctx, project, &approved)
	if e != nil {
		return zero, "", 0, e
	}
	b, e := scopedOfflineReviewRead(ctx, project, &approved)
	if e != nil || !reflect.DeepEqual(a, b) {
		return zero, "", 0, errors.New("offline native prefix changed")
	}
	again, path, e := scopedOfflineArchiveBinding(project, digest)
	if e != nil || path != archive || !reflect.DeepEqual(approved, again) {
		return zero, "", 0, errors.New("offline archive changed across observation")
	}
	return approved, archive, b.Prefix, nil
}

func NativeScopedOfflineArchiveProgress(ctx context.Context, project, digest string) (ScopedOfflineProgress, error) {
	var zero ScopedOfflineProgress
	review, archive, prefix, e := scopedOfflineObserveArchive(ctx, project, digest)
	if e != nil {
		return zero, e
	}
	completed, pending, e := scopedOfflineJournal(archive, review)
	if e != nil {
		return zero, e
	}
	next, e := scopedOfflineNext(completed, pending, prefix, 2*len(review.Targets))
	if e != nil {
		return zero, e
	}
	return ScopedOfflineProgress{digest, completed, 2 * len(review.Targets), next}, nil
}

// NativeScopedOfflineArchiveStep is deliberately local and archive-bound.
// It admits no business operation and never changes original ledger rows.
// A prior intent permits only independent absence verification, no replay.
func NativeScopedOfflineArchiveStep(ctx context.Context, project, digest string, approved bool) (ScopedOfflineProgress, error) {
	var zero ScopedOfflineProgress
	if !approved {
		return zero, errors.New("exact local offline retirement approval required")
	}
	review, archive, prefix, e := scopedOfflineObserveArchive(ctx, project, digest)
	if e != nil {
		return zero, e
	}
	completed, pending, e := scopedOfflineJournal(archive, review)
	if e != nil {
		return zero, e
	}
	total := 2 * len(review.Targets)
	next, e := scopedOfflineNext(completed, pending, prefix, total)
	if e != nil {
		return zero, e
	}
	if next == "COMPLETE" {
		return ScopedOfflineProgress{digest, completed, total, next}, nil
	}
	if next == "DISPATCH" {
		if e = scopedOfflineReceiptWrite(archive, scopedOfflineReceipt(review, completed, "INTENT", "")); e != nil {
			return zero, e
		}
		again, path, observed, e := scopedOfflineObserveArchive(ctx, project, digest)
		if e != nil || path != archive || observed != completed || !reflect.DeepEqual(review, again) {
			return zero, errors.New("offline pre-dispatch proof changed; preserve intent")
		}
		c, e := prepareClient(ctx)
		if e != nil {
			return zero, e
		}
		c = c.UseProject("default")
		target := review.Targets[completed/2]
		if completed%2 == 0 {
			own := c.UseProject(project)
			inst, _, e := own.GetInstance(target.Name)
			if e != nil || inst == nil || inst.Project != project || inst.Status != "Stopped" || inst.Config["volatile.uuid"] != target.UUID || nativePrepareStateDigest(inst.Writable(), inst.ExpandedConfig, inst.ExpandedDevices) != target.InstanceSHA256 {
				return zero, errors.New("offline instance pre-dispatch identity changed")
			}
			op, e := own.DeleteInstance(target.Name)
			if e != nil || op == nil || !scopedPeerUUID.MatchString(op.Get().ID) {
				return zero, errors.New("offline instance dispatch outcome unverified; preserve intent")
			}
			if e = scopedOfflineReceiptWrite(archive, scopedOfflineReceipt(review, completed, "ACK", op.Get().ID)); e != nil {
				return zero, e
			}
			if e = op.WaitContext(ctx); e != nil {
				return zero, errors.New("offline instance outcome unverified; preserve intent")
			}
		} else {
			acl, tag, e := c.GetNetworkACL(target.ACL)
			if e != nil || acl == nil || tag == "" || acl.Project != "default" || len(acl.UsedBy) != 0 || nativePrepareStateDigest(acl.Writable()) != target.ACLSHA256 {
				return zero, errors.New("offline ACL pre-dispatch identity changed")
			}
			if e = c.DeleteNetworkACL(target.ACL); e != nil {
				return zero, errors.New("offline ACL outcome unverified; preserve intent")
			}
		}
	}
	again, path, observed, e := scopedOfflineObserveArchive(ctx, project, digest)
	if e != nil || path != archive || observed != completed+1 || !reflect.DeepEqual(review, again) {
		return zero, errors.New("offline native absence not independently verified")
	}
	if e = scopedOfflineReceiptWrite(archive, scopedOfflineReceipt(review, completed, "VERIFIED", "")); e != nil {
		return zero, e
	}
	return NativeScopedOfflineArchiveProgress(ctx, project, digest)
}

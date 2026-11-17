package install

// Root-only staged global restricted-TLS routing. The journal is durable
// before the first native PUT. A transport error is UNKNOWN and is never
// retried; an operator must inspect and explicitly reconcile the original.
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

const nativeScriptletJournalPath = "/var/lib/boxweave-native-scriptlet-v1.json"

type NativeScriptletStatusReport struct {
	Status       string `json:"status"`
	ReviewSHA256 string `json:"review_sha256"`
	SourceSHA256 string `json:"source_sha256"`
	ReadyForPeer bool   `json:"ready_for_peer"`
}

type nativeScriptletJournal struct {
	Schema        int                 `json:"schema"`
	Status        string              `json:"status"`
	Plan          NativeScriptletPlan `json:"plan"`
	BeforeConfig  api.ConfigMap       `json:"before_config"`
	AppliedConfig api.ConfigMap       `json:"applied_config"`
}

func nativeScriptletReport(j nativeScriptletJournal) NativeScriptletStatusReport {
	return NativeScriptletStatusReport{Status: j.Status, ReviewSHA256: j.Plan.ReviewSHA256,
		SourceSHA256: j.Plan.SourceSHA256, ReadyForPeer: false}
}
func nativeScriptletCopyConfig(in api.ConfigMap) api.ConfigMap {
	out := api.ConfigMap{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func nativeScriptletCheckRootOwner(t NativeScriptletTarget) error {
	if os.Geteuid() != 0 {
		return errors.New("native global Scriptlet routing requires host-local root")
	}
	actor, err := user.Lookup("boxweave")
	if err != nil {
		return err
	}
	if t.OwnerProject != "user-"+actor.Uid {
		return errors.New("native Scriptlet owner Project must match the confined local service UID")
	}
	return nil
}
func nativeScriptletSaveJournal(path string, j nativeScriptletJournal) error {
	if j.Schema != 1 || j.Plan.ReviewSHA256 == "" {
		return errors.New("incomplete Scriptlet journal")
	}
	if st, err := os.Lstat(path); err == nil {
		if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
			return errors.New("unsafe Scriptlet journal file")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := atomicWrite(path, b, 0600); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
func nativeScriptletLoadJournal(path string) (nativeScriptletJournal, error) {
	var j nativeScriptletJournal
	st, err := os.Lstat(path)
	if err != nil {
		return j, err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return j, errors.New("unsafe Scriptlet journal type or mode")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return j, err
	}
	if err := json.Unmarshal(b, &j); err != nil {
		return j, err
	}
	if j.Schema != 1 || j.Plan.ReviewSHA256 == "" || j.Plan.SourceSHA256 == "" ||
		j.BeforeConfig == nil || j.AppliedConfig == nil {
		return j, errors.New("unsupported Scriptlet journal schema")
	}
	return j, nil
}
func NativeScriptletStatus() (NativeScriptletStatusReport, error) {
	if os.Geteuid() != 0 {
		return NativeScriptletStatusReport{}, errors.New("root-only Scriptlet status")
	}
	j, err := nativeScriptletLoadJournal(nativeScriptletJournalPath)
	if err != nil {
		return NativeScriptletStatusReport{}, err
	}
	return nativeScriptletReport(j), nil
}

func NativeScriptletApply(ctx context.Context, t NativeScriptletTarget, approvedSHA string, approved bool) (NativeScriptletStatusReport, error) {
	if err := nativeScriptletCheckRootOwner(t); err != nil {
		return NativeScriptletStatusReport{}, err
	}
	host, err := migrationHostDigest()
	if err != nil {
		return NativeScriptletStatusReport{}, err
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return NativeScriptletStatusReport{}, err
	}
	return nativeScriptletApplyWithClient(ctx, root, t, host, nativeScriptletJournalPath, approvedSHA, approved)
}

func nativeScriptletApplyWithClient(ctx context.Context, root incus.InstanceServer, t NativeScriptletTarget, host, path, approvedSHA string, approved bool) (NativeScriptletStatusReport, error) {
	var zero NativeScriptletStatusReport
	if !approved || !scriptletFingerprint.MatchString(approvedSHA) {
		return zero, errors.New("reviewed SHA and explicit local Scriptlet approval required")
	}
	if _, err := os.Lstat(path); err == nil {
		return zero, errors.New("existing Scriptlet journal: inspect or rollback, never redispatch")
	} else if !os.IsNotExist(err) {
		return zero, err
	}
	plan, err := nativeScriptletInspectWithClient(ctx, root, t, host)
	if err != nil {
		return zero, err
	}
	if plan.ReviewSHA256 != approvedSHA {
		return zero, errors.New("approved Scriptlet plan differs from fresh native inventory")
	}
	// Final ETag and full trust/ACL inventory must still match the reviewed
	// snapshot just before a persistent intent. No claim of global transaction.
	server, etag, err := root.GetServer()
	if err != nil {
		return zero, err
	}
	certs, err := root.GetCertificates()
	if err != nil {
		return zero, err
	}
	acl, aclETag, err := root.GetNetworkACL(t.ACL)
	if err != nil {
		return zero, err
	}
	owner, _, err := root.UseProject(t.OwnerProject).GetProject(t.OwnerProject)
	if err != nil {
		return zero, err
	}
	fresh, err := nativeScriptletPlanFromSnapshot(t, host, server, etag, certs, acl, aclETag, owner)
	if err != nil {
		return zero, err
	}
	if fresh.ReviewSHA256 != approvedSHA {
		return zero, errors.New("native certificate or Server inventory changed before dispatch")
	}
	source, err := NativeScriptletPolicy(t)
	if err != nil {
		return zero, err
	}
	before := nativeScriptletCopyConfig(server.Config)
	applied := nativeScriptletCopyConfig(before)
	applied["authorization.scriptlet"] = source
	applied["authorization.client.tls-restricted"] = "scriptlet"
	j := nativeScriptletJournal{Schema: 1, Status: "DISPATCH_UNKNOWN", Plan: plan,
		BeforeConfig: before, AppliedConfig: applied}
	if err := nativeScriptletSaveJournal(path, j); err != nil {
		return zero, err
	}
	// Incus server PUT is a native global mutation. Its failure may still
	// leave partially applied authorization; do not automatic retry/restore.
	if err := root.UpdateServer(api.ServerPut{Config: applied}, etag); err != nil {
		return nativeScriptletReport(j), fmt.Errorf("original Scriptlet native PUT uncertain; root-only reconciliation required: %w", err)
	}
	actual, _, err := root.GetServer()
	if err != nil {
		return nativeScriptletReport(j), fmt.Errorf("native Scriptlet postread uncertain: %w", err)
	}
	if !reflect.DeepEqual(actual.Config, applied) {
		return nativeScriptletReport(j), errors.New("native Scriptlet postread differed; journal held UNKNOWN")
	}
	j.Status = "ROUTE_APPLIED_PARITY_PENDING"
	if err := nativeScriptletSaveJournal(path, j); err != nil {
		return nativeScriptletReport(j), fmt.Errorf("Scriptlet routing applied but journal final status uncertain: %w", err)
	}
	return nativeScriptletReport(j), nil
}

// Explicit root-only rollback refuses any server config drift or partial
// native state. There is no blind redispatch, host-wide reset, or claim that
// restored routing closes an in-flight Exec/ACL kernel operation.
func NativeScriptletRollback(ctx context.Context, approvedSHA string, approved bool) (NativeScriptletStatusReport, error) {
	if os.Geteuid() != 0 {
		return NativeScriptletStatusReport{}, errors.New("native Scriptlet rollback requires host root")
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return NativeScriptletStatusReport{}, err
	}
	return nativeScriptletRollbackWithClient(ctx, root, nativeScriptletJournalPath, approvedSHA, approved)
}
func nativeScriptletRollbackWithClient(ctx context.Context, root incus.InstanceServer, path, approvedSHA string, approved bool) (NativeScriptletStatusReport, error) {
	var zero NativeScriptletStatusReport
	if !approved || !scriptletFingerprint.MatchString(approvedSHA) {
		return zero, errors.New("explicit exact rollback approval required")
	}
	j, err := nativeScriptletLoadJournal(path)
	if err != nil {
		return zero, err
	}
	if j.Plan.ReviewSHA256 != approvedSHA {
		return zero, errors.New("rollback review SHA differs from recorded original")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	actual, etag, err := root.GetServer()
	if err != nil {
		return zero, err
	}
	if !reflect.DeepEqual(actual.Config, j.BeforeConfig) && !reflect.DeepEqual(actual.Config, j.AppliedConfig) {
		return nativeScriptletReport(j), errors.New("native server authorization drift or partial route; refuse unsafe rollback")
	}
	if reflect.DeepEqual(actual.Config, j.BeforeConfig) {
		j.Status = "ROLLED_BACK"
		if err := nativeScriptletSaveJournal(path, j); err != nil {
			return nativeScriptletReport(j), err
		}
		return nativeScriptletReport(j), nil
	}
	j.Status = "ROLLBACK_UNKNOWN"
	if err := nativeScriptletSaveJournal(path, j); err != nil {
		return nativeScriptletReport(j), err
	}
	if err := root.UpdateServer(api.ServerPut{Config: j.BeforeConfig}, etag); err != nil {
		return nativeScriptletReport(j), fmt.Errorf("native Scriptlet rollback effect uncertain: %w", err)
	}
	check, _, err := root.GetServer()
	if err != nil {
		return nativeScriptletReport(j), err
	}
	if !reflect.DeepEqual(check.Config, j.BeforeConfig) {
		return nativeScriptletReport(j), errors.New("native Scriptlet rollback postread disagrees; manual root reconciliation required")
	}
	j.Status = "ROLLED_BACK"
	if err := nativeScriptletSaveJournal(path, j); err != nil {
		return nativeScriptletReport(j), err
	}
	return nativeScriptletReport(j), nil
}

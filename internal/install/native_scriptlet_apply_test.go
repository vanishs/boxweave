package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

type fakeScriptletRoot struct {
	incus.InstanceServer
	server      *api.Server
	certs       []api.Certificate
	acl         *api.NetworkACL
	owner       *api.Project
	version     int
	updateCount int
	failMode    string
}

func (f *fakeScriptletRoot) GetServer() (*api.Server, string, error) {
	s := &api.Server{}
	s.Config = nativeScriptletCopyConfig(f.server.Config)
	return s, fmtScriptletVersion(f.version), nil
}
func fmtScriptletVersion(v int) string {
	if v == 0 {
		return "server-etag-0"
	}
	return "server-etag-1"
}
func (f *fakeScriptletRoot) GetCertificates() ([]api.Certificate, error) { return f.certs, nil }
func (f *fakeScriptletRoot) GetNetworkACL(string) (*api.NetworkACL, string, error) {
	return f.acl, "acl-etag", nil
}
func (f *fakeScriptletRoot) GetProject(string) (*api.Project, string, error) {
	return f.owner, "project-etag", nil
}
func (f *fakeScriptletRoot) UseProject(string) incus.InstanceServer { return f }
func (f *fakeScriptletRoot) UpdateServer(v api.ServerPut, etag string) error {
	f.updateCount++
	if etag != fmtScriptletVersion(f.version) {
		return errors.New("stale native ETag")
	}
	if f.failMode == "before_dispatch" {
		f.failMode = ""
		return errors.New("native before-send failure")
	}
	next := nativeScriptletCopyConfig(v.Config)
	if f.failMode == "partial_after_dispatch" {
		next = nativeScriptletCopyConfig(f.server.Config)
		next["authorization.scriptlet"] = v.Config["authorization.scriptlet"]
	}
	f.server.Config = next
	f.version++
	if f.failMode == "lost_ack_after_dispatch" || f.failMode == "partial_after_dispatch" {
		f.failMode = ""
		return errors.New("native write ACK lost")
	}
	return nil
}

func fakeScriptletFixture() *fakeScriptletRoot {
	_, s, cs, acl, owner := scriptletTestFixture()
	return &fakeScriptletRoot{server: s, certs: cs, acl: acl, owner: owner}
}

func TestS4B2RootScriptletDurableApplyRollbackExactOriginal(t *testing.T) {
	f := fakeScriptletFixture()
	target, _, _, _, _ := scriptletTestFixture()
	plan, err := nativeScriptletInspectWithClient(t.Context(), f, target, "host-id")
	if err != nil {
		t.Fatal(err)
	}
	old := nativeScriptletCopyConfig(f.server.Config)
	path := filepath.Join(t.TempDir(), "journal.json")
	result, err := nativeScriptletApplyWithClient(t.Context(), f, target, "host-id", path, plan.ReviewSHA256, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ROUTE_APPLIED_PARITY_PENDING" || result.ReadyForPeer ||
		f.server.Config["authorization.client.tls-restricted"] != "scriptlet" ||
		f.server.Config["authorization.scriptlet"] == "" || f.updateCount != 1 {
		t.Fatalf("invalid native effect proof: %+v writes=%d", result, f.updateCount)
	}
	st, err := os.Stat(path)
	if err != nil || st.Mode().Perm() != 0600 {
		t.Fatal("root only persistent journal not protected", err)
	}
	if _, err := nativeScriptletApplyWithClient(t.Context(), f, target, "host-id", path, plan.ReviewSHA256, true); err == nil {
		t.Fatal("original native effect redispatched")
	}
	if f.updateCount != 1 {
		t.Fatal("duplicate applied journal replay")
	}
	rolled, err := nativeScriptletRollbackWithClient(t.Context(), f, path, plan.ReviewSHA256, true)
	if err != nil {
		t.Fatal(err)
	}
	if rolled.Status != "ROLLED_BACK" || rolled.ReadyForPeer || !reflect.DeepEqual(old, f.server.Config) || f.updateCount != 2 {
		t.Fatalf("native rollback did not restore exact original: %+v", rolled)
	}
	again, err := nativeScriptletRollbackWithClient(t.Context(), f, path, plan.ReviewSHA256, true)
	if err != nil || again.Status != "ROLLED_BACK" || f.updateCount != 2 {
		t.Fatal("repeated rollback must not dispatch", err)
	}
}

func TestS4B2RootScriptletLostAckRecoveryAndPartialRefusal(t *testing.T) {
	for _, mode := range []string{"before_dispatch", "lost_ack_after_dispatch", "partial_after_dispatch"} {
		t.Run(mode, func(t *testing.T) {
			target, _, _, _, _ := scriptletTestFixture()
			f := fakeScriptletFixture()
			plan, err := nativeScriptletInspectWithClient(t.Context(), f, target, "host")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "journal.json")
			f.failMode = mode
			result, err := nativeScriptletApplyWithClient(t.Context(), f, target, "host", path, plan.ReviewSHA256, true)
			if err == nil || result.Status != "DISPATCH_UNKNOWN" || result.ReadyForPeer {
				t.Fatalf("uncertain native ACL scriptlet effect falsely classified: %+v, %v", result, err)
			}
			saved, err := nativeScriptletLoadJournal(path)
			if err != nil || saved.Status != "DISPATCH_UNKNOWN" {
				t.Fatal("missing durable original receipt", err)
			}
			_, err = nativeScriptletApplyWithClient(t.Context(), f, target, "host", path, plan.ReviewSHA256, true)
			if err == nil || f.updateCount != 1 {
				t.Fatal("blind redispatch after uncertain native effect")
			}
			status, err := nativeScriptletRollbackWithClient(t.Context(), f, path, plan.ReviewSHA256, true)
			if mode == "partial_after_dispatch" {
				if err == nil || status.Status != "DISPATCH_UNKNOWN" {
					t.Fatal("unsafe partial global route rollback accepted")
				}
			} else {
				if err != nil || status.Status != "ROLLED_BACK" {
					t.Fatal("explicit original-safe rollback failed", err)
				}
				if !reflect.DeepEqual(f.server.Config, saved.BeforeConfig) {
					t.Fatal("rolled-back global auth config drift")
				}
			}
		})
	}
}

func TestS4B2ScriptletMutationRequiresApprovedOriginalDigest(t *testing.T) {
	target, _, _, _, _ := scriptletTestFixture()
	f := fakeScriptletFixture()
	p, err := nativeScriptletInspectWithClient(context.Background(), f, target, "host")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		sha      string
		approved bool
	}{
		{"", true}, {p.ReviewSHA256, false}, {"deadbeef", true},
	} {
		path := filepath.Join(t.TempDir(), "journal.json")
		if _, err := nativeScriptletApplyWithClient(t.Context(), f, target, "host", path, tt.sha, tt.approved); err == nil {
			t.Fatal("unauthorized native Scriptlet route applied")
		}
		if f.updateCount != 0 {
			t.Fatal("unapproved native PUT dispatched")
		}
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatal("unauthorized request left journal", err)
		}
	}
}

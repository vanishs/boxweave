package install

import (
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func TestNativeRetirementSnapshot(t *testing.T) {
	rec := NativePrepareRecord{Status: "APPLIED",
		Plan: NativePreparePlan{Target: NativePrepareTarget{Project: "user-1001", ACL: "bw-test"}},
		BeforeProject: api.ProjectPut{Config: api.ConfigMap{"restricted": "true"}},
		AfterProject: api.ProjectPut{Config: api.ConfigMap{"restricted": "true", "restricted.devices.proxy": "allow"}},
		BeforeProfile: api.ProfilePut{Config: api.ConfigMap{}},
		AfterProfile: api.ProfilePut{Config: api.ConfigMap{"user.example": "reviewed"}},
		ACL: api.NetworkACLsPost{Name: "bw-test", NetworkACLPut: api.NetworkACLPut{Description: "owned"}},
	}
	acl := &api.NetworkACL{Name: "bw-test", NetworkACLPut: rec.ACL.NetworkACLPut,
		UsedBy: []string{"/1.0/profiles/default?project=user-1001", "/1.0/instances/a?project=user-1001"}}
	digest, err := nativeRetirementSnapshot(rec, rec.AfterProject, rec.AfterProfile, acl)
	if err != nil || !validRetirementReviewSHA(digest) {
		t.Fatal(digest, err)
	}
	acl.UsedBy = acl.UsedBy[:1]
	if after, e := nativeRetirementSnapshot(rec, rec.AfterProject, rec.AfterProfile, acl); e != nil || after != digest {
		t.Fatal("owned instance removal should not change configuration observation", after, e)
	}
	drift := rec.AfterProject
	drift.Config = api.ConfigMap{"restricted": "true", "restricted.devices.proxy": "block", "user.foreign": "true"}
	if _, e := nativeRetirementSnapshot(rec, drift, rec.AfterProfile, acl); e == nil {
		t.Fatal("third Project state accepted")
	}
	if _, e := nativeRetirementSnapshot(rec, rec.AfterProject, api.ProfilePut{Description: "foreign"}, acl); e == nil {
		t.Fatal("third Profile state accepted")
	}
	acl.UsedBy = []string{"/1.0/profiles/default?project=user-2000"}
	if _, e := nativeRetirementSnapshot(rec, rec.AfterProject, rec.AfterProfile, acl); e == nil {
		t.Fatal("foreign ACL reference accepted")
	}
	if _, e := nativeRetirementSnapshot(rec, rec.AfterProject, rec.AfterProfile, nil); e == nil {
		t.Fatal("missing APPLIED ACL accepted")
	}
	rec.Status = "ROLLED_BACK"
	if _, e := nativeRetirementSnapshot(rec, rec.BeforeProject, rec.BeforeProfile, nil); e != nil {
		t.Fatal(e)
	}
	if _, e := nativeRetirementSnapshot(rec, rec.BeforeProject, rec.BeforeProfile, acl); e == nil {
		t.Fatal("rolled back ACL still exists")
	}
}

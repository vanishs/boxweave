package install

import (
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func scriptletTestFixture() (NativeScriptletTarget, *api.Server, []api.Certificate, *api.NetworkACL, *api.Project) {
	t := NativeScriptletTarget{OwnerProject: "user-1001", ACLProject: "default", ACL: "bw-exclusive", WriterFingerprint: strings.Repeat("a", 64)}
	s := &api.Server{}
	s.Config = api.ConfigMap{"core.https_address": "127.0.0.1:8443"}
	certs := []api.Certificate{
		{Fingerprint: strings.Repeat("b", 64), CertificatePut: api.CertificatePut{Type: "client", Restricted: true, Projects: []string{"user-1001"}, Name: "legacy"}},
		{Fingerprint: t.WriterFingerprint, CertificatePut: api.CertificatePut{Type: "client", Restricted: true, Projects: []string{"default"}, Name: "bw-acl-writer"}},
		{Fingerprint: strings.Repeat("c", 64), CertificatePut: api.CertificatePut{Type: "client", Restricted: true, Projects: []string{"other"}, Name: "foreign"}},
	}
	acl := &api.NetworkACL{}
	acl.Name = t.ACL
	project := &api.Project{}
	project.Name = t.OwnerProject
	project.Config = api.ConfigMap{"restricted": "true"}
	return t, s, certs, acl, project
}

func TestS4B2ScriptletPreflightPlanStableAndNoLegacyCredentialLeak(t *testing.T) {
	target, server, certs, acl, owner := scriptletTestFixture()
	p, e := nativeScriptletPlanFromSnapshot(target, "trusted-host", server, "server-etag", certs, acl, "acl-etag", owner)
	if e != nil {
		t.Fatal(e)
	}
	if p.RestrictedClientCount != 3 || len(p.ReviewSHA256) != 64 || len(p.NativeStateSHA256) != 64 || len(p.SourceSHA256) != 64 {
		t.Fatalf("incomplete native root plan: %+v", p)
	}
	source, e := NativeScriptletPolicy(target)
	if e != nil {
		t.Fatal(e)
	}
	for _, needle := range []string{
		"details.Certificate.projects", "details.IsAllProjectsRequest", "network_acl:default/bw-exclusive",
	} {
		if !strings.Contains(source, needle) {
			t.Errorf("policy missing native permission invariant: %q", needle)
		}
	}
	before := p.ReviewSHA256
	certs[0], certs[2] = certs[2], certs[0]
	after, e := nativeScriptletPlanFromSnapshot(target, "trusted-host", server, "server-etag", certs, acl, "acl-etag", owner)
	if e != nil || before != after.ReviewSHA256 {
		t.Fatalf("trust order changed plan identity: %v", e)
	}
	// Cert PEM is excluded; even a server secret is hashed, not reflected.
	server.Config["core.proxy_http"] = "http://secret-should-not-appear"
	changed, e := nativeScriptletPlanFromSnapshot(target, "trusted-host", server, "server-etag", certs, acl, "acl-etag", owner)
	if e != nil || changed.ReviewSHA256 == before {
		t.Fatal("native config drift was not bound")
	}
	if strings.Contains(changed.ReviewSHA256, "secret-should-not-appear") {
		t.Fatal("secret leaked")
	}
}

func TestS4B2ScriptletPlanRefusesUnsafeInstallerTakeover(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(*NativeScriptletTarget, *api.Server, *[]api.Certificate, *api.NetworkACL, *api.Project)
	}{
		{"route_allow", func(_ *NativeScriptletTarget, s *api.Server, _ *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			s.Config["authorization.client.tls-restricted"] = "allow"
		}},
		{"route_openfga", func(_ *NativeScriptletTarget, s *api.Server, _ *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			s.Config["authorization.client.tls-restricted"] = "openfga"
		}},
		{"existing_scriptlet", func(_ *NativeScriptletTarget, s *api.Server, _ *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			s.Config["authorization.scriptlet"] = "def authorize(*args):\n    return True\n"
		}},
		{"other_client_routed_scriptlet", func(_ *NativeScriptletTarget, s *api.Server, _ *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			s.Config["authorization.client.oidc"] = "scriptlet"
		}},
		{"wrong_acl", func(_ *NativeScriptletTarget, _ *api.Server, _ *[]api.Certificate, a *api.NetworkACL, _ *api.Project) {
			a.Name = "foreign"
		}},
		{"unrestricted_project", func(_ *NativeScriptletTarget, _ *api.Server, _ *[]api.Certificate, _ *api.NetworkACL, p *api.Project) {
			p.Config["restricted"] = "false"
		}},
		{"missing_writer", func(_ *NativeScriptletTarget, _ *api.Server, cs *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			*cs = (*cs)[:1]
		}},
		{"writer_unrestricted", func(_ *NativeScriptletTarget, _ *api.Server, cs *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			(*cs)[1].Restricted = false
		}},
		{"writer_extra_project", func(_ *NativeScriptletTarget, _ *api.Server, cs *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			(*cs)[1].Projects = []string{"default", "other"}
		}},
		{"legacy_no_projects", func(_ *NativeScriptletTarget, _ *api.Server, cs *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			(*cs)[0].Projects = nil
		}},
		{"legacy_duplicate_project", func(_ *NativeScriptletTarget, _ *api.Server, cs *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			(*cs)[0].Projects = []string{"user-1001", "user-1001"}
		}},
		{"duplicate_fingerprint", func(_ *NativeScriptletTarget, _ *api.Server, cs *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			(*cs)[2].Fingerprint = (*cs)[0].Fingerprint
		}},
		{"invalid_target_acl", func(t *NativeScriptletTarget, _ *api.Server, _ *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			t.ACL = "unsafe-acl"
		}},
		{"invalid_writer", func(t *NativeScriptletTarget, _ *api.Server, _ *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			t.WriterFingerprint = "not-a-fingerprint"
		}},
		{"wrong_default_project", func(t *NativeScriptletTarget, _ *api.Server, _ *[]api.Certificate, _ *api.NetworkACL, _ *api.Project) {
			t.ACLProject = "other"
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			target, server, certs, acl, project := scriptletTestFixture()
			tt.mutate(&target, server, &certs, acl, project)
			if _, err := nativeScriptletPlanFromSnapshot(target, "trusted-host", server, "server-etag", certs, acl, "acl-etag", project); err == nil {
				t.Fatal("unreviewable global Scriptlet authorization change admitted")
			}
		})
	}
}

func TestS4B2ScriptletPlannerRefusesMissingNativeReceipts(t *testing.T) {
	t0, s, cs, a, p := scriptletTestFixture()
	for _, tt := range []struct{ label, host, serverTag, aclTag string }{
		{"host", "", "etag", "acl"}, {"server_etag", "host", "", "acl"}, {"acl_etag", "host", "etag", ""},
	} {
		t.Run(tt.label, func(t *testing.T) {
			if _, err := nativeScriptletPlanFromSnapshot(t0, tt.host, s, tt.serverTag, cs, a, tt.aclTag, p); err == nil {
				t.Fatal("missing native root evidence admitted")
			}
		})
	}
}

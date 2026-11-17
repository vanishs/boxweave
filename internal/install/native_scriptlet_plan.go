package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// NativeScriptletPlan is an immutable preview. It neither installs the
// source nor changes global restricted-TLS routing. Exact permission parity,
// preexisting client regressions and journaled rollback remain mandatory.
type NativeScriptletPlan struct {
	Target                     NativeScriptletTarget `json:"target"`
	HostSHA256                 string                `json:"host_sha256"`
	NativeStateSHA256          string                `json:"native_state_sha256"`
	SourceSHA256               string                `json:"source_sha256"`
	ReviewSHA256               string                `json:"review_sha256"`
	RestrictedClientCount      int                   `json:"restricted_client_count"`
	OriginalRestrictedTLSRoute string                `json:"original_restricted_tls_route"`
	Warnings                   []string              `json:"warnings"`
}

type nativeScriptletCertScope struct {
	Fingerprint string
	Name        string
	Type        string
	Restricted  bool
	Projects    []string
}

func nativeScriptletDigest(p NativeScriptletPlan) string {
	p.ReviewSHA256 = ""
	b, _ := json.Marshal(p)
	sum := sha256.Sum256(append([]byte("s4b2-native-scriptlet-review-v1\n"), b...))
	return hex.EncodeToString(sum[:])
}

func nativeScriptletPlanFromSnapshot(t NativeScriptletTarget, host string, server *api.Server, serverETag string, certs []api.Certificate, acl *api.NetworkACL, aclETag string, owner *api.Project) (NativeScriptletPlan, error) {
	var p NativeScriptletPlan
	source, err := NativeScriptletPolicy(t)
	if err != nil {
		return p, err
	}
	if server == nil || acl == nil || owner == nil || host == "" || serverETag == "" || aclETag == "" {
		return p, errors.New("native Scriptlet plan requires complete independent root snapshot")
	}
	if acl.Name != t.ACL || owner.Name != t.OwnerProject || owner.Config["restricted"] != "true" {
		return p, errors.New("Scriptlet cannot adopt foreign ACL or unrestricted owner Project")
	}
	original := server.Config["authorization.client.tls-restricted"]
	if original != "" && original != "tls" {
		return p, errors.New("existing restricted TLS routing is not builtin TLS; refuse takeover")
	}
	if server.Config["authorization.scriptlet"] != "" {
		return p, errors.New("existing authorization Scriptlet cannot be silently replaced")
	}
	for k, v := range server.Config {
		if strings.HasPrefix(k, "authorization.client.") && v == "scriptlet" {
			return p, errors.New("existing other-client Scriptlet routing cannot be safely replaced")
		}
	}
	if len(certs) > 1000 {
		return p, errors.New("oversized native trust inventory")
	}
	scopes := make([]nativeScriptletCertScope, 0, len(certs))
	found := 0
	seen := map[string]bool{}
	for _, c := range certs {
		if !scriptletFingerprint.MatchString(c.Fingerprint) || seen[c.Fingerprint] {
			return p, errors.New("ambiguous certificate fingerprint in root trust inventory")
		}
		seen[c.Fingerprint] = true
		if c.Type != "client" {
			if c.Fingerprint == t.WriterFingerprint {
				return p, errors.New("ACL writer is not a client certificate")
			}
			continue
		}
		scope := nativeScriptletCertScope{Fingerprint: c.Fingerprint, Name: c.Name, Type: c.Type, Restricted: c.Restricted,
			Projects: append([]string(nil), c.Projects...)}
		if c.Restricted {
			if len(c.Projects) == 0 {
				return p, errors.New("existing restricted certificate has empty Project scope; reject ambiguous parity")
			}
			dedup := map[string]bool{}
			for _, project := range c.Projects {
				if !safePrereqName.MatchString(project) || dedup[project] {
					return p, errors.New("unsafe or duplicated existing certificate Project scope")
				}
				dedup[project] = true
			}
			p.RestrictedClientCount++
		}
		if c.Fingerprint == t.WriterFingerprint {
			if !c.Restricted || len(c.Projects) != 1 || c.Projects[0] != t.ACLProject {
				return p, errors.New("dedicated ACL writer lacks exact one-Project restricted certificate")
			}
			found++
		}
		slices.Sort(scope.Projects)
		scopes = append(scopes, scope)
	}
	if found != 1 {
		return p, errors.New("dedicated restricted TLS ACL writer certificate not found")
	}
	slices.SortFunc(scopes, func(a, b nativeScriptletCertScope) int { return strings.Compare(a.Fingerprint, b.Fingerprint) })
	// Hash the exact server state in memory; do not persist server secrets,
	// PEM certificates or raw authorization source in the public review.
	state, _ := json.Marshal(struct {
		ServerConfig api.ConfigMap
		ServerETag   string
		Certificates []nativeScriptletCertScope
		ACL          *api.NetworkACL
		ACLETag      string
		Owner        *api.Project
	}{server.Config, serverETag, scopes, acl, aclETag, owner})
	sum := sha256.Sum256(append([]byte("s4b2-native-scriptlet-preflight-v1\n"), state...))
	p = NativeScriptletPlan{Target: t, HostSHA256: host,
		NativeStateSHA256: hex.EncodeToString(sum[:]), SourceSHA256: nativeScriptletSourceDigest(source),
		RestrictedClientCount: p.RestrictedClientCount, OriginalRestrictedTLSRoute: original,
		Warnings: []string{
			"The restricted TLS Scriptlet route changes the entire Incus daemon client class",
			"Source parity tests do not authorize installation or constitute complete inherited-rights equivalence",
			"Apply requires root-local approval, durable rollback journal, original state recheck, and client matrix; not yet enabled",
		}}
	p.ReviewSHA256 = nativeScriptletDigest(p)
	return p, nil
}

// NativeScriptletPlanReview is root-local only. It makes no native writes,
// creates no TLS credentials, and does not expose the root Incus socket.
func NativeScriptletPlanReview(ctx context.Context, t NativeScriptletTarget) (NativeScriptletPlan, error) {
	var p NativeScriptletPlan
	if os.Geteuid() != 0 {
		return p, errors.New("Scriptlet review requires host-local root")
	}
	host, err := migrationHostDigest()
	if err != nil {
		return p, err
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return p, err
	}
	return nativeScriptletInspectWithClient(ctx, root, t, host)
}

func nativeScriptletInspectWithClient(ctx context.Context, root incus.InstanceServer, t NativeScriptletTarget, host string) (NativeScriptletPlan, error) {
	var p NativeScriptletPlan
	if err := ctx.Err(); err != nil {
		return p, err
	}
	if root == nil {
		return p, errors.New("missing native root client")
	}
	server, etag, err := root.GetServer()
	if err != nil {
		return p, err
	}
	certs, err := root.GetCertificates()
	if err != nil {
		return p, err
	}
	acl, aETag, err := root.GetNetworkACL(t.ACL)
	if err != nil {
		return p, err
	}
	owner, _, err := root.UseProject(t.OwnerProject).GetProject(t.OwnerProject)
	if err != nil {
		return p, err
	}
	plan, err := nativeScriptletPlanFromSnapshot(t, host, server, etag, certs, acl, aETag, owner)
	if err != nil {
		return p, fmt.Errorf("native Scriptlet preflight refused: %w", err)
	}
	// A second inventory read detects ordinary concurrent certificate churn.
	// A privileged out-of-band writer can still race after plan creation.
	again, e2, err := root.GetServer()
	if err != nil {
		return p, err
	}
	certs2, err := root.GetCertificates()
	if err != nil {
		return p, err
	}
	acl2, at2, err := root.GetNetworkACL(t.ACL)
	if err != nil {
		return p, err
	}
	owner2, _, err := root.UseProject(t.OwnerProject).GetProject(t.OwnerProject)
	if err != nil {
		return p, err
	}
	plan2, err := nativeScriptletPlanFromSnapshot(t, host, again, e2, certs2, acl2, at2, owner2)
	if err != nil {
		return p, err
	}
	if plan.NativeStateSHA256 != plan2.NativeStateSHA256 || plan.ReviewSHA256 != plan2.ReviewSHA256 {
		return p, errors.New("concurrent root Scriptlet/certificate/ACL inventory change")
	}
	return plan, nil
}

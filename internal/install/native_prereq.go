package install

// This is an operator-only, read-only review of S2's native dependencies.
// It does not initialize Incus, adopt foreign resources or constitute proof
// of ACL packet enforcement. A reviewed prerequisite can be operator-prepared
// rather than silently modified by the MCP service.
import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"regexp"
	"slices"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

type NativePrereqCheck struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}
type NativePrereqReport struct {
	Project                string              `json:"project"`
	HostSHA256             string              `json:"host_sha256"`
	Pool                   string              `json:"selected_pool,omitempty"`
	Network                string              `json:"selected_network,omitempty"`
	ACL                    string              `json:"selected_acl,omitempty"`
	ImageFingerprint       string              `json:"approved_image_fingerprint,omitempty"`
	NativeStateSHA256      string              `json:"native_state_sha256"`
	Checks                 []NativePrereqCheck `json:"checks"`
	ConfigReady            bool                `json:"config_ready"`
	ReviewSHA256           string              `json:"review_sha256"`
	PacketWitnessRequired  bool                `json:"packet_witness_required"`
	QuotaMountPrecondition bool                `json:"selected_storage_quota_mount_precondition"`
	Warnings               []string            `json:"warnings,omitempty"`
}

var safePrereqName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

func prereqRecord(r *NativePrereqReport, id string, ok bool, detail string) {
	r.Checks = append(r.Checks, NativePrereqCheck{ID: id, OK: ok, Detail: detail})
}

// Refuse ACLs used in another project or by a shared network. Native ACL
// used_by is not a reliable packet witness; this is only an exclusivity guard.
func aclExclusiveUse(usedBy []string, project string) bool {
	if len(usedBy) == 0 {
		return false
	}
	profile := false
	for _, reference := range usedBy {
		u, err := url.Parse(reference)
		if err != nil || !strings.HasPrefix(u.Path, "/1.0/") || u.Query().Get("project") != project ||
			len(u.Query()) != 1 || u.Fragment != "" || u.Host != "" {
			return false
		}
		if !strings.HasPrefix(u.Path, "/1.0/profiles/") && !strings.HasPrefix(u.Path, "/1.0/instances/") {
			return false
		}
		if u.Path == "/1.0/profiles/default" {
			profile = true
		}
	}
	return profile
}

func prereqDevices(devices api.DevicesMap) (pool, network, acl string, ok bool) {
	rootCount, nicCount := 0, 0
	for _, d := range devices {
		switch d["type"] {
		case "disk":
			if d["path"] != "/" || d["source"] != "" {
				return "", "", "", false
			}
			rootCount++
			pool = d["pool"]
		case "nic":
			nicCount++
			network, acl = d["network"], d["security.acls"]
			if strings.Contains(acl, ",") || acl == "" ||
				d["security.acls.default.ingress.action"] != "reject" ||
				d["security.ipv4_filtering"] != "true" ||
				d["security.ipv6_filtering"] != "true" ||
				d["security.mac_filtering"] != "true" {
				return "", "", "", false
			}
		default:
			return "", "", "", false // no unreviewed inherited disk, proxy or host device
		}
	}
	return pool, network, acl, rootCount == 1 && nicCount == 1 &&
		safePrereqName.MatchString(pool) && safePrereqName.MatchString(network) && safePrereqName.MatchString(acl)
}

func prereqDigest(r NativePrereqReport) string {
	// Do not hash volatile free-space counters or status text: the operator
	// approves exact native object identities and a stable required-check set.
	type stable struct {
		Schema                                          int
		Host, Project, Pool, Network, ACL, Image, State string
		QuotaMountPrecondition                          bool
		Checks                                          []struct {
			ID string
			OK bool
		}
	}
	v := stable{Schema: 2, Host: r.HostSHA256, Project: r.Project, Pool: r.Pool,
		Network: r.Network, ACL: r.ACL, Image: r.ImageFingerprint, State: r.NativeStateSHA256,
		QuotaMountPrecondition: r.QuotaMountPrecondition}
	for _, c := range r.Checks {
		v.Checks = append(v.Checks, struct {
			ID string
			OK bool
		}{c.ID, c.OK})
	}
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Do not hash incidental SDK read fields such as image last_used_at,
// diagnostic daemon environment or Project/profile used_by. They change
// without an operator changing any configuration. Hash only confinement-
// relevant values, including exact ACL attachments, image identity and
// the native daemon identity, then re-read the same fields at approval.
func nativePrereqCanonical(label string, object any) any {
	switch v := object.(type) {
	case *api.Server:
		return struct {
			Config     api.ConfigMap
			Extensions []string
			Version    string
			PID        int
			Host       string
		}{v.Config, v.APIExtensions, v.Environment.ServerVersion, v.Environment.ServerPid, v.Environment.ServerName}
	case *api.Project:
		return struct {
			Name string
			Put  api.ProjectPut
		}{v.Name, v.Writable()}
	case *api.Profile:
		return struct {
			Name string
			Put  api.ProfilePut
		}{v.Name, v.Writable()}
	case *api.StoragePool:
		return struct {
			Name, Driver, Status string
			Put                  api.StoragePoolPut
		}{v.Name, v.Driver, v.Status, v.Writable()}
	case *api.Network:
		return struct {
			Name, Type, Status string
			Managed            bool
			Put                api.NetworkPut
		}{v.Name, v.Type, v.Status, v.Managed, v.Writable()}
	case *api.NetworkACL:
		refs := slices.Clone(v.UsedBy)
		slices.Sort(refs)
		return struct {
			Name   string
			Put    api.NetworkACLPut
			UsedBy []string
		}{v.Name, v.Writable(), refs}
	case *api.ImageAliasesEntry:
		return struct {
			Name, Target string
		}{v.Name, v.Target}
	case *api.Image:
		return struct {
			Fingerprint, Architecture string
			Type                      string
			Properties                map[string]string
		}{v.Fingerprint, v.Architecture, v.Type, v.Properties}
	default:
		return object
	}
}

// NativePrerequisites is an explicitly root-local *inspection* using the
// official Incus SDK. All failures are represented as non-PASS checks.
func NativePrerequisites(ctx context.Context, project string) (NativePrereqReport, error) {
	r := NativePrereqReport{Project: project, PacketWitnessRequired: true}
	if os.Geteuid() != 0 {
		return r, errors.New("root-local native prerequisite review required")
	}
	if !safePrereqName.MatchString(project) || project == "default" {
		return r, errors.New("explicit nondefault restricted Project required")
	}
	host, err := migrationHostDigest()
	if err != nil {
		return r, err
	}
	r.HostSHA256 = host
	// Bind the review to the observed security-relevant native configuration;
	// preserve exact intended object identities, without volatile read state.
	state := sha256.New()
	observe := func(label string, object any, _ string) {
		data, _ := json.Marshal(nativePrereqCanonical(label, object))
		_, _ = fmt.Fprintf(state, "%s:%x\n", label, sha256.Sum256(data))
	}
	finish := func() NativePrereqReport {
		r.NativeStateSHA256 = hex.EncodeToString(state.Sum(nil))
		r.ConfigReady = len(r.Checks) != 0
		for _, c := range r.Checks {
			r.ConfigReady = r.ConfigReady && c.OK
		}
		r.ReviewSHA256 = prereqDigest(r)
		return r
	}
	u, err := user.Lookup("boxweave")
	actor := false
	actorDetail := "dedicated incus-only OS identity and exclusive Project"
	if err == nil && project == "user-"+u.Uid {
		if checkErr := nativeServiceActorReady(); checkErr == nil {
			actor = true
		} else {
			actorDetail = "restricted actor verification refused: " + checkErr.Error()
		}
	}
	prereqRecord(&r, "restricted_service_identity", actor, actorDetail)
	c, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		prereqRecord(&r, "native_daemon", false, "root-local Incus API unavailable")
		return finish(), nil
	}
	server, serverTag, err := c.GetServer()
	if err == nil {
		observe("server", server, serverTag)
	}
	extensions := err == nil && server != nil
	if extensions {
		for _, required := range []string{"projects", "projects_restrictions", "container_exec_recording", "network_acl"} {
			found := false
			for _, present := range server.APIExtensions {
				if present == required {
					found = true
					break
				}
			}
			extensions = extensions && found
		}
	}
	prereqRecord(&r, "native_extensions", extensions, "required actual daemon extensions")
	scoped := c.UseProject(project)
	p, projectTag, err := scoped.GetProject(project)
	if err == nil {
		observe("project", p, projectTag)
	}
	projectOK := err == nil && p != nil && p.Config["restricted"] == "true" &&
		p.Config["restricted.containers.nesting"] == "allow"
	prereqRecord(&r, "restricted_project_nesting", projectOK, "restricted Project with explicit L1 nesting capability")
	profile, profileTag, err := scoped.GetProfile("default")
	if err == nil {
		observe("profile", profile, profileTag)
	}
	var poolName, netName, aclName string
	devicesOK := false
	if err == nil && profile != nil {
		poolName, netName, aclName, devicesOK = prereqDevices(profile.Devices)
		incusGroup, groupErr := user.LookupGroup("incus")
		if u == nil || groupErr != nil ||
			!native.SafePerUserProfileConfig(profile.Config, u.Uid, incusGroup.Gid) {
			devicesOK = false
		}
	}
	r.Pool, r.Network, r.ACL = poolName, netName, aclName
	prereqRecord(&r, "root_disk_and_safe_nic", devicesOK, "exact root disk and one default-deny v4/v6 anti-spoofed NIC")
	poolOK := false
	if devicesOK {
		pool, poolTag, e := scoped.GetStoragePool(poolName)
		if e == nil {
			observe("pool", pool, poolTag)
		}
		if e == nil && pool != nil && pool.Status == "Created" && pool.Driver == "dir" {
			free, separate, quota, e := native.InspectNativeDirPool(poolName)
			poolOK = e == nil && (separate || poolName == "default") && free >= 64<<20
			r.QuotaMountPrecondition = quota
			if poolOK && !separate {
				r.Warnings = append(r.Warnings,
					"DIR_POOL_HOST_ROOT_SHARED: pool is not on an independent mount; account for shared filesystem exhaustion.")
			}
			if poolOK && !quota {
				r.Warnings = append(r.Warnings,
					"DIR_QUOTA_UNVERIFIED: Incus may skip requested volume size limits; inspect daemon logs and separately verify actual instance/Exec spool bounds.")
			}
		}
	}
	prereqRecord(&r, "dir_pool_backing", poolOK,
		"available dedicated Incus dir storage backing (quota evidence is separate, never silently assumed)")
	networkOK := false
	aclOK := false
	if devicesOK {
		// Incus can expose a bridge or ACL from the default network
		// project to a confined instance Project. Query both scopes as
		// root-local inspector; this never widens the runtime credential.
		network, networkTag, e := scoped.GetNetwork(netName)
		if e != nil {
			network, networkTag, e = c.GetNetwork(netName)
		}
		if e == nil {
			observe("network", network, networkTag)
		}
		networkOK = e == nil && network != nil && network.Type == "bridge" && network.Managed && network.Status == "Created"
		acl, aclTag, e := scoped.GetNetworkACL(aclName)
		if e != nil {
			acl, aclTag, e = c.GetNetworkACL(aclName)
		}
		if e == nil {
			observe("acl", acl, aclTag)
		}
		if e == nil && acl != nil {
			aclOK = aclExclusiveUse(acl.UsedBy, project)
		}
	}
	prereqRecord(&r, "managed_bridge", networkOK, "native managed bridge is present")
	prereqRecord(&r, "exclusive_acl_used_by", aclOK, "ACL used only by this Project's default profile/instances; no shared network or foreign owner")
	alias, aliasTag, err := scoped.GetImageAlias("boxweave-ubuntu-24.04")
	if err == nil {
		observe("image_alias", alias, aliasTag)
	}
	imageOK := false
	if err == nil && alias != nil && alias.Target != "" {
		img, imageTag, e := scoped.GetImage(alias.Target)
		if e == nil {
			observe("image", img, imageTag)
		}
		if e == nil && img != nil && strings.EqualFold(img.Properties["os"], "Ubuntu") &&
			(img.Properties["release"] == "noble" || img.Properties["release"] == "24.04") {
			r.ImageFingerprint = alias.Target
			imageOK = true
		}
	}
	prereqRecord(&r, "approved_image", imageOK, "existing local, approved Ubuntu 24.04 image; no remote alias pull")
	return finish(), nil
}

func NativePrerequisiteVerify(ctx context.Context, project, expectedSHA string, approved bool) (NativePrereqReport, error) {
	r, err := NativePrerequisites(ctx, project)
	if err != nil {
		return r, err
	}
	if !approved || len(expectedSHA) != 64 || expectedSHA != r.ReviewSHA256 {
		return r, fmt.Errorf("native pre-first-Box plan digest not approved or changed")
	}
	if !r.ConfigReady {
		return r, errors.New("native pre-first-Box prerequisites not ready: review failed checks")
	}
	return r, nil
}

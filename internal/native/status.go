// Package native provides read-only, project-confined official Incus SDK access.
package native

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"slices"

	incus "github.com/lxc/incus/v7/client"
)

type Status struct {
	Ready                 bool            `json:"ready"`
	Version               string          `json:"incus_version,omitempty"`
	Project               string          `json:"project"`
	ProfileReady          bool            `json:"default_profile_ready"`
	StorageReady          bool            `json:"selected_storage_pool_ready"`
	StorageSpaceChecked   bool            `json:"selected_storage_space_checked"`
	StorageAvailableBytes uint64          `json:"selected_storage_available_bytes,omitempty"`
	StorageSeparateMount  bool            `json:"selected_storage_separate_mount"`
	StorageQuotaVerified  bool            `json:"selected_storage_quota_verified"`
	Warnings              []string        `json:"warnings,omitempty"`
	NICConfigured         bool            `json:"default_nic_configured"`
	ImageReady            bool            `json:"approved_ubuntu_image_ready"`
	APIExtensions         map[string]bool `json:"required_extensions"`
	ErrorClass            string          `json:"error_class,omitempty"`
}

type Reader struct {
	Socket  string
	Project string
}

// Incus advertises projects_restrictions, not a generic "restricted" extension.
// Inspect the actual Project.Config["restricted"] separately at runtime.
var requiredExtensions = []string{"projects", "projects_restrictions", "container_exec_recording"}

func (r Reader) Read(ctx context.Context) Status {
	s := Status{Project: r.Project, APIExtensions: map[string]bool{}}
	if os.Geteuid() == 0 {
		s.ErrorClass = "REFUSE_ROOT_RUNTIME"
		return s
	}
	current, err := user.Current()
	if err != nil {
		s.ErrorClass = "OS_IDENTITY_UNAVAILABLE"
		return s
	}
	ids, err := current.GroupIds()
	if err != nil {
		s.ErrorClass = "OS_GROUPS_UNAVAILABLE"
		return s
	}
	incusGroup, err := user.LookupGroup("incus")
	if err != nil {
		s.ErrorClass = "INCUS_GROUP_MISSING"
		return s
	}
	if !slices.Contains(ids, incusGroup.Gid) {
		s.ErrorClass = "INCUS_USER_GROUP_REQUIRED"
		return s
	}
	for _, forbidden := range []string{"incus-admin", "sudo", "docker", "lxd"} {
		g, e := user.LookupGroup(forbidden)
		if e == nil && slices.Contains(ids, g.Gid) {
			s.ErrorClass = "UNSAFE_OS_PRIVILEGE"
			return s
		}
	}
	if r.Project == "" || r.Socket == "" {
		s.ErrorClass = "MISSING_NATIVE_CONFIG"
		return s
	}
	c, err := incus.ConnectIncusUnixWithContext(ctx, r.Socket, nil)
	if err != nil {
		s.ErrorClass = "INCUS_CONNECT_FAILED"
		return s
	}
	names, err := c.GetProjectNames()
	if err != nil {
		s.ErrorClass = "INCUS_PROJECT_QUERY_FAILED"
		return s
	}
	if !slices.Contains(names, r.Project) || len(names) != 1 {
		s.ErrorClass = "PROJECT_NOT_EXCLUSIVELY_CONFINED"
		return s
	}
	// UseProject is necessary, but native OS credentials must ALSO enforce confinement.
	scoped := c.UseProject(r.Project)
	project, _, err := scoped.GetProject(r.Project)
	if err != nil {
		s.ErrorClass = "PROJECT_CONFIG_UNAVAILABLE"
		return s
	}
	if project.Config["restricted"] != "true" {
		s.ErrorClass = "PROJECT_NOT_RESTRICTED"
		return s
	}
	server, _, err := scoped.GetServer()
	if err != nil {
		s.ErrorClass = "INCUS_SERVER_QUERY_FAILED"
		return s
	}
	s.Version = server.Environment.ServerVersion
	for _, ext := range requiredExtensions {
		s.APIExtensions[ext] = slices.Contains(server.APIExtensions, ext)
	}
	profile, _, err := scoped.GetProfile("default")
	if err != nil {
		s.ErrorClass = "PROFILE_UNAVAILABLE"
		return s
	}
	if !SafePerUserProfileConfig(profile.Config, current.Uid, incusGroup.Gid) {
		s.ErrorClass = "UNSAFE_DEFAULT_PROFILE_CONFIG"
		return s
	}
	var selectedPool string
	for _, d := range profile.Devices {
		if d["type"] == "disk" && d["path"] == "/" && d["pool"] != "" {
			s.ProfileReady = true
			selectedPool = d["pool"]
		}
		// Do not label a present NIC as safely configured merely because
		// it references a managed network. ACL default ingress and native
		// IPv4/IPv6/MAC filtering are mandatory before the first Box starts.
	}
	if !s.ProfileReady {
		s.ErrorClass = "ROOT_DISK_NOT_CONFIGURED"
		return s
	}
	pool, _, err := scoped.GetStoragePool(selectedPool)
	if err != nil || pool.Status != "Created" {
		s.ErrorClass = "SELECTED_STORAGE_POOL_UNAVAILABLE"
		return s
	}
	// The official SDK can return Created even if a selected dir pool lost
	// its own backing mount. Restricted actors cannot read pool.Config["source"],
	// so check Incus's actual canonical pool path with statfs, not host /.
	if pool.Driver == "dir" {
		free, separate, checkErr := localDirPoolSpace(selectedPool)
		s.StorageAvailableBytes = free
		s.StorageSeparateMount = separate
		if checkErr != nil {
			s.ErrorClass = "SELECTED_STORAGE_FILESYSTEM_UNAVAILABLE"
			switch {
			case errors.Is(checkErr, errNativeBackingUnverified):
				s.ErrorClass = "SELECTED_STORAGE_BACKING_UNVERIFIED"
			case errors.Is(checkErr, errNativeSpaceLow):
				s.ErrorClass = "SELECTED_STORAGE_SPACE_INSUFFICIENT"
			}
			return s
		}
		s.StorageSpaceChecked = true
		if !separate {
			s.Warnings = append(s.Warnings,
				"DIR_POOL_HOST_ROOT_SHARED: native dir storage shares the host root filesystem; monitor host-wide capacity.")
		}
		// All usable Incus dir backings are supported. A filesystem name
		// cannot establish whether the native daemon enforces size quotas.
		// Incus 7.6 may log a skipped-quota warning without returning it
		// through the SDK operation. Report conservative capability evidence.
		s.StorageQuotaVerified = localDirPoolQuotaReady(selectedPool)
		if !s.StorageQuotaVerified {
			s.Warnings = append(s.Warnings,
				"DIR_QUOTA_UNVERIFIED: Incus may skip per-Box disk size limits on this backing filesystem; recorded Exec output may also grow without that bound. Inspect Incus daemon warnings and monitor free space.")
		}
	} else {
		// Do not claim other storage drivers' *whole-instance* Exec sibling
		// quotas before a separately accepted native driver contract.
		s.ErrorClass = "SELECTED_STORAGE_QUOTA_DRIVER_UNVERIFIED"
		return s
	}
	s.StorageReady = true
	// Share the exact admission guard with doctor/host_status so a
	// misconfigured NIC is rejected before any mutation, not only at
	// the later Create/Start call. This is not packet-level acceptance.
	s.NICConfigured = nicProfilePrivate(profile.Devices)
	if !s.NICConfigured {
		s.ErrorClass = "DEFAULT_NIC_NOT_CONFIGURED"
		return s
	}
	// Readiness must cover the same immutable local image requirement as
	// Create. A profile and ACL alone cannot produce the first owned Box.
	// Never pull or trust a remote alias from the controller preflight.
	alias, _, err := scoped.GetImageAlias("boxweave-ubuntu-24.04")
	if err != nil || alias == nil || alias.Target == "" {
		s.ErrorClass = "APPROVED_IMAGE_ALIAS_UNAVAILABLE"
		return s
	}
	image, _, err := scoped.GetImage(alias.Target)
	if err != nil || image == nil || !approvedUbuntu2404(image.Properties) {
		s.ErrorClass = "APPROVED_IMAGE_UNVERIFIED"
		return s
	}
	s.ImageReady = true
	for _, required := range requiredExtensions {
		if !s.APIExtensions[required] {
			s.ErrorClass = "REQUIRED_EXTENSION_MISSING"
			return s
		}
	}
	s.Ready = true
	return s
}

func (r Reader) Strict(ctx context.Context) error {
	v := r.Read(ctx)
	if !v.Ready {
		if v.ErrorClass == "DEFAULT_NIC_NOT_CONFIGURED" {
			return fmt.Errorf("native preflight: %s: %w", v.ErrorClass, ErrUnsafeNetwork)
		}
		if v.ErrorClass == "SELECTED_STORAGE_QUOTA_DRIVER_UNVERIFIED" {
			// The requirement is checked before ANY new native request,
			// so a scoped worker may safely fail this admission without UNKNOWN.
			return fmt.Errorf("native preflight: %s: %w", v.ErrorClass, ErrNativeUnsupported)
		}
		return fmt.Errorf("native preflight: %s", v.ErrorClass)
	}
	if !v.APIExtensions["projects"] {
		return errors.New("native projects extension absent")
	}
	return nil
}

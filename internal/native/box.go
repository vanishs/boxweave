package native

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/lxc/incus/v7/shared/units"
)

var (
	ErrNotOwned          = errors.New("native instance UUID/project identity differs from owned ledger")
	ErrUnsafeNetwork     = errors.New("native private NIC ACL and IPv4/IPv6 anti-spoof not verified")
	ErrNativeUnsupported = errors.New("required safe native capability or approved image unavailable")
	ErrNativeDependents  = errors.New("instance has native dependent or foreign proxy devices")
	ErrUnverified        = errors.New("native operation or postcondition not independently verified")
)

var safeName = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

// InstanceAction never accepts raw Incus config, profiles or host paths.
// The credential and actual restricted project are fixed by Reader, not caller.
type InstanceAction struct {
	Name      string
	UUID      string
	Start     bool
	CPU       int
	MemoryMiB int64
	DiskGiB   int64
	Nesting   bool
	Internet  bool
}

type InstanceChanges struct {
	CPU       *int
	MemoryMiB *int64
	DiskGiB   *int64
	Nesting   *bool
	Internet  *bool
}

type BoxState struct {
	Name         string `json:"name"`
	UUID         string `json:"box_id"`
	Project      string `json:"project"`
	NativeState  string `json:"native_state"`
	GuestReady   bool   `json:"guest_ready"`
	CPU          string `json:"cpu"`
	Memory       string `json:"memory"`
	RootDiskSize string `json:"root_disk_size"`
	Nesting      bool   `json:"nesting"`
	Internet     bool   `json:"internet"`
}

type ExecSpec struct {
	Argv        []string
	Cwd         string
	UID         uint32
	GID         uint32
	Environment map[string]string
	TimeoutMS   int
}

type ExecEvidence struct {
	Exit       *int
	StdoutPath string
	StderrPath string
}

func (r Reader) confined(ctx context.Context) (incus.InstanceServer, error) {
	if err := r.Strict(ctx); err != nil {
		return nil, err
	}
	base, err := incus.ConnectIncusUnixWithContext(ctx, r.Socket, nil)
	if err != nil {
		return nil, err
	}
	// UseProject is not an authorization boundary by itself; Strict first
	// proves that the actual OS user has only the intended restricted Project.
	return base.UseProject(r.Project), nil
}

func validSize(cpu int, memory, disk int64) bool {
	return cpu >= 1 && cpu <= 64 && memory >= 128 && memory <= 262144 && disk >= 2 && disk <= 2048
}

func nicPrivate(devices api.DevicesMap) bool {
	count := 0
	for _, d := range devices {
		if d["type"] != "nic" {
			continue
		}
		count++
		if d["network"] == "" || strings.TrimSpace(d["security.acls"]) == "" ||
			strings.Contains(d["security.acls"], ",") ||
			d["security.acls.default.ingress.action"] != "reject" ||
			d["security.ipv4_filtering"] != "true" ||
			d["security.ipv6_filtering"] != "true" ||
			d["security.mac_filtering"] != "true" {
			return false
		}
	}
	return count == 1
}

// CheckBootstrap is intentionally stricter than native existence: a running
// unisolated bridge NIC is unacceptable even in an exclusive Incus Project.
// ownedPrivateNICEgress shadows exactly one verified private NIC without
// editing the shared Incus network, Profile or ACL. Incus 7.6's bridge NIC
// default egress is reject when an ACL is attached; an explicitly requested
// per-Box "internet" capability is represented by a local allow action.
// The inbound default reject and MAC/IPv4/IPv6 anti-spoof are preserved.
func ownedPrivateNICEgress(local, effective api.DevicesMap, enabled bool) (api.DevicesMap, error) {
	if !nicPrivate(effective) {
		return nil, ErrUnsafeNetwork
	}
	out := api.DevicesMap{}
	for k, d := range local {
		copyDevice := map[string]string{}
		for dk, dv := range d {
			copyDevice[dk] = dv
		}
		out[k] = copyDevice
	}
	for name, d := range effective {
		if d["type"] != "nic" {
			continue
		}
		// Prefer fully verified effective configuration rather than a
		// partial or stale shadow that could discard ACL/anti-spoof keys.
		copyNIC := map[string]string{}
		for dk, dv := range d {
			copyNIC[dk] = dv
		}
		if enabled {
			copyNIC["security.acls.default.egress.action"] = "allow"
		} else {
			copyNIC["security.acls.default.egress.action"] = "reject"
		}
		out[name] = copyNIC
		if !nicPrivate(out) {
			return nil, ErrUnsafeNetwork
		}
		return out, nil
	}
	return nil, ErrUnsafeNetwork
}

func (r Reader) CheckBootstrap(ctx context.Context) error {
	c, err := r.confined(ctx)
	if err != nil {
		return err
	}
	p, _, err := c.GetProfile("default")
	if err != nil {
		return err
	}
	if !nicPrivate(p.Devices) {
		return ErrUnsafeNetwork
	}
	return nil
}

// observeBoxEffectiveDevices reads ALL effective devices. Incus api.DevicesMap
// is a Go map: finding the root disk cannot terminate traversal because eth0
// may be visited later. The earlier break made internet=true randomly read
// false and even caused correctly created native Boxes to become UNKNOWN.
func observeBoxEffectiveDevices(out *BoxState, devices api.DevicesMap) {
	for _, d := range devices {
		if d["type"] == "nic" {
			out.Internet = d["security.acls.default.egress.action"] == "allow"
		}
		if d["type"] == "disk" && d["path"] == "/" {
			out.RootDiskSize = d["size"]
		}
	}
}

func (r Reader) GetExactBox(ctx context.Context, name, uuid string) (BoxState, string, error) {
	out := BoxState{Project: r.Project, Name: name}
	if !safeName.MatchString(name) || uuid == "" {
		return out, "", ErrNotOwned
	}
	c, err := r.confined(ctx)
	if err != nil {
		return out, "", err
	}
	inst, etag, err := c.GetInstance(name)
	if err != nil {
		return out, "", err
	}
	if inst.Config["volatile.uuid"] != uuid || inst.Project != "" && inst.Project != r.Project {
		return out, "", ErrNotOwned
	}
	out.UUID = uuid
	out.NativeState = inst.Status
	out.CPU = inst.ExpandedConfig["limits.cpu"]
	out.Memory = inst.ExpandedConfig["limits.memory"]
	out.Nesting = inst.ExpandedConfig["security.nesting"] == "true"
	observeBoxEffectiveDevices(&out, inst.ExpandedDevices)
	// Only a real guest probe may establish GuestReady. Native RUNNING
	// and guest shell readiness are different truths.
	return out, etag, nil
}

func (r Reader) CheckExactPrivateNIC(ctx context.Context, name, uuid string) error {
	c, err := r.confined(ctx)
	if err != nil {
		return err
	}
	inst, _, err := c.GetInstance(name)
	if err != nil {
		return err
	}
	if inst.Config["volatile.uuid"] != uuid || inst.Project != "" && inst.Project != r.Project {
		return ErrNotOwned
	}
	if !nicPrivate(inst.ExpandedDevices) {
		return ErrUnsafeNetwork
	}
	return nil
}

// Create uses only a preinstalled approved Ubuntu LTS image fingerprint. No
// remote image alias or arbitrary server is resolved from an MCP request.
// Incus simplestreams images label Ubuntu 24.04 using its release codename
// "noble". An operator-managed local alias and immutable fingerprint still
// select the actual image; do not infer trust from display description alone.
func approvedUbuntu2404(properties map[string]string) bool {
	return strings.EqualFold(properties["os"], "Ubuntu") &&
		(properties["release"] == "noble" || properties["release"] == "24.04")
}

func (r Reader) Create(ctx context.Context, a InstanceAction) (string, error) {
	if !safeName.MatchString(a.Name) || !validSize(a.CPU, a.MemoryMiB, a.DiskGiB) {
		return "", ErrNativeUnsupported
	}
	if err := r.Strict(ctx); err != nil {
		return "", err
	}
	c, err := r.confined(ctx)
	if err != nil {
		return "", err
	}
	if a.Start && !c.HasExtension("instance_create_start") {
		return "", ErrNativeUnsupported
	}
	project, _, err := c.GetProject(r.Project)
	if err != nil {
		return "", err
	}
	if a.Nesting && project.Config["restricted.containers.nesting"] != "allow" {
		return "", ErrNativeUnsupported
	}
	// Local operator provisioned image aliases are the sole allowed source.
	alias, _, err := c.GetImageAlias("boxweave-ubuntu-24.04")
	if err != nil || alias.Target == "" {
		return "", ErrNativeUnsupported
	}
	image, _, err := c.GetImage(alias.Target)
	if err != nil || image == nil || !approvedUbuntu2404(image.Properties) {
		return "", ErrNativeUnsupported
	}
	if _, _, err := c.GetInstance(a.Name); err == nil {
		return "", ErrNotOwned
	} else if !api.StatusErrorCheck(err, http.StatusNotFound) {
		return "", err
	}
	config := api.ConfigMap{
		"limits.cpu":    strconv.Itoa(a.CPU),
		"limits.memory": fmt.Sprintf("%dMiB", a.MemoryMiB),
	}
	if a.Nesting {
		config["security.nesting"] = "true"
	}
	root := map[string]string{}
	p, _, err := c.GetProfile("default")
	if err != nil {
		return "", err
	}
	rootKey := ""
	for name, d := range p.Devices {
		if d["type"] == "disk" && d["path"] == "/" && d["pool"] != "" {
			rootKey = name
			for k, v := range d {
				root[k] = v
			}
			root["size"] = fmt.Sprintf("%dGiB", a.DiskGiB)
			break
		}
	}
	if rootKey == "" {
		return "", ErrNativeUnsupported
	}
	devices, err := ownedPrivateNICEgress(api.DevicesMap{rootKey: root}, p.Devices, a.Internet)
	if err != nil {
		return "", err
	}
	// No magic host mount, arbitrary device or profile edit.
	op, err := c.CreateInstance(api.InstancesPost{
		Name: a.Name, Type: api.InstanceTypeContainer, Start: a.Start,
		Source: api.InstanceSource{Type: "image", Fingerprint: alias.Target},
		InstancePut: api.InstancePut{Profiles: []string{"default"}, Config: config,
			Devices: devices},
	})
	if err != nil {
		return "", err
	}
	id := op.Get().ID
	if id == "" {
		return "", ErrUnverified
	}
	return id, nil
}

func (r Reader) NativeOperation(ctx context.Context, id string) (*api.Operation, error) {
	if id == "" {
		return nil, ErrUnverified
	}
	c, err := r.confined(ctx)
	if err != nil {
		return nil, err
	}
	op, _, err := c.GetOperation(id)
	return op, err
}

// ObserveUntilFinal only polls native GET, never WaitContext (which can send
// Cancel on a timed-out context in the pinned Incus SDK).
func (r Reader) ObserveUntilFinal(ctx context.Context, id string) (*api.Operation, error) {
	for {
		op, err := r.NativeOperation(ctx, id)
		if err != nil {
			return nil, err
		}
		if op.StatusCode.IsFinal() {
			return op, nil
		}
		timer := time.NewTimer(400 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (r Reader) SetPower(ctx context.Context, name, uuid, action string) (string, error) {
	if action != "start" && action != "stop" {
		return "", ErrNativeUnsupported
	}
	state, _, err := r.GetExactBox(ctx, name, uuid)
	if err != nil {
		return "", err
	}
	if action == "start" {
		if err := r.Strict(ctx); err != nil {
			return "", err
		}
		if state.NativeState == "Running" {
			return "", ErrNativeUnsupported
		}
		if err := r.CheckExactPrivateNIC(ctx, name, uuid); err != nil {
			return "", err
		}
	} else if state.NativeState != "Running" {
		return "", ErrNativeUnsupported
	}
	c, err := r.confined(ctx)
	if err != nil {
		return "", err
	}
	op, err := c.UpdateInstanceState(name, api.InstanceStatePut{Action: action, Timeout: 30}, "")
	if err != nil {
		return "", err
	}
	id := op.Get().ID
	if id == "" {
		return "", ErrUnverified
	}
	return id, nil
}

func (r Reader) UpdateBox(ctx context.Context, name, uuid string, changes InstanceChanges) (string, error) {
	if err := r.Strict(ctx); err != nil {
		return "", err
	}
	_, etag, err := r.GetExactBox(ctx, name, uuid)
	if err != nil {
		return "", err
	}
	c, err := r.confined(ctx)
	if err != nil {
		return "", err
	}
	inst, etag2, err := c.GetInstance(name)
	if err != nil {
		return "", err
	}
	if etag2 != etag || inst.Config["volatile.uuid"] != uuid {
		return "", ErrNotOwned
	}
	w := inst.InstancePut
	config := api.ConfigMap{}
	for k, v := range w.Config {
		config[k] = v
	}
	w.Config = config
	if changes.CPU != nil {
		if *changes.CPU < 1 || *changes.CPU > 64 {
			return "", ErrNativeUnsupported
		}
		config["limits.cpu"] = strconv.Itoa(*changes.CPU)
	}
	if changes.MemoryMiB != nil {
		if *changes.MemoryMiB < 128 || *changes.MemoryMiB > 262144 {
			return "", ErrNativeUnsupported
		}
		config["limits.memory"] = fmt.Sprintf("%dMiB", *changes.MemoryMiB)
	}
	if changes.Nesting != nil {
		if *changes.Nesting {
			p, _, err := c.GetProject(r.Project)
			if err != nil {
				return "", err
			}
			if p.Config["restricted.containers.nesting"] != "allow" {
				return "", ErrNativeUnsupported
			}
		}
		config["security.nesting"] = strconv.FormatBool(*changes.Nesting)
	}
	if changes.Internet != nil {
		// Modify only this owned instance NIC. The shared ACL/rules and
		// immutable native identity are unchanged; ETag protects the merge.
		updated, err := ownedPrivateNICEgress(w.Devices, inst.ExpandedDevices, *changes.Internet)
		if err != nil {
			return "", err
		}
		w.Devices = updated
	}
	if changes.DiskGiB != nil {
		if *changes.DiskGiB < 2 || *changes.DiskGiB > 2048 {
			return "", ErrNativeUnsupported
		}
		old := int64(-1)
		var key string
		for k, d := range inst.ExpandedDevices {
			if d["type"] == "disk" && d["path"] == "/" {
				key = k
				if d["size"] != "" {
					old, err = units.ParseByteSizeString(d["size"])
					if err != nil {
						return "", ErrNativeUnsupported
					}
				}
				break
			}
		}
		if key == "" || old <= 0 || *changes.DiskGiB <= old/(1024*1024*1024) {
			return "", ErrNativeUnsupported
		}
		devices := api.DevicesMap{}
		for k, d := range w.Devices {
			copyD := map[string]string{}
			for dk, dv := range d {
				copyD[dk] = dv
			}
			devices[k] = copyD
		}
		if _, ok := devices[key]; !ok {
			devices[key] = map[string]string{}
			for dk, dv := range inst.ExpandedDevices[key] {
				devices[key][dk] = dv
			}
		}
		devices[key]["size"] = fmt.Sprintf("%dGiB", *changes.DiskGiB)
		w.Devices = devices
	}
	op, err := c.UpdateInstance(name, w, etag)
	if err != nil {
		return "", err
	}
	id := op.Get().ID
	if id == "" {
		return "", ErrUnverified
	}
	return id, nil
}

func (r Reader) DeleteBox(ctx context.Context, name, uuid string) (string, error) {
	box, _, err := r.GetExactBox(ctx, name, uuid)
	if err != nil {
		return "", err
	}
	if box.NativeState != "Stopped" {
		return "", ErrNativeDependents
	}
	c, err := r.confined(ctx)
	if err != nil {
		return "", err
	}
	inst, _, err := c.GetInstance(name)
	if err != nil {
		return "", err
	}
	if inst.Config["volatile.uuid"] != uuid {
		return "", ErrNotOwned
	}
	for _, d := range inst.ExpandedDevices {
		if d["type"] == "proxy" {
			return "", ErrNativeDependents
		}
	}
	op, err := c.DeleteInstance(name)
	if err != nil {
		return "", err
	}
	id := op.Get().ID
	if id == "" {
		return "", ErrUnverified
	}
	return id, nil
}

// boundedGuestCommand is tested independently of a native server: the GNU
// timeout end-of-options delimiter belongs BEFORE the duration argument.
// The executed program and its argv always remain inside the exact guest.
func boundedGuestCommand(spec ExecSpec) ([]string, error) {
	if spec.TimeoutMS == 0 {
		return spec.Argv, nil
	}
	if spec.TimeoutMS < 1000 || spec.TimeoutMS > 900000 {
		return nil, ErrNativeUnsupported
	}
	// Incus recorded output and native WaitForWS cannot be combined in 7.6.
	// Guest timeout bounds the command but does not promise rollback or
	// remote cancellation of already dispatched effects.
	return append([]string{"/usr/bin/timeout", "--signal=TERM",
		"--kill-after=5s", "--", fmt.Sprintf("%.3fs", float64(spec.TimeoutMS)/1000)},
		spec.Argv...), nil
}

func (r Reader) Exec(ctx context.Context, name, uuid string, spec ExecSpec) (string, error) {
	if err := r.Strict(ctx); err != nil {
		return "", err
	}
	box, _, err := r.GetExactBox(ctx, name, uuid)
	if err != nil {
		return "", err
	}
	if box.NativeState != "Running" {
		return "", ErrNativeUnsupported
	}
	if len(spec.Argv) == 0 || len(spec.Argv) > 64 {
		return "", ErrNativeUnsupported
	}
	for _, a := range spec.Argv {
		if a == "" || len(a) > 4096 || strings.ContainsRune(a, 0) {
			return "", ErrNativeUnsupported
		}
	}
	if spec.Cwd != "" && (!strings.HasPrefix(spec.Cwd, "/") ||
		len(spec.Cwd) > 256 || strings.Contains(spec.Cwd, "..")) {
		return "", ErrNativeUnsupported
	}
	if len(spec.Environment) > 32 {
		return "", ErrNativeUnsupported
	}
	for k, v := range spec.Environment {
		if len(k) > 64 || len(v) > 4096 || strings.ContainsAny(k, "=\x00\n\r") ||
			strings.ContainsRune(v, 0) {
			return "", ErrNativeUnsupported
		}
	}
	c, err := r.confined(ctx)
	if err != nil {
		return "", err
	}
	argv, err := boundedGuestCommand(spec)
	if err != nil {
		return "", err
	}
	op, err := c.ExecInstance(name, api.InstanceExecPost{
		Command: argv, Cwd: spec.Cwd, User: spec.UID, Group: spec.GID,
		Environment: spec.Environment, RecordOutput: true, WaitForWS: false,
	}, nil)
	if err != nil {
		return "", err
	}
	id := op.Get().ID
	if id == "" {
		return "", ErrUnverified
	}
	return id, nil
}

func execPath(raw any) string {
	s, ok := raw.(string)
	if !ok {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.IsAbs() || u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, "/1.0/instances/") ||
		!strings.Contains(u.Path, "/logs/exec-output/") || strings.Contains(u.Path, "..") {
		return ""
	}
	return u.Path
}

func ExecEvidenceFromFinal(op *api.Operation) (ExecEvidence, error) {
	var out ExecEvidence
	if op == nil || op.StatusCode != api.Success {
		return out, ErrUnverified
	}
	exit, ok := op.Metadata["return"]
	if !ok {
		return out, ErrUnverified
	}
	switch v := exit.(type) {
	case float64:
		if v < -2147483648 || v > 2147483647 || float64(int(v)) != v {
			return out, ErrUnverified
		}
		e := int(v)
		out.Exit = &e
	case int:
		e := v
		out.Exit = &e
	default:
		return out, ErrUnverified
	}
	paths, ok := op.Metadata["output"].(map[string]any)
	if !ok {
		return out, ErrUnverified
	}
	out.StdoutPath = execPath(paths["1"])
	out.StderrPath = execPath(paths["2"])
	if out.StdoutPath == "" && out.StderrPath == "" {
		return out, ErrUnverified
	}
	return out, nil
}

// MaxRecordedOutputBytes is a per-stream public read budget, not an assertion
// that native Incus has already truncated the underlying recorded log.
const MaxRecordedOutputBytes int64 = 16 << 20

// verifiedRecordedOutputURL builds the URL expected by official SDK DoHTTP.
// Unlike typed Incus queries, DoHTTP does not add the Project selector or
// complete a relative URL. Both must be supplied from trusted SDK connection
// data; guest/client arguments can never choose the host or arbitrary path.
func verifiedRecordedOutputURL(base, path, project string) (string, error) {
	if execPath(path) == "" || project == "" || len(project) > 128 {
		return "", ErrNativeUnsupported
	}
	u, err := url.Parse(base)
	if err != nil || u.User != nil || u.Host == "" ||
		(u.Scheme != "http" && u.Scheme != "https") ||
		u.RawQuery != "" || u.Fragment != "" {
		return "", ErrUnverified
	}
	u.Path = path
	q := u.Query()
	q.Set("project", project)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ReadRecordedOutput performs a bounded read through the official SDK's
// authenticated DoHTTP transport; URLs come only from verified native
// operation metadata already journaled by BoxWeave, not from the client.
func (r Reader) ReadRecordedOutput(ctx context.Context, path string, offset, length int64) ([]byte, bool, error) {
	if offset < 0 || offset > MaxRecordedOutputBytes ||
		length < 1 || length > 65536 || execPath(path) == "" {
		return nil, false, ErrNativeUnsupported
	}
	if offset == MaxRecordedOutputBytes {
		return []byte{}, true, nil
	}
	if length > MaxRecordedOutputBytes-offset {
		length = MaxRecordedOutputBytes - offset
	}
	c, err := r.confined(ctx)
	if err != nil {
		return nil, false, err
	}
	info, err := c.GetConnectionInfo()
	if err != nil || info == nil {
		return nil, false, ErrUnverified
	}
	readURL, err := verifiedRecordedOutputURL(info.URL, path, r.Project)
	if err != nil {
		return nil, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readURL, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := c.DoHTTP(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("native recorded output HTTP %d", resp.StatusCode)
	}
	if _, err := io.CopyN(io.Discard, resp.Body, offset); err != nil {
		if errors.Is(err, io.EOF) {
			return []byte{}, true, nil
		}
		return nil, false, err
	}
	buf := make([]byte, length+1)
	n, err := io.ReadFull(resp.Body, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, false, err
	}
	ended := int64(n) <= length
	if int64(n) > length {
		n = int(length)
	}
	return buf[:n], ended, nil
}

// ObserveCreated is only for a previously admitted create with its actual
// successful native operation ID. An isolated GET by name is NOT authority
// to adopt a foreign or originally UNKNOWN instance.
func (r Reader) ObserveCreated(ctx context.Context, name string) (BoxState, error) {
	var out BoxState
	if !safeName.MatchString(name) {
		return out, ErrNotOwned
	}
	c, err := r.confined(ctx)
	if err != nil {
		return out, err
	}
	inst, _, err := c.GetInstance(name)
	if err != nil {
		return out, err
	}
	uuid := inst.Config["volatile.uuid"]
	if uuid == "" || inst.Project != "" && inst.Project != r.Project {
		return out, ErrUnverified
	}
	if !nicPrivate(inst.ExpandedDevices) {
		return out, ErrUnsafeNetwork
	}
	out = BoxState{Project: r.Project, Name: name, UUID: uuid,
		NativeState: inst.Status, CPU: inst.ExpandedConfig["limits.cpu"],
		Memory:  inst.ExpandedConfig["limits.memory"],
		Nesting: inst.ExpandedConfig["security.nesting"] == "true"}
	observeBoxEffectiveDevices(&out, inst.ExpandedDevices)
	return out, nil
}

// ConfirmDeleteAbsence requires an independently completed native Delete
// Operation and two subsequent exact Project/name 404 observations. It never
// treats the pre-operation 404 (or name reuse) as permission to delete.
func (r Reader) ConfirmDeleteAbsence(ctx context.Context, name string) error {
	if !safeName.MatchString(name) {
		return ErrNotOwned
	}
	c, err := r.confined(ctx)
	if err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		_, _, err = c.GetInstance(name)
		if !api.StatusErrorCheck(err, http.StatusNotFound) {
			if err == nil {
				return ErrUnverified
			}
			return err
		}
		if i == 0 {
			timer := time.NewTimer(350 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil
}

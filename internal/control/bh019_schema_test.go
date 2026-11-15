package control

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

// The native dotted MCP properties must survive both official SDK schema
// inference and Go JSON decode, including optional explicit false values.
func TestBH019DottedMCPFieldsAndAgentHelp(t *testing.T) {
	keys := []string{
		"lxc.apparmor.profile",
		"security.privileged",
		"security.nesting",
		"security.idmap.isolated",
		"security.idmap.size",
		"security.idmap.base",
		"security.protection.delete",
		"boot.autostart",
		"security.guestapi",
		"lxc.apparmor.allow_nesting",
		"security.syscalls.intercept.mknod",
		"security.syscalls.intercept.setxattr",
		"security.syscalls.intercept.bpf",
		"security.syscalls.intercept.bpf.devices",
	}
	for _, typ := range []reflect.Type{
		reflect.TypeFor[createBoxInput](), reflect.TypeFor[updateBoxInput](),
	} {
		s, err := jsonschema.ForType(typ, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			prop := s.Properties[key]
			if prop == nil || prop.Description == "" {
				t.Errorf("%s missing described dotted field %s", typ, key)
			}
		}
		if _, ok := s.Properties["nesting"]; ok {
			t.Errorf("%s accidentally exposes legacy nesting alias", typ)
		}
	}
	var in createBoxInput
	if err := json.Unmarshal([]byte(`{"name":"bh019-dotted","security.nesting":true,"security.protection.delete":false,"security.guestapi":true,"lxc.apparmor.profile":"unset"}`), &in); err != nil {
		t.Fatal(err)
	}
	if !in.Nesting || in.ProtectionDelete == nil || *in.ProtectionDelete ||
		in.GuestAPI == nil || !*in.GuestAPI || in.AppArmorProfile == nil || *in.AppArmorProfile != "unset" {
		t.Fatal("dotted fields or explicit false lost during decode")
	}
}

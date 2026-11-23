package control

import (
	"testing"

	"github.com/vanishs/boxweave/internal/native"
)

func TestCreateNativeResourceShapeMustMatchOriginalIntent(t *testing.T) {
	in := createBoxInput{
		Name: "real-native", CPU: 2, MemoryMiB: 768, DiskGiB: 4, Nesting: false,
	}
	base := native.BoxState{CPU: "2", Memory: "768MiB", RootDiskSize: "4GiB", Nesting: false}
	if !exactCreateNativeShape(base, in) {
		t.Fatal("exact post-read incorrectly refused")
	}
	tests := []struct {
		name  string
		shape native.BoxState
	}{
		{"cpu", native.BoxState{CPU: "1", Memory: base.Memory, RootDiskSize: base.RootDiskSize}},
		{"memory", native.BoxState{CPU: base.CPU, Memory: "512MiB", RootDiskSize: base.RootDiskSize}},
		{"disk", native.BoxState{CPU: base.CPU, Memory: base.Memory, RootDiskSize: "2GiB"}},
		{"nesting", native.BoxState{CPU: base.CPU, Memory: base.Memory, RootDiskSize: base.RootDiskSize, Nesting: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if exactCreateNativeShape(tc.shape, in) {
				t.Fatal("wrong native guest configuration was accepted")
			}
		})
	}
}

func TestPR178CreateShapeRequiresPersistedAutostartDefault(t *testing.T) {
	in := createBoxInput{Name: "autostart-default", CPU: 2, MemoryMiB: 768, DiskGiB: 4}
	in.InstanceOptions = in.InstanceOptions.ForCreate()
	observed := native.BoxState{CPU: "2", Memory: "768MiB", RootDiskSize: "4GiB",
		NativeOptions: map[string]string{"boot.autostart": "true"}}
	if !exactCreateNativeShape(observed, in) {
		t.Fatal("valid native default rejected")
	}
	observed.NativeOptions["boot.autostart"] = "false"
	if exactCreateNativeShape(observed, in) {
		t.Fatal("native false accepted for default true")
	}
	delete(observed.NativeOptions, "boot.autostart")
	if exactCreateNativeShape(observed, in) {
		t.Fatal("native unset accepted for default true")
	}
}

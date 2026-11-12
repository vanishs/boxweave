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

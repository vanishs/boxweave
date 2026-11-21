package install

import "testing"

func TestNativeMigrationDirectoryLinks(t *testing.T) {
	for _, tt := range []struct {
		links uint64
		fs    int64
		want  bool
	}{
		{2, 0xef53, true},
		{1, 0xef53, false},
		{1, 0x9123683e, true},
		{0, 0x9123683e, false},
		{2, 0x9123683e, true},
		{1, 0, false},
	} {
		if got := nativeMigrationDirectoryLinksOK(tt.links, tt.fs); got != tt.want {
			t.Errorf("links=%d fs=%x: got %v, want %v", tt.links, tt.fs, got, tt.want)
		}
	}
}

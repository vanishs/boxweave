package native

import "testing"

func TestRC3IncusVersionBaseline(t *testing.T) {
	for _, tc := range []struct{ version string; accept bool }{
		{"7.6", true}, {"7.5.1", true}, {"7.7", true}, {"8.0", true},
		{"7.4", false}, {"6.0", false}, {"invalid", false}, {"", false},
	} {
		if got := CompatibleIncusVersion(tc.version); got != tc.accept {
			t.Errorf("version %q: got %v want %v", tc.version, got, tc.accept)
		}
	}
}

func TestRC3GuestImageSelectionNoArbitraryURLs(t *testing.T) {
	for _, tc := range []struct{ choice string; valid bool }{
		{"", true}, {"local:boxweave-default", true}, {"debian-13", true},
		{"images:ubuntu/24.04", true}, {"images:debian/13", true},
		{"https://evil.invalid/image", false}, {"foreign:ubuntu", false},
		{"images:", false}, {"local:../x", false}, {"local:a//b", false},
		{"images:../../x", false}, {"a:b", false},
	} {
		if got := ValidImageChoice(tc.choice); got != tc.valid {
			t.Errorf("image %q: got %v want %v", tc.choice, got, tc.valid)
		}
	}
	s := catalogImageSource("debian/13")
	if s.Type != "image" || s.Server != defaultCatalogURL || s.Protocol != "simplestreams" || s.Alias != "debian/13" {
		t.Fatal("unexpected Incus-native catalog request")
	}
}

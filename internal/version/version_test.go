package version

import "testing"

func TestVersionValid(t *testing.T) {
	cases := []struct {
		v    string
		want bool
	}{
		{"0.0.1-dev", true},
		{"1.2.3", true},
		{"0.1.0-rc.1+build.5", true},
		{"1.2", false},
		{"v1.2.3", false},
		{"", false},
		{"latest", false},
	}
	for _, c := range cases {
		if got := Valid(c.v); got != c.want {
			t.Errorf("Valid(%q) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestDefaultVersionIsValid(t *testing.T) {
	if !Valid(Version) {
		t.Fatalf("default Version %q is not a valid semver", Version)
	}
}

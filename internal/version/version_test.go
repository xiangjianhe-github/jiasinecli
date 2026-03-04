package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    Info
		wantErr bool
	}{
		{"v1.0.0", Info{Major: 1, Minor: 0, Patch: 0}, false},
		{"v2.3.1-beta.1", Info{Major: 2, Minor: 3, Patch: 1, PreRelease: "beta.1"}, false},
		{"1.0.0-alpha.1+build.123", Info{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha.1", BuildMeta: "build.123"}, false},
		{"invalid", Info{}, true},
	}

	for _, tt := range tests {
		got, err := Parse(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr {
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if got.PreRelease != tt.want.PreRelease {
				t.Errorf("Parse(%q).PreRelease = %q, want %q", tt.input, got.PreRelease, tt.want.PreRelease)
			}
		}
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"v1.0.1", "v1.0.0", 1},
		{"v1.0.0", "v1.0.1", -1},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.0.0", "v1.0.0-alpha.1", 1},  // 正式版 > 预发布
		{"v1.0.0-beta.1", "v1.0.0-alpha.1", 1},
	}

	for _, tt := range tests {
		a, _ := Parse(tt.a)
		b, _ := Parse(tt.b)
		got := a.Compare(b)
		if got != tt.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		info Info
		want string
	}{
		{Info{Major: 1, Minor: 0, Patch: 0}, "v1.0.0"},
		{Info{Major: 0, Minor: 1, Patch: 0, PreRelease: "alpha.1"}, "v0.1.0-alpha.1"},
		{Info{Major: 1, Minor: 2, Patch: 3, PreRelease: "rc.1", BuildMeta: "abc123"}, "v1.2.3-rc.1+abc123"},
	}

	for _, tt := range tests {
		got := tt.info.String()
		if got != tt.want {
			t.Errorf("Info.String() = %q, want %q", got, tt.want)
		}
	}
}

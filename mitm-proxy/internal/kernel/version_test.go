package kernel

import "testing"

func TestParseClampVersion(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{"", "?.?.?"},
		{"?.?.?", "?.?.?"},
		{"\r\n", "?.?.?"},
		{"?.?.?\r\n", "?.?.?"},
		{"1.2.3", "1.2.3"},
		{"v1.2.3-dev+abc\r\n", "v1.2.3-dev+abc"},
		{" 1.2.3 ", " 1.2.3 "},
	} {
		t.Run(test.input, func(t *testing.T) {
			version := ParseClampVersion(test.input)
			if got := version.String(); got != test.want {
				t.Fatalf("version = %q, want %q", got, test.want)
			}
			if test.want == "?.?.?" && version != UnknownClampVersion {
				t.Fatal("expected unknown version")
			}
		})
	}
}

func TestZeroVersionIsUnknown(t *testing.T) {
	var version ClampVersion
	if got := version.String(); got != "?.?.?" {
		t.Fatalf("zero version = %q", got)
	}
}

package forwardproxy

import "testing"

func TestSplitAuthority(t *testing.T) {
	tests := []struct {
		input string
		host  string
		port  int
		valid bool
	}{
		{"example.com", "example.com", 443, true},
		{"example.com:8443", "example.com", 8443, true},
		{"[2001:4860:4860::8888]:443", "2001:4860:4860::8888", 443, true},
		{"user@example.com:443", "", 0, false},
		{"example.com:0", "", 0, false},
		{"[example.com:443", "", 0, false},
	}
	for _, test := range tests {
		host, port, err := splitAuthority(test.input, 443)
		if (err == nil) != test.valid || host != test.host || port != test.port {
			t.Errorf("splitAuthority(%q) = %q, %d, %v", test.input, host, port, err)
		}
	}
}

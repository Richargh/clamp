package kernel

import "strings"

// BuildVersion is set with -ldflags -X.
var BuildVersion = "?.?.?"

var UnknownClampVersion = ClampVersion{}

type ClampVersion struct {
	value string
}

func ParseClampVersion(value string) ClampVersion {
	version := strings.TrimRight(value, "\r\n")
	if version == "" || version == "?.?.?" {
		return UnknownClampVersion
	}
	return ClampVersion{value: version}
}

func (v ClampVersion) String() string {
	if v.value == "" {
		return "?.?.?"
	}
	return v.value
}

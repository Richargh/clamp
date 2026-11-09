package session

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	containerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)
	volumeNamePattern    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	localImagePattern    = regexp.MustCompile(`^[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*$`)
	projectSeparators    = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)
)

type ContainerName struct{ value string }

func ParseContainerName(raw string) (ContainerName, error) {
	if !containerNamePattern.MatchString(raw) {
		return ContainerName{}, fmt.Errorf("invalid container name: %q", raw)
	}
	return ContainerName{value: raw}, nil
}

func (n ContainerName) String() string { return n.value }

// ProjectName is the container-facing project name, not the host folder path.
// Its zero value names an unnamed project "project".
type ProjectName struct{ value string }

func ParseProjectName(raw string) (ProjectName, error) {
	if !volumeNamePattern.MatchString(raw) {
		return ProjectName{}, fmt.Errorf("invalid project name: %q", raw)
	}
	return ProjectName{value: raw}, nil
}

func (n ProjectName) String() string {
	if n.value == "" {
		return "project"
	}
	return n.value
}

// ProjectNameFor converts a folder name into a safe volume-name prefix.
// Valid project names are unchanged.
func ProjectNameFor(workspace Workspace) ProjectName {
	name := projectSeparators.ReplaceAllString(workspace.baseName(), "-")
	name = strings.TrimLeft(name, "_.-")
	if name == "" {
		name = "project"
	}
	return ProjectName{value: name}
}

func (n ProjectName) NodeModulesVolume() VolumeName {
	return VolumeName{value: n.String() + "-node-modules"}
}

func (n ProjectName) GradleBuildVolume() VolumeName {
	return VolumeName{value: n.String() + "-gradle-build"}
}

func (n ProjectName) GradleCacheVolume() VolumeName {
	return VolumeName{value: n.String() + "-gradle-cache"}
}

// ImageName identifies a local, untagged image produced by Clamp.
// Registry references, tags, and digests are not accepted here.
type ImageName struct{ value string }

func ParseImageName(raw string) (ImageName, error) {
	if len(raw) > 255 || !localImagePattern.MatchString(raw) {
		return ImageName{}, fmt.Errorf("invalid local image name: %q", raw)
	}
	return ImageName{value: raw}, nil
}

func (n ImageName) String() string { return n.value }

type VolumeName struct{ value string }

func ParseVolumeName(raw string) (VolumeName, error) {
	if !volumeNamePattern.MatchString(raw) {
		return VolumeName{}, fmt.Errorf("invalid volume name: %q", raw)
	}
	return VolumeName{value: raw}, nil
}

func (n VolumeName) String() string { return n.value }

const sessionLogLayout = "20060102-150405-network.log"

// SessionLogName is a timestamped network-log filename, with no directory part.
type SessionLogName struct{ value string }

func ParseSessionLogName(raw string) (SessionLogName, error) {
	stamp, err := time.Parse(sessionLogLayout, raw)
	if err != nil || stamp.Format(sessionLogLayout) != raw {
		return SessionLogName{}, fmt.Errorf("invalid session log name: %q", raw)
	}
	return SessionLogName{value: raw}, nil
}

func (n SessionLogName) String() string { return n.value }

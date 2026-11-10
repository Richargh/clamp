package session

import (
	"fmt"
	"strconv"
)

type HarnessVolumeChoice uint8

const (
	NoChoiceMade  HarnessVolumeChoice = 0
	SharedVolume  HarnessVolumeChoice = 1
	ProjectVolume HarnessVolumeChoice = 2
)

func ParseHarnessVolumeChoice(raw string) (HarnessVolumeChoice, error) {
	switch raw {
	case "":
		return NoChoiceMade, nil
	case "global":
		return SharedVolume, nil
	case "project":
		return ProjectVolume, nil
	default:
		return NoChoiceMade, fmt.Errorf("unsupported harness storage: %q", raw)
	}
}

// HarnessVolume holds a validated volume name and an explicit choice.
// NoChoiceMade prevents launch until a shared or project volume is selected.
type HarnessVolume struct {
	choice HarnessVolumeChoice
	volume VolumeName
}

func NewHarnessStorage(volume VolumeName, choice HarnessVolumeChoice) (HarnessVolume, error) {
	name, err := ParseVolumeName(volume.String())
	if err != nil {
		return HarnessVolume{}, err
	}
	if choice != NoChoiceMade && choice != SharedVolume && choice != ProjectVolume {
		return HarnessVolume{}, fmt.Errorf("cannot construct harness storage with choice %d", choice)
	}
	return HarnessVolume{choice: choice, volume: name}, nil
}

func (s HarnessVolume) Choice() HarnessVolumeChoice { return s.choice }
func (s HarnessVolume) Volume() VolumeName          { return s.volume }

func resolveHarnessStorage(choice HarnessVolumeChoice, project projectIdentity) (HarnessVolume, error) {
	name := project.harness.prefix
	if choice == ProjectVolume {
		name += "-" + project.name.String() + "-" + strconv.FormatUint(uint64(project.key), 10)
	}
	volume, err := ParseVolumeName(name)
	if err != nil {
		return HarnessVolume{}, err
	}
	return NewHarnessStorage(volume, choice)
}

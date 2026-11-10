package session

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHarnessStorageChoices(t *testing.T) {
	for _, scenario := range []struct {
		raw    string
		choice HarnessVolumeChoice
	}{
		{"", NoChoiceMade}, {"global", SharedVolume}, {"project", ProjectVolume},
	} {
		t.Run(fmt.Sprintf("Should parse storage choice %q", scenario.raw), func(t *testing.T) {
			// given
			testee := ParseHarnessVolumeChoice
			// when
			actual, err := testee(scenario.raw)
			// then
			require.NoError(t, err)
			require.Equal(t, scenario.choice, actual)
		})
	}
	for _, raw := range []string{"invalid", "GLOBAL", " project", "project\n"} {
		t.Run(fmt.Sprintf("Should reject unsupported storage choice %q", raw), func(t *testing.T) {
			// given
			testee := ParseHarnessVolumeChoice
			// when
			_, err := testee(raw)
			// then
			require.ErrorContains(t, err, "unsupported harness storage")
		})
	}
	t.Run("Should use NoChoiceMade as the zero choice", func(t *testing.T) {
		// given
		var testee HarnessVolumeChoice
		// when
		actual := testee
		// then
		require.Equal(t, NoChoiceMade, actual)
	})
}

func TestHarnessStorageConstruction(t *testing.T) {
	for _, choice := range []HarnessVolumeChoice{NoChoiceMade, SharedVolume, ProjectVolume} {
		t.Run(fmt.Sprintf("Should construct storage with a valid volume name for choice %d", choice), func(t *testing.T) {
			// given
			volume, err := ParseVolumeName("harness-volume")
			require.NoError(t, err)
			testee := NewHarnessStorage
			// when
			actual, err := testee(volume, choice)
			// then
			require.NoError(t, err)
			require.Equal(t, volume, actual.Volume())
			require.Equal(t, choice, actual.Choice())
		})
	}
	for _, raw := range []string{"", "bad/name", "-invalid"} {
		t.Run(fmt.Sprintf("Should reject invalid volume name %q", raw), func(t *testing.T) {
			// given
			testee := NewHarnessStorage
			// when
			actual, err := testee(VolumeName{value: raw}, SharedVolume)
			// then
			require.ErrorContains(t, err, "invalid volume name")
			require.Equal(t, HarnessVolume{}, actual)
		})
	}
	for _, choice := range []HarnessVolumeChoice{HarnessVolumeChoice(99)} {
		t.Run(fmt.Sprintf("Should not construct selected storage for choice %d", choice), func(t *testing.T) {
			// given
			volume, err := ParseVolumeName("harness-volume")
			require.NoError(t, err)
			testee := NewHarnessStorage
			// when
			actual, err := testee(volume, choice)
			// then
			require.Error(t, err)
			require.Equal(t, HarnessVolume{}, actual)
		})
	}
}

func TestHarnessStorageVolumeResolution(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		choice HarnessVolumeChoice
		volume string
	}{
		{"unselected", NoChoiceMade, "claude-clamp"},
		{"shared", SharedVolume, "claude-clamp"},
		{"project", ProjectVolume, "claude-clamp-workspace-306203429"},
	} {
		t.Run("Should resolve the "+scenario.name+" volume before construction", func(t *testing.T) {
			// given
			workspace, err := ParseWorkspace("/workspace")
			require.NoError(t, err)
			harness, err := ParseHarness("claude")
			require.NoError(t, err)
			project := newProjectIdentity(harness, workspace)
			testee := resolveHarnessStorage
			// when
			actual, err := testee(scenario.choice, project)
			// then
			require.NoError(t, err)
			require.Equal(t, scenario.volume, actual.Volume().String())
			require.Equal(t, scenario.choice, actual.Choice())
		})
	}
	t.Run("Should distinguish project volumes for matching folder names", func(t *testing.T) {
		// given
		first, err := ParseWorkspace("/first/my-app")
		require.NoError(t, err)
		second, err := ParseWorkspace("/second/my-app")
		require.NoError(t, err)
		harness, err := ParseHarness("pi")
		require.NoError(t, err)
		testee := resolveHarnessStorage
		// when
		firstStorage, err := testee(ProjectVolume, newProjectIdentity(harness, first))
		require.NoError(t, err)
		secondStorage, err := testee(ProjectVolume, newProjectIdentity(harness, second))
		// then
		require.NoError(t, err)
		require.NotEqual(t, firstStorage.Volume(), secondStorage.Volume())
	})
}

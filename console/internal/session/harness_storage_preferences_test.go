package session

import (
	"path/filepath"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestHarnessStoragePreferenceWriting(t *testing.T) {
	for _, scenario := range []struct {
		value  string
		choice HarnessVolumeChoice
	}{
		{"global", SharedVolume}, {"project", ProjectVolume},
	} {
		t.Run("Should save the existing "+scenario.value+" preference format", func(t *testing.T) {
			// given
			memory := environment.NewMemory()
			settings := launchConfig(t)
			testee := New(memory.Env, settings)
			// when
			err := testee.rememberHarnessStorage("/project", scenario.choice)
			// then
			require.NoError(t, err)
			data, err := memory.Files.ReadFile(settings.AuthStore().String())
			require.NoError(t, err)
			require.Equal(t, "/project\t"+scenario.value+"\n", string(data))
		})
	}
	for _, scenario := range []struct {
		name   string
		choice HarnessVolumeChoice
	}{
		{"NoChoiceMade", NoChoiceMade}, {"an invalid choice", HarnessVolumeChoice(99)},
	} {
		t.Run("Should reject "+scenario.name+" without changing saved preferences", func(t *testing.T) {
			// given
			memory := environment.NewMemory()
			settings := launchConfig(t)
			path := settings.AuthStore().String()
			require.NoError(t, memory.Files.MkdirAll(filepath.Dir(path), 0700))
			original := []byte("/project\tproject\n")
			require.NoError(t, memory.Files.WriteFile(path, original, 0600))
			testee := New(memory.Env, settings)
			// when
			err := testee.rememberHarnessStorage("/project", scenario.choice)
			// then
			require.ErrorContains(t, err, "cannot save harness storage choice")
			data, err := memory.Files.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, original, data)
		})
	}
}

func TestSavedHarnessStorage(t *testing.T) {
	for _, scenario := range []struct {
		raw     string
		storage HarnessVolumeChoice
	}{
		{raw: "global", storage: SharedVolume},
		{raw: "project", storage: ProjectVolume},
	} {
		t.Run("Should parse saved "+scenario.raw+" storage without a prompt", func(t *testing.T) {
			// given
			memory := environment.NewMemory()
			settings := launchConfig(t)
			path := settings.AuthStore().String()
			require.NoError(t, memory.Files.MkdirAll(filepath.Dir(path), 0700))
			require.NoError(t, memory.Files.WriteFile(path, []byte("/project\t"+scenario.raw+"\n"), 0600))
			testee := New(memory.Env, settings)
			// when
			storage, err := testee.loadHarnessStorage("/project", false)
			// then
			require.NoError(t, err)
			require.Equal(t, scenario.storage, storage)
		})
	}
	t.Run("Should reject an unsupported saved storage choice instead of prompting", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		settings := launchConfig(t)
		path := settings.AuthStore().String()
		require.NoError(t, memory.Files.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, memory.Files.WriteFile(path, []byte("/project\tinvalid\n"), 0600))
		testee := New(memory.Env, settings)
		// when
		_, err := testee.loadHarnessStorage("/project", false)
		// then
		require.ErrorContains(t, err, "invalid Clamp harness storage for /project")
	})
	t.Run("Should replace an unsupported saved choice when a project volume is explicit", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		settings := launchConfig(t)
		path := settings.AuthStore().String()
		require.NoError(t, memory.Files.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, memory.Files.WriteFile(path, []byte("/other\tglobal\n/project\tinvalid\n"), 0600))
		testee := New(memory.Env, settings)
		// when
		storage, err := testee.loadHarnessStorage("/project", true)
		require.NoError(t, err)
		err = testee.rememberHarnessStorage("/project", storage)
		// then
		require.NoError(t, err)
		require.Equal(t, ProjectVolume, storage)
		stored, err := memory.Files.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "/other\tglobal\n/project\tproject\n", string(stored))
	})
}

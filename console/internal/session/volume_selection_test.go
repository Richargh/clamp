package session

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/stretchr/testify/require"
)

func TestLaunchVolumeSelection(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		choice HarnessVolumeChoice
		volume string
	}{
		{"shared", SharedVolume, "claude-clamp"},
		{"project", ProjectVolume, ""},
	} {
		t.Run("Should mount the selected "+scenario.name+" volume", func(t *testing.T) {
			// given
			env, workspace, processes := newLaunchEnv(t)
			options := launchOptions(workspace)
			options.PerProjectAuth = false
			selectHarnessVolume := func(selected Workspace) (HarnessVolumeChoice, error) {
				require.Equal(t, workspace, selected.String())
				return scenario.choice, nil
			}
			volume := scenario.volume
			if scenario.choice == ProjectVolume {
				volume = fmt.Sprintf("claude-clamp-project-with-spaces-%d", projectChecksum([]byte(workspace)))
			}
			testee := Launch

			// when
			err := testee(t.Context(), env, launchConfig(t), options, selectHarnessVolume)

			// then
			require.NoError(t, err)
			require.Contains(t, requireContainerRun(t, processes.Calls).Args, volume+":/home/dev/.claude")
		})
	}
	for _, source := range []string{"saved", "explicit"} {
		t.Run("Should not select a volume when the choice is "+source, func(t *testing.T) {
			// given
			env, workspace, _ := newLaunchEnv(t)
			settings := launchConfig(t)
			options := launchOptions(workspace)
			if source == "saved" {
				options.PerProjectAuth = false
				path := settings.User().AuthStore().String()
				require.NoError(t, env.Files.MkdirAll(filepath.Dir(path), 0700))
				require.NoError(t, env.Files.WriteFile(path, []byte(workspace+"\tglobal\n"), 0600))
			}
			selected := false
			selectHarnessVolume := func(Workspace) (HarnessVolumeChoice, error) {
				selected = true
				return ProjectVolume, nil
			}
			testee := Launch

			// when
			err := testee(t.Context(), env, settings, options, selectHarnessVolume)

			// then
			require.NoError(t, err)
			require.False(t, selected)
		})
	}
	for _, scenario := range []struct {
		name   string
		choice HarnessVolumeChoice
		err    error
	}{
		{"selection fails", NoChoiceMade, errors.New("selection failed")},
		{"no choice is made", NoChoiceMade, nil},
		{"the choice is invalid", HarnessVolumeChoice(99), nil},
	} {
		t.Run("Should stop before bootstrap when "+scenario.name, func(t *testing.T) {
			// given
			env, workspace, processes := newLaunchEnv(t)
			settings := launchConfig(t)
			options := launchOptions(workspace)
			options.PerProjectAuth = false
			selectHarnessVolume := func(Workspace) (HarnessVolumeChoice, error) {
				return scenario.choice, scenario.err
			}
			testee := Launch

			// when
			err := testee(t.Context(), env, settings, options, selectHarnessVolume)

			// then
			require.Error(t, err)
			if scenario.err != nil {
				require.ErrorIs(t, err, scenario.err)
			}
			require.Empty(t, processes.Calls)
			_, err = env.Files.Stat(settings.User().AuthStore().String())
			require.ErrorIs(t, err, filesystem.ErrNotExist)
			_, err = env.Files.Stat(filepath.Join(workspace, ".clamp"))
			require.ErrorIs(t, err, filesystem.ErrNotExist)
		})
	}
}

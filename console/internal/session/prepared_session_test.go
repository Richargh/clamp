package session

import (
	"path/filepath"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/filesystem"
	"github.com/stretchr/testify/require"
)

func TestSessionPreparation(t *testing.T) {
	t.Run("Should leave a new session without a harness volume choice", func(t *testing.T) {
		// given
		env, workspace, _ := newLaunchEnv(t)
		options := launchOptions(workspace)
		options.PerProjectAuth = false
		testee := New(env, launchConfig(t))
		// when
		prepared, err := testee.Prepare(options)
		// then
		require.NoError(t, err)
		require.True(t, prepared.HasNoHarnessVolume())
		require.Equal(t, workspace, prepared.Workspace().String())
	})
	t.Run("Should restore the saved volume choice during preparation", func(t *testing.T) {
		// given
		env, workspace, _ := newLaunchEnv(t)
		settings := launchConfig(t)
		path := settings.User().AuthStore().String()
		require.NoError(t, env.Files.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, env.Files.WriteFile(path, []byte(workspace+"\tglobal\n"), 0600))
		options := launchOptions(workspace)
		options.PerProjectAuth = false
		testee := New(env, settings)
		// when
		prepared, err := testee.Prepare(options)
		// then
		require.NoError(t, err)
		require.False(t, prepared.HasNoHarnessVolume())
		require.Equal(t, "claude-clamp", prepared.container.harnessStorage.Volume().String())
	})
	t.Run("Should prepare an explicit project volume without launching processes", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		testee := New(env, launchConfig(t))
		// when
		prepared, err := testee.Prepare(launchOptions(workspace))
		// then
		require.NoError(t, err)
		require.False(t, prepared.HasNoHarnessVolume())
		require.Empty(t, processes.Calls)
	})
	t.Run("Should not persist a storage preference during preparation", func(t *testing.T) {
		// given
		env, workspace, _ := newLaunchEnv(t)
		settings := launchConfig(t)
		testee := New(env, settings)
		// when
		_, err := testee.Prepare(launchOptions(workspace))
		// then
		require.NoError(t, err)
		_, err = env.Files.Stat(settings.User().AuthStore().String())
		require.ErrorIs(t, err, filesystem.ErrNotExist)
	})
}

func TestPreparedSessionVolume(t *testing.T) {
	t.Run("Should block launch when the choice is reset to NoChoiceMade", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		testee, err := New(env, launchConfig(t)).Prepare(launchOptions(workspace))
		require.NoError(t, err)
		require.False(t, testee.HasNoHarnessVolume())
		// when
		err = testee.SetHarnessVolume(NoChoiceMade)
		require.NoError(t, err)
		err = testee.Launch(t.Context())
		// then
		require.True(t, testee.HasNoHarnessVolume())
		require.Equal(t, NoChoiceMade, testee.container.harnessStorage.Choice())
		require.NotEmpty(t, testee.container.harnessStorage.Volume().String())
		require.ErrorContains(t, err, "harness storage selection is required")
		require.Empty(t, processes.Calls)
	})
	t.Run("Should use the latest storage selection for persistence and mounting", func(t *testing.T) {
		// given
		env, workspace, processes := newLaunchEnv(t)
		settings := launchConfig(t)
		options := launchOptions(workspace)
		options.PerProjectAuth = false
		testee, err := New(env, settings).Prepare(options)
		require.NoError(t, err)
		require.NoError(t, testee.SetHarnessVolume(ProjectVolume))
		// when
		require.NoError(t, testee.SetHarnessVolume(SharedVolume))
		err = testee.Launch(t.Context())
		// then
		require.NoError(t, err)
		require.Equal(t, SharedVolume, testee.container.harnessStorage.Choice())
		require.Equal(t, "claude-clamp", testee.container.harnessStorage.Volume().String())
		saved, err := env.Files.ReadFile(settings.User().AuthStore().String())
		require.NoError(t, err)
		require.Equal(t, workspace+"\tglobal\n", string(saved))
		run := requireContainerRun(t, processes.Calls)
		require.Contains(t, run.Args, "claude-clamp:/home/dev/.claude")
	})
	for _, scenario := range []struct {
		name         string
		volumeChoice HarnessVolumeChoice
	}{
		{"shared", SharedVolume}, {"project", ProjectVolume},
	} {
		t.Run("Should assign the selected "+scenario.name+" volume before launch", func(t *testing.T) {
			// given
			env, workspace, _ := newLaunchEnv(t)
			options := launchOptions(workspace)
			options.PerProjectAuth = false
			testee, err := New(env, launchConfig(t)).Prepare(options)
			require.NoError(t, err)
			// when
			err = testee.SetHarnessVolume(scenario.volumeChoice)
			// then
			require.NoError(t, err)
			require.False(t, testee.HasNoHarnessVolume())
			require.Equal(t, scenario.volumeChoice, testee.container.harnessStorage.Choice())
			require.NotEmpty(t, testee.container.harnessStorage.Volume().String())
		})
	}
}

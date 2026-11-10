package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/session"
	"github.com/stretchr/testify/require"
)

func TestRunHarnessStorage(t *testing.T) {
	t.Run("Should ask for storage through the run command and save the answer", func(t *testing.T) {
		// given
		memory, settings := cliMemory(t)
		memory.Stdin = strings.NewReader("2\n")
		memory.ProcessMemory.OnRun = withProxy(nil)
		testee := New(memory.Env, settings)
		// when
		err := testee.Run(t.Context(), []string{"run", "pi", "/project"})
		// then
		require.NoError(t, err)
		require.Contains(t, memory.Errors.String(), "Choice [1/2]:")
		saved, err := memory.Files.ReadFile(settings.User().AuthStore().String())
		require.NoError(t, err)
		require.Equal(t, "/project\tproject\n", string(saved))
	})
	t.Run("Should reuse saved storage without asking for input", func(t *testing.T) {
		// given
		memory, settings := cliMemory(t)
		memory.ProcessMemory.OnRun = withProxy(nil)
		path := settings.User().AuthStore().String()
		require.NoError(t, memory.Files.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, memory.Files.WriteFile(path, []byte("/project\tglobal\n"), 0600))
		testee := New(memory.Env, settings)
		// when
		err := testee.Run(t.Context(), []string{"run", "pi", "/project"})
		// then
		require.NoError(t, err)
		require.NotContains(t, memory.Errors.String(), "Choice [1/2]:")
	})
	t.Run("Should stop before process execution when the prompt reaches EOF", func(t *testing.T) {
		// given
		memory, settings := cliMemory(t)
		testee := New(memory.Env, settings)
		// when
		err := testee.Run(t.Context(), []string{"run", "pi", "/project"})
		// then
		require.ErrorContains(t, err, "unable to read harness storage preference")
		require.Empty(t, memory.ProcessMemory.Calls)
	})
}

func TestHarnessStoragePrompt(t *testing.T) {
	for _, scenario := range []struct {
		name, input string
		storage     session.HarnessVolumeChoice
	}{
		{name: "Should select a shared volume for choice one", input: "1\n", storage: session.SharedVolume},
		{name: "Should select a project volume for choice two", input: "2\n", storage: session.ProjectVolume},
		{name: "Should default to a shared volume for an empty choice", input: "\n", storage: session.SharedVolume},
		{name: "Should trim whitespace from the choice", input: " 2 \n", storage: session.ProjectVolume},
		{name: "Should retry an unsupported choice", input: "invalid\n2\n", storage: session.ProjectVolume},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			memory := environment.NewMemory()
			memory.Stdin = strings.NewReader(scenario.input)
			testee := promptHarnessStorage
			// when
			storage, err := testee(memory.Env, "/project")
			// then
			require.NoError(t, err)
			require.Equal(t, scenario.storage, storage)
		})
	}
	t.Run("Should fail when input ends before a choice is read", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		testee := promptHarnessStorage
		// when
		_, err := testee(memory.Env, "/project")
		// then
		require.ErrorContains(t, err, "unable to read harness storage preference")
	})
	t.Run("Should show the workspace and credential choices", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		memory.Stdin = strings.NewReader("1\n")
		testee := promptHarnessStorage
		// when
		_, err := testee(memory.Env, "/project")
		// then
		require.NoError(t, err)
		require.Contains(t, memory.Errors.String(), "/project")
		require.Contains(t, memory.Errors.String(), "global/shared credentials")
		require.Contains(t, memory.Errors.String(), "per-project credentials")
	})
}

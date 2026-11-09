package session

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/stretchr/testify/require"
)

func TestAuthPolicyPrompt(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		input  string
		policy AuthPolicy
	}{
		{name: "Should select global auth for choice one", input: "1\n", policy: GlobalAuthPolicy()},
		{name: "Should select project auth for choice two", input: "2\n", policy: ProjectAuthPolicy()},
		{name: "Should default to global auth for an empty choice", input: "\n", policy: GlobalAuthPolicy()},
		{name: "Should trim whitespace from the choice", input: " 2 \n", policy: ProjectAuthPolicy()},
		{name: "Should retry an unsupported choice", input: "invalid\n2\n", policy: ProjectAuthPolicy()},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			memory := environment.NewMemory()
			memory.Stdin = strings.NewReader(scenario.input)
			testee := New(memory.Env, launchConfig(t))

			// when
			policy, err := testee.promptAuthPolicy("/project")

			// then
			require.NoError(t, err)
			require.Equal(t, scenario.policy, policy)
		})
	}
	t.Run("Should fail when input ends before a choice is read", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		testee := New(memory.Env, launchConfig(t))

		// when
		_, err := testee.promptAuthPolicy("/project")

		// then
		require.ErrorContains(t, err, "unable to read auth preference")
	})
}

func TestSavedAuthPolicy(t *testing.T) {
	for _, scenario := range []struct {
		raw    string
		policy AuthPolicy
	}{
		{raw: "global", policy: GlobalAuthPolicy()},
		{raw: "project", policy: ProjectAuthPolicy()},
	} {
		t.Run("Should parse saved "+scenario.raw+" auth without a prompt", func(t *testing.T) {
			// given
			memory := environment.NewMemory()
			settings := launchConfig(t)
			path := settings.AuthStore().String()
			require.NoError(t, memory.Files.MkdirAll(filepath.Dir(path), 0700))
			require.NoError(t, memory.Files.WriteFile(path, []byte("/project\t"+scenario.raw+"\n"), 0600))
			testee := New(memory.Env, settings)

			// when
			policy, err := testee.authPolicy("/project", false)

			// then
			require.NoError(t, err)
			require.Equal(t, scenario.policy, policy)
		})
	}
	t.Run("Should reject an unsupported saved policy instead of prompting", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		memory.Stdin = strings.NewReader("2\n")
		settings := launchConfig(t)
		path := settings.AuthStore().String()
		require.NoError(t, memory.Files.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, memory.Files.WriteFile(path, []byte("/project\tinvalid\n"), 0600))
		testee := New(memory.Env, settings)

		// when
		_, err := testee.authPolicy("/project", false)

		// then
		require.ErrorContains(t, err, "invalid Clamp auth policy for /project")
	})
	t.Run("Should replace an unsupported saved policy when project auth is explicit", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		settings := launchConfig(t)
		path := settings.AuthStore().String()
		require.NoError(t, memory.Files.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, memory.Files.WriteFile(path, []byte("/other\tglobal\n/project\tinvalid\n"), 0600))
		testee := New(memory.Env, settings)

		// when
		policy, err := testee.authPolicy("/project", true)

		// then
		require.NoError(t, err)
		require.Equal(t, ProjectAuthPolicy(), policy)
		stored, err := memory.Files.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "/other\tglobal\n/project\tproject\n", string(stored))
	})
}

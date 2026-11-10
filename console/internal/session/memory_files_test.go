package session

import (
	"testing"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/kernel/process"
	"github.com/stretchr/testify/require"
)

func TestMemoryFileSystem(t *testing.T) {
	t.Run("Should copy shared domain rules using the injected file system", func(t *testing.T) {
		// given
		memory := environment.NewMemory()
		memory.ProcessMemory.OnRun = withProxy(nil)
		executable, err := process.ParseExecutable("/bin/docker")
		require.NoError(t, err)
		memory.ProcessMemory.Executables["docker"] = executable
		files := memory.Files
		for _, directory := range []string{"/project", "/clamp/clamp-shared/allowed-domains.d"} {
			require.NoError(t, files.MkdirAll(directory, 0700))
		}
		for _, name := range []string{"clamp-base", "clamp-pi"} {
			require.NoError(t, files.WriteFile("/clamp/"+name+".Dockerfile", []byte("FROM scratch\n"), 0600))
		}
		require.NoError(t, files.WriteFile("/clamp/clamp-shared/allowed-domains.d/go.txt", []byte("proxy.golang.org:443\n"), 0600))
		harness, err := ParseHarness("pi")
		require.NoError(t, err)
		runtime, err := ParseRuntime("docker")
		require.NoError(t, err)
		testee := New(memory.Env, launchConfig(t))
		options := Options{Harness: harness, Runtime: runtime, Folder: "/project", PerProjectAuth: true}

		// when
		err = testee.Launch(t.Context(), options)

		// then
		require.NoError(t, err)
		contents, err := files.ReadFile("/project/.clamp/allowed-domains.d/go.txt")
		require.NoError(t, err)
		require.Equal(t, "proxy.golang.org:443\n", string(contents))
	})
}

package alloweddomains

import (
	"os"
	"testing"

	filesystem "github.com/richargh/clamp/mitm-proxy/internal/kernel/fs"
	"github.com/richargh/clamp/mitm-proxy/internal/kernel/network"
	"github.com/stretchr/testify/require"
)

func TestParseAllowedEndpoint(t *testing.T) {
	t.Run("Should return a normalized address", func(t *testing.T) {
		// given
		const line = "PROXY.GOLANG.ORG.:0443"
		want, err := network.ParseAddress("proxy.golang.org:443")
		require.NoError(t, err)

		// when
		testee, err := parseAllowedEndpoint(line)

		// then
		require.NoError(t, err)
		require.Equal(t, want, testee)
	})
	t.Run("Should return a zero address for an invalid rule", func(t *testing.T) {
		// given
		const line = "example.com:0"

		// when
		testee, err := parseAllowedEndpoint(line)

		// then
		require.Error(t, err)
		require.Equal(t, network.Address{}, testee)
	})
}

func TestAllowedDomainDecisions(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		address string
		want    bool
	}{
		{name: "Should allow a listed address", address: "example.com:443", want: true},
		{name: "Should block an unlisted address", address: "other.example:443"},
		{name: "Should block an unlisted port", address: "example.com:80"},
		{name: "Should block an IP address", address: "127.0.0.1:443"},
		{name: "Should block port zero", address: "example.com:0"},
		{name: "Should block a zero address"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			testee, err := FromAddresses(mustAddress(t, "example.com:443"))
			require.NoError(t, err)
			var address network.Address
			if scenario.address != "" {
				address = mustAddress(t, scenario.address)
			}

			// when
			allowed := testee.IsAllowed(address)
			notAllowed := testee.IsNotAllowed(address)

			// then
			require.Equal(t, scenario.want, allowed)
			require.Equal(t, !allowed, notAllowed)
		})
	}
}

func TestUnrestrictedDomainDecisions(t *testing.T) {
	for _, target := range []string{"other.example:443", "127.0.0.1:443", "[::1]:443"} {
		t.Run("Should allow "+target+" with an unrestricted policy", func(t *testing.T) {
			// given
			testee := AllowAll()
			address := mustAddress(t, target)

			// when
			allowed := testee.IsAllowed(address)
			notAllowed := testee.IsNotAllowed(address)

			// then
			require.True(t, allowed)
			require.False(t, notAllowed)
		})
	}
}

func TestAllowedDomains(t *testing.T) {
	t.Run("Should allow only the endpoint loaded from rules.txt", func(t *testing.T) {
		// given
		files := filesystem.NewMemory(filesystem.MemoryOptions{Files: map[filesystem.Path][]byte{
			mustPath(t, "rules/rules.txt"): []byte("example.com:443"),
		}})
		address := mustAddress(t, "example.com:443")
		unlisted := mustAddress(t, "other.example:443")

		// when
		testee, err := Load(files, mustPath(t, "rules"))

		// then
		require.NoError(t, err)
		require.True(t, testee.IsAllowed(address))
		require.False(t, testee.IsAllowed(unlisted))
	})
	t.Run("Should merge text files", func(t *testing.T) {
		// given
		files := filesystem.NewMemory(filesystem.MemoryOptions{Files: map[filesystem.Path][]byte{
			mustPath(t, "rules/go.txt"):    []byte("proxy.golang.org:443"),
			mustPath(t, "rules/other.txt"): []byte("sum.golang.org:443"),
		}})

		// when
		testee, err := Load(files, mustPath(t, "rules"))

		// then
		require.NoError(t, err)
		require.True(t, testee.IsAllowed(mustAddress(t, "proxy.golang.org:443")))
		require.True(t, testee.IsAllowed(mustAddress(t, "sum.golang.org:443")))
	})
	t.Run("Should keep a startup snapshot", func(t *testing.T) {
		// given
		directory := mustPath(t, t.TempDir())
		path, err := directory.Join("rules.txt")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path.String(), []byte("example.com:443"), 0600))
		testee, err := Load(filesystem.NewOS(), directory)
		require.NoError(t, err)

		// when
		require.NoError(t, os.WriteFile(path.String(), []byte("changed.example:443"), 0600))

		// then
		require.True(t, testee.IsAllowed(mustAddress(t, "example.com:443")))
		require.False(t, testee.IsAllowed(mustAddress(t, "changed.example:443")))
	})
	for _, line := range []string{"example.com", "example.com:0", "example.com:65536", "*.example.com:443", "127.0.0.1:443", "127.0.0.1.:443", "[::1]:443", "bad host:443", "https://example.com:443", "-bad.example:443", "example.com..:443"} {
		t.Run("Should reject invalid rule "+line, func(t *testing.T) {
			// given
			files := filesystem.NewMemory(filesystem.MemoryOptions{Files: map[filesystem.Path][]byte{
				mustPath(t, "rules/rules.txt"): []byte("# comment\n" + line),
			}})

			// when
			_, err := Load(files, mustPath(t, "rules"))

			// then
			require.ErrorContains(t, err, "rules.txt:2")
		})
	}
	for _, ignored := range []string{"rules/ignored.md", "rules/nested/ignored.txt", "rules/nested.txt/ignored.txt"} {
		t.Run("Should ignore non-rule entry "+ignored, func(t *testing.T) {
			// given
			files := filesystem.NewMemory(filesystem.MemoryOptions{Files: map[filesystem.Path][]byte{
				mustPath(t, ignored): []byte("not a rule"),
			}})

			// when
			testee, err := Load(files, mustPath(t, "rules"))

			// then
			require.NoError(t, err)
			require.False(t, testee.IsAllowed(mustAddress(t, "example.com:443")))
		})
	}
	t.Run("Should block all destinations for an empty folder", func(t *testing.T) {
		// given
		directory := mustPath(t, t.TempDir())

		// when
		testee, err := Load(filesystem.NewOS(), directory)

		// then
		require.NoError(t, err)
		require.False(t, testee.IsAllowed(mustAddress(t, "example.com:443")))
	})
}

func TestAllowedRuleFormatting(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		contents string
	}{
		{name: "Should ignore comment lines", contents: "# Go modules\nexample.com:443"},
		{name: "Should ignore blank lines", contents: "\n\nexample.com:443\n\n"},
		{name: "Should ignore inline comments", contents: "example.com:443 # HTTPS"},
		{name: "Should read CRLF lines", contents: "example.com:443\r\n"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			files := filesystem.NewMemory(filesystem.MemoryOptions{Files: map[filesystem.Path][]byte{
				mustPath(t, "rules/rules.txt"): []byte(scenario.contents),
			}})

			// when
			testee, err := Load(files, mustPath(t, "rules"))

			// then
			require.NoError(t, err)
			require.True(t, testee.IsAllowed(mustAddress(t, "example.com:443")))
		})
	}
}

func mustPath(t *testing.T, value string) filesystem.Path {
	t.Helper()
	path, err := filesystem.ParsePath(value)
	require.NoError(t, err)
	return path
}

func mustAddress(t *testing.T, value string) network.Address {
	t.Helper()
	address, err := network.ParseAddress(value)
	require.NoError(t, err)
	return address
}

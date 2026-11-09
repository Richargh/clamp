package session

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthPolicyParsing(t *testing.T) {
	for _, scenario := range []struct {
		raw    string
		policy AuthPolicy
	}{
		{raw: "global", policy: GlobalAuthPolicy()},
		{raw: "project", policy: ProjectAuthPolicy()},
	} {
		t.Run("Should parse the "+scenario.raw+" auth policy", func(t *testing.T) {
			// given
			raw := scenario.raw

			// when
			testee, err := ParseAuthPolicy(raw)

			// then
			require.NoError(t, err)
			require.Equal(t, scenario.policy, testee)
		})
	}
	for _, raw := range []string{"", "invalid", "GLOBAL", " project", "project\n"} {
		t.Run(fmt.Sprintf("Should reject unsupported auth policy %q", raw), func(t *testing.T) {
			// given
			// when
			_, err := ParseAuthPolicy(raw)

			// then
			require.ErrorContains(t, err, "unsupported auth policy")
		})
	}
}

func TestAuthPolicyRepresentation(t *testing.T) {
	t.Run("Should use global auth for the zero value", func(t *testing.T) {
		// given
		var testee AuthPolicy

		// when
		actual := testee.String()

		// then
		require.Equal(t, "global", actual)
	})
	for _, scenario := range []struct {
		name   string
		policy AuthPolicy
	}{
		{name: "global", policy: GlobalAuthPolicy()},
		{name: "project", policy: ProjectAuthPolicy()},
	} {
		t.Run("Should serialize the "+scenario.name+" policy for storage", func(t *testing.T) {
			// given
			testee := scenario.policy

			// when
			actual := testee.String()

			// then
			require.Equal(t, scenario.name, actual)
		})
		t.Run("Should identify the credential scope of the "+scenario.name+" policy", func(t *testing.T) {
			// given
			testee := scenario.policy

			// when
			actual := testee.IsProject()

			// then
			require.Equal(t, scenario.name == "project", actual)
		})
	}
}

package session

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectFileRendering(t *testing.T) {
	t.Run("Should prepend the dynamic header without changing project additions", func(t *testing.T) {
		// given
		tool, err := ParseImageName("clamp-pi")
		require.NoError(t, err)
		additions := []byte("RUN echo project\n")
		testee := renderProjectDockerfile

		// when
		actual := testee(tool, additions)

		// then
		require.Equal(t, "ARG CLAMP_TOOL_IMAGE=clamp-pi\nFROM ${CLAMP_TOOL_IMAGE}\n\nARG CONTAINER_USER=dev\n\nCOPY .clamp/allowed-domains.d/ /opt/clamp-shared/allowed-domains.d/\n\nUSER $CONTAINER_USER\nWORKDIR /workspace\n\nRUN echo project\n", string(actual))
	})
	t.Run("Should restrict the build context before applying project exceptions", func(t *testing.T) {
		// given
		additions := []byte("!src/\n!src/**\n")
		testee := renderProjectDockerignore

		// when
		actual := testee(additions)

		// then
		require.Equal(t, "*\n!.clamp/\n!.clamp/**\n!src/\n!src/**\n", string(actual))
	})
}

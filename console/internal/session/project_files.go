package session

import "fmt"

const projectHeader = `ARG CLAMP_TOOL_IMAGE=%s
FROM ${CLAMP_TOOL_IMAGE}

ARG CONTAINER_USER=dev

COPY .clamp/allowed-domains.d/ /opt/clamp-shared/allowed-domains.d/

USER $CONTAINER_USER
WORKDIR /workspace

`

func renderProjectDockerfile(tool ImageName, additions []byte) []byte {
	return append([]byte(fmt.Sprintf(projectHeader, tool)), additions...)
}

func renderProjectDockerignore(additions []byte) []byte {
	return append([]byte("*\n!.clamp/\n!.clamp/**\n"), additions...)
}

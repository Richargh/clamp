package settings

const projectTemplate = `# Clamp prepends the base image automatically.
#
USER 0
# Add system packages as needed here, for example:
# RUN apt-get update && apt-get install -y --no-install-recommends \
#     <package> \
#     && rm -rf /var/lib/apt/lists/*
USER $CONTAINER_USER

# Other examples:
# COPY --chown=$CONTAINER_USER:$CONTAINER_USER mise.toml /tmp/mise.toml
# RUN mise install -C /tmp
# COPY --chown=$CONTAINER_USER:$CONTAINER_USER requirements.txt /tmp/requirements.txt
# RUN pip install --user -r /tmp/requirements.txt
# RUN npm install --global <package>
`
const ignoreTemplate = `# Include build-context files referenced by .clamp/clamp.Dockerfile.
# Clamp already includes .clamp/**.
#
# Examples:
# !mise.toml
# !requirements.txt
# !src/
# !src/**
`

func defaultProjectDockerfile(mise bool) []byte {
	content := projectTemplate
	if mise {
		content += "\nCOPY --chown=$CONTAINER_USER:$CONTAINER_USER mise.toml /tmp/mise.toml\nRUN mise install -C /tmp\n"
	}
	return []byte(content)
}

func defaultProjectDockerignore(mise bool) []byte {
	content := ignoreTemplate
	if mise {
		content += "\n!mise.toml\n"
	}
	return []byte(content)
}

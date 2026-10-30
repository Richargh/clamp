# hadolint ignore=DL3006
FROM clamp-base

# renovate: datasource=npm depName=@anthropic-ai/claude-code
ARG CLAUDE_CODE_VERSION=2.1.312
ARG CONTAINER_USER=dev

USER 0
RUN HOME=/root npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}

ENV DISABLE_AUTOUPDATER=1

COPY --chown=$CONTAINER_USER:$CONTAINER_USER claude-clamp-core /opt/claude-config
COPY --chown=$CONTAINER_USER:$CONTAINER_USER claude-clamp-workflows /opt/claude-workflows

USER $CONTAINER_USER

FROM clamp-base

ARG CLAUDE_CODE_VERSION=2.1.138
ARG USERNAME=dev

USER root
RUN npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}

ENV DISABLE_AUTOUPDATER=1

COPY --chown=$USERNAME:$USERNAME claude-clamp-core /opt/claude-config
COPY --chown=$USERNAME:$USERNAME claude-clamp-workflows /opt/claude-workflows

USER $USERNAME

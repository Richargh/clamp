# hadolint ignore=DL3006
FROM clamp-base

ARG OPENCODE_VERSION=1.2.27
ARG USERNAME=dev

USER root
RUN npm install -g opencode-ai@${OPENCODE_VERSION}

ENV OPENCODE_DISABLE_AUTOUPDATE=1

COPY --chown=$USERNAME:$USERNAME opencode-clamp-core /opt/opencode-config
RUN mkdir -p /home/$USERNAME/.local/share/opencode && chown -R $USERNAME:$USERNAME /home/$USERNAME/.local

USER $USERNAME

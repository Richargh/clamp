# hadolint ignore=DL3006
FROM clamp-base

ARG PI_VERSION=0.75.3
ARG USERNAME=dev

USER root
# Keep Pi packages on the same patch version. pi-coding-agent uses caret
# dependencies for these packages, and mixing 0.75.3 with newer pi-ai
# breaks GitHub Copilot login callbacks.
RUN HOME=/root npm install -g \
    @earendil-works/pi-coding-agent@${PI_VERSION} \
    @earendil-works/pi-agent-core@${PI_VERSION} \
    @earendil-works/pi-ai@${PI_VERSION} \
    @earendil-works/pi-tui@${PI_VERSION}

COPY --chown=$USERNAME:$USERNAME clamp-pi /opt/pi-config
RUN mkdir -p /home/$USERNAME/.pi && chown $USERNAME:$USERNAME /home/$USERNAME/.pi

USER $USERNAME

# hadolint ignore=DL3006
FROM clamp-base

# renovate: datasource=npm depName=@earendil-works/pi-coding-agent
ARG PI_VERSION=0.84.4
ARG CONTAINER_USER=dev

ENV PI_SKIP_VERSION_CHECK=1 \
    PI_TELEMETRY=0

USER 0
# Keep Pi packages on the same patch version. pi-coding-agent uses caret
# dependencies for these packages, and mixing 0.75.3 with newer pi-ai
# breaks GitHub Copilot login callbacks.
RUN HOME=/root npm install -g \
    @earendil-works/pi-coding-agent@${PI_VERSION} \
    @earendil-works/pi-agent-core@${PI_VERSION} \
    @earendil-works/pi-ai@${PI_VERSION} \
    @earendil-works/pi-tui@${PI_VERSION}

COPY --chown=$CONTAINER_USER:$CONTAINER_USER clamp-pi /opt/pi-config
RUN mkdir -p /home/$CONTAINER_USER/.pi && chown $CONTAINER_USER:$CONTAINER_USER /home/$CONTAINER_USER/.pi

USER $CONTAINER_USER

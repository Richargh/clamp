# hadolint ignore=DL3006
FROM clamp-base

# renovate: datasource=npm depName=opencode-ai
ARG OPENCODE_VERSION=1.2.27
ARG CONTAINER_USER=dev

USER root
RUN HOME=/root npm install -g opencode-ai@${OPENCODE_VERSION}

ENV OPENCODE_DISABLE_AUTOUPDATE=1

COPY --chown=$CONTAINER_USER:$CONTAINER_USER opencode-clamp-core /opt/opencode-config
RUN mkdir -p /home/$CONTAINER_USER/.local/share/opencode && chown -R $CONTAINER_USER:$CONTAINER_USER /home/$CONTAINER_USER/.local

USER $CONTAINER_USER

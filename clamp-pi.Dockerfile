FROM clamp-base

ARG PI_VERSION=0.75.3
ARG USERNAME=dev

USER root
RUN npm install -g @earendil-works/pi-coding-agent@${PI_VERSION}

COPY --chown=$USERNAME:$USERNAME clamp-pi /opt/pi-config
RUN mkdir -p /home/$USERNAME/.pi && chown $USERNAME:$USERNAME /home/$USERNAME/.pi

USER $USERNAME

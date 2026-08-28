FROM debian:13.6-slim

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install prerequisites and tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl=7.88.1-10+deb12u* \
    ca-certificates=20230311+deb12u* \
    && apt-get install -y --no-install-recommends \
    iptables=1.8.9-2 \
    ipset=7.17-1 \
    iproute2=6.1.0-3 \
    bind9-dnsutils=1:9.18.49-1~deb12u2 \
    aggregate=1.6-7+b1 \
    fd-find=8.6.0-3 \
    git=1:2.39.5-0+deb12u* \
    jq=1.6-2.1+deb12u* \
    procps=2:4.0.2-3 \
    ripgrep=13.0.0-4+b2 \
    zip=3.0-13 \
    unzip=6.0-28 \
    libglib2.0-0=2.74.6-2+deb12u* \
    libnspr4=2:4.35-1 \
    libnss3=2:3.87.1-1+deb12u* \
    libatk1.0-0=2.46.0-5 \
    libatk-bridge2.0-0=2.46.0-5 \
    libdbus-1-3=1.14.10-1~deb12u* \
    libcups2=2.4.2-3+deb12u* \
    libxkbcommon0=1.5.0-1 \
    libatspi2.0-0=2.46.0-5 \
    libxcomposite1=1:0.4.5-1 \
    libxdamage1=1:1.1.6-1 \
    libxfixes3=1:6.0.0-2 \
    libxrandr2=2:1.5.2-2+b1 \
    libgbm1=22.3.6-1+deb12u* \
    libcairo2=1.16.0-7 \
    libpango-1.0-0=1.50.12+ds-1 \
    shellcheck=0.9.0-1 \
    && ln -s /usr/bin/fdfind /usr/local/bin/fd \
    && HADOLINT_ARCH="$(dpkg --print-architecture)" \
    && if [ "$HADOLINT_ARCH" = "amd64" ]; then HADOLINT_ARCH="x86_64"; fi \
    && curl -fsSL "https://github.com/hadolint/hadolint/releases/download/v2.12.0/hadolint-Linux-${HADOLINT_ARCH}" -o /usr/local/bin/hadolint \
    && chmod +x /usr/local/bin/hadolint \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

ARG USERNAME=dev
ARG USER_UID=1001
ARG USER_GID=$USER_UID
ARG NODE_VERSION=24.13.0

# Install mise
RUN curl -fsSL https://mise.run -o /tmp/mise-install.sh \
    && MISE_INSTALL_PATH=/usr/local/bin/mise sh /tmp/mise-install.sh \
    && rm /tmp/mise-install.sh

# Install image-provided Node.js with mise into a root-owned system location
RUN export MISE_YES=1 MISE_DATA_DIR=/opt/mise MISE_CONFIG_DIR=/etc/mise MISE_CACHE_DIR=/var/cache/mise \
    && mise install node@${NODE_VERSION} \
    && ln -s "$(mise where node@${NODE_VERSION})" /opt/node

ENV HOME=/home/$USERNAME \
    PATH=/home/$USERNAME/.local/share/mise/shims:/opt/node/bin:$PATH

# Create non-root user and workspace
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && useradd --system --gid $USER_GID --no-create-home --shell /usr/sbin/nologin clamp-proxy \
    && mkdir -p /workspace /workspace/node_modules /workspace/build /home/$USERNAME/.gradle \
    && chown -R $USERNAME:$USERNAME /workspace /home/$USERNAME

# Copy startup scripts. Project-specific .clamp/clamp.Dockerfile copies selected
COPY startup-scripts/ /usr/local/bin/

WORKDIR /workspace
USER $USERNAME

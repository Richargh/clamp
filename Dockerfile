FROM debian:12.13-slim

ARG CLAUDE_CODE_VERSION=2.1.138
ARG OPENCODE_VERSION=1.2.27

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install prerequisites, add external repositories, and tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl=7.88.1-10+deb12u* \
    ca-certificates=20230311+deb12u* \
    gnupg=2.2.40-1.1+deb12u* \
    && mkdir -p /etc/apt/keyrings \
    # Add Eclipse Temurin (Adoptium) repository \
    && curl -fsSL https://packages.adoptium.net/artifactory/api/gpg/key/public | gpg --dearmor -o /etc/apt/keyrings/adoptium.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/adoptium.gpg] https://packages.adoptium.net/artifactory/deb bookworm main" > /etc/apt/sources.list.d/adoptium.list \
    # Add NodeSource repository for Node.js 24.x \
    && curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_24.x nodistro main" > /etc/apt/sources.list.d/nodesource.list \
    # Version control \
    && apt-get install -y --no-install-recommends \
    # Network (firewall) tools \
    iptables=1.8.9-2 \
    ipset=7.17-1 \
    iproute2=6.1.0-3 \
    bind9-dnsutils=1:9.18.47-1~deb12u1 \
    aggregate=1.6-7+b1 \
    # General tools \
    git=1:2.39.5-0+deb12u* \
    jq=1.6-2.1+deb12u* \
    sudo=1.9.13p3-1+deb12u* \
    procps=2:4.0.2-3 \
    zip=3.0-13 \
    unzip=6.0-28 \
    # Chromium dependencies \
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
    # Linting tools \
    shellcheck=0.9.0-1 \
    && HADOLINT_ARCH="$(dpkg --print-architecture)" \
    && if [ "$HADOLINT_ARCH" = "amd64" ]; then HADOLINT_ARCH="x86_64"; fi \
    && curl -fsSL "https://github.com/hadolint/hadolint/releases/download/v2.12.0/hadolint-Linux-${HADOLINT_ARCH}" -o /usr/local/bin/hadolint \
    && chmod +x /usr/local/bin/hadolint \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Java and Node.js (from external repos)
RUN apt-get update && apt-get install -y --no-install-recommends \
    temurin-21-jdk=21.0.9.0.0+10-0 \
    nodejs=24.13.0-1nodesource1 \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Create non-root user and workspace
ARG USERNAME=dev
ARG USER_UID=1001
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    # Allow passwordless sudo for changing firewall in startup script \
    && echo "$USERNAME ALL=(root) NOPASSWD: /usr/local/bin/container-startup.sh" > /etc/sudoers.d/$USERNAME \
    && chmod u=r,g=r,o= /etc/sudoers.d/$USERNAME \
    # Create workspace and volume mount directories with correct ownership \
    # (Docker preserves ownership when initializing named volumes from existing dirs) \
    && mkdir -p /workspace /workspace/node_modules /workspace/build /home/$USERNAME/.gradle \
    && mkdir -p /home/$USERNAME/.local/share/opencode \
    && chown -R $USERNAME:$USERNAME /workspace /home/$USERNAME/.gradle /home/$USERNAME/.local

# Install Claude Code and OpenCode
RUN npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION} \
    && npm install -g opencode-ai@${OPENCODE_VERSION} \
    && npm install -g @earendil-works/pi-coding-agent@0.75.3

# Disable auto-updates since global npm packages require root permissions
ENV DISABLE_AUTOUPDATER=1
ENV OPENCODE_DISABLE_AUTOUPDATE=1

# Copy startup scripts
COPY startup-scripts/ /usr/local/bin/

# Copy shared config (allowed-domains.txt)
COPY --chown=$USERNAME:$USERNAME clamp-shared /opt/clamp-shared
# Copy claude config to template location (fresh copy on each container start)
COPY --chown=$USERNAME:$USERNAME claude-clamp-core /opt/claude-config
# Copy claude workflows to template location
COPY --chown=$USERNAME:$USERNAME claude-clamp-workflows /opt/claude-workflows
# Copy opencode config to template location
COPY --chown=$USERNAME:$USERNAME opencode-clamp-core /opt/opencode-config
# Copy pi config to template location
COPY --chown=$USERNAME:$USERNAME clamp-pi /opt/pi-config
RUN mkdir -p /home/$USERNAME/.pi && chown $USERNAME:$USERNAME /home/$USERNAME/.pi

WORKDIR /workspace
USER $USERNAME

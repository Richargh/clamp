FROM debian:12.13-slim

ARG CLAUDE_CODE_VERSION=2.1.0
ARG OPENCODE_VERSION=1.1.49

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install prerequisites, add external repositories, and tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl=7.88.1-10+deb12u14 \
    ca-certificates=20230311+deb12u1 \
    gnupg=2.2.40-1.1+deb12u2 \
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
    dnsutils=1:9.18.41-1~deb12u1 \
    aggregate=1.6-7+b1 \
    # General tools \
    git=1:2.39.5-0+deb12u3 \
    jq=1.6-2.1+deb12u1 \
    sudo=1.9.13p3-1+deb12u3 \
    procps=2:4.0.2-3 \
    zip \
    unzip \
    # Chromium dependencies \
    libglib2.0-0 \
    libnspr4 \
    libnss3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libdbus-1-3 \
    libcups2 \
    libxkbcommon0 \
    libatspi2.0-0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libcairo2 \
    libpango-1.0-0 \
    # Linting tools \
    shellcheck=0.9.0-1 \
    && curl -fsSL https://github.com/hadolint/hadolint/releases/download/v2.12.0/hadolint-Linux-x86_64 -o /usr/local/bin/hadolint \
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
    && npm install -g opencode-ai@${OPENCODE_VERSION}

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

WORKDIR /workspace
USER $USERNAME

FROM debian:12.13-slim

ARG CLAUDE_CODE_VERSION=2.1.0

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
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Java and Node.js (from external repos)
RUN apt-get update && apt-get install -y --no-install-recommends \
    temurin-21-jdk=21.0.9.0.0+10-0 \
    nodejs=24.13.0-1nodesource1 \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Create non-root user, workspace, and install Claude Code
ARG USERNAME=dev
ARG USER_UID=1001
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    # Allow passwordless sudo for firewall script only \
    && echo "$USERNAME ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" > /etc/sudoers.d/$USERNAME \
    && chmod u=r,g=r,o= /etc/sudoers.d/$USERNAME \
    # Create workspace and config directories \
    && mkdir -p /workspace /home/$USERNAME/.claude \
    && chown $USERNAME:$USERNAME /workspace /home/$USERNAME/.claude \
    # Install Claude Code \
    && npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}

# Disable auto-update since global npm packages require root permissions
ENV CLAUDE_CODE_DISABLE_AUTO_UPDATE=1

# Copy claude directory (settings, hooks, allowed-domains.txt)
COPY --chown=$USERNAME:$USERNAME claude/ /home/$USERNAME/.claude/
RUN chmod +x /home/$USERNAME/.claude/hooks/**/*.mjs

# Copy firewall script
COPY init-firewall.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/init-firewall.sh

WORKDIR /workspace
USER $USERNAME

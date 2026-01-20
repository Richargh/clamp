FROM eclipse-temurin:21.0.8_9-jdk

ARG CLAUDE_CODE_VERSION=latest

# Install Node.js 20.x + minimal tools for Claude and firewall
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    gnupg \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_20.x nodistro main" > /etc/apt/sources.list.d/nodesource.list \
    && apt-get update && apt-get install -y --no-install-recommends \
    nodejs \
    git \
    iptables \
    ipset \
    iproute2 \
    dnsutils \
    aggregate \
    jq \
    sudo \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Create non-root user
ARG USERNAME=dev
ARG USER_UID=1001
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && echo "$USERNAME ALL=(root) NOPASSWD:ALL" > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# Create workspace and claude config directory
RUN mkdir -p /workspace /home/$USERNAME/.claude \
    && chown $USERNAME:$USERNAME /workspace /home/$USERNAME/.claude

# Install Claude Code globally
RUN npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}

# Copy claude directory (settings, hooks, allowed-domains.txt)
COPY --chown=$USERNAME:$USERNAME claude/ /home/$USERNAME/.claude/
RUN chmod +x /home/$USERNAME/.claude/hooks/**/*.mjs

# Copy firewall script
COPY init-firewall.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/init-firewall.sh

WORKDIR /workspace
USER $USERNAME

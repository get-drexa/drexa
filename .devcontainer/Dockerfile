FROM mcr.microsoft.com/devcontainers/javascript-node:24

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    gnupg \
    lsb-release \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Bun as the node user
USER node
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/home/node/.bun/bin:$PATH"

# Switch back to root for any remaining setup
USER root

# Ensure the node user owns their home directory
RUN chown -R node:node /home/node

# Set the default user back to node
USER node

# Set working directory
WORKDIR /workspace

# Verify installations
RUN bun --version
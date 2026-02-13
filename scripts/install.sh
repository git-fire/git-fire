#!/bin/bash
# Git-Fire Installer
#
# Install git-fire:
#   curl -fsSL https://git-fire.sh/install | bash
#
# OR:
#   wget -qO- https://git-fire.sh/install | bash

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
GITHUB_REPO="TBRX103/git-fire"

echo -e "${GREEN}🔥 Installing Git-Fire...${NC}\n"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo "Detected: $OS-$ARCH"

# Download URL
BINARY_URL="https://github.com/$GITHUB_REPO/releases/latest/download/git-fire-$OS-$ARCH"

# Alternative: build from source if binary not available
BUILD_FROM_SOURCE=false

# Try to download binary
echo "Downloading from $BINARY_URL..."

mkdir -p "$INSTALL_DIR"

if command -v curl &> /dev/null; then
    if ! curl -fsSL "$BINARY_URL" -o "$INSTALL_DIR/git-fire" 2>/dev/null; then
        BUILD_FROM_SOURCE=true
    fi
elif command -v wget &> /dev/null; then
    if ! wget -q "$BINARY_URL" -O "$INSTALL_DIR/git-fire" 2>/dev/null; then
        BUILD_FROM_SOURCE=true
    fi
else
    echo -e "${RED}Error: curl or wget required${NC}"
    exit 1
fi

# Build from source if download failed
if [ "$BUILD_FROM_SOURCE" = true ]; then
    echo -e "${YELLOW}Binary not available, building from source...${NC}"

    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is required to build from source${NC}"
        echo "Install Go from: https://golang.org/dl/"
        exit 1
    fi

    # Clone and build
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    git clone "https://github.com/$GITHUB_REPO.git"
    cd git-fire

    go build -o "$INSTALL_DIR/git-fire" .

    cd -
    rm -rf "$TMP_DIR"
fi

# Make executable
chmod +x "$INSTALL_DIR/git-fire"

# Verify installation
if [ -x "$INSTALL_DIR/git-fire" ]; then
    echo -e "\n${GREEN}✓ Git-Fire installed successfully!${NC}\n"

    # Check if in PATH
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        echo -e "${YELLOW}⚠️  $INSTALL_DIR is not in your PATH${NC}\n"
        echo "Add this to your shell rc file (~/.bashrc, ~/.zshrc, etc.):"
        echo -e "  ${GREEN}export PATH=\"$INSTALL_DIR:\$PATH\"${NC}\n"

        # Detect shell and suggest
        if [ -n "$BASH_VERSION" ]; then
            echo "For bash:"
            echo -e "  ${GREEN}echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc${NC}"
            echo -e "  ${GREEN}source ~/.bashrc${NC}\n"
        elif [ -n "$ZSH_VERSION" ]; then
            echo "For zsh:"
            echo -e "  ${GREEN}echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc${NC}"
            echo -e "  ${GREEN}source ~/.zshrc${NC}\n"
        fi
    fi

    # Test run
    echo "Testing installation..."
    if "$INSTALL_DIR/git-fire" --help > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Test passed${NC}\n"
    else
        echo -e "${YELLOW}⚠️  Installation may have issues${NC}\n"
    fi

    # Next steps
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  Next steps:${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "1. Generate config (optional):"
    echo -e "   ${GREEN}git-fire --init${NC}"
    echo ""
    echo "2. Test with dry-run:"
    echo -e "   ${GREEN}git-fire --dry-run${NC}"
    echo ""
    echo "3. Emergency mode (when building is on fire):"
    echo -e "   ${GREEN}git-fire${NC}"
    echo ""
    echo "For help:"
    echo -e "   ${GREEN}git-fire --help${NC}"
    echo ""

else
    echo -e "${RED}✗ Installation failed${NC}"
    exit 1
fi

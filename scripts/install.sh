#!/bin/bash
# Git-Fire Installer
#
# Install git-fire:
#   curl -fsSL https://raw.githubusercontent.com/git-fire/git-fire/main/scripts/install.sh | bash
#
# OR:
#   wget -qO- https://raw.githubusercontent.com/git-fire/git-fire/main/scripts/install.sh | bash

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
GITHUB_REPO="git-fire/git-fire"

echo -e "${GREEN}🔥 Installing Git-Fire...${NC}\n"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)          ARCH="amd64" ;;
    aarch64|arm64)   ARCH="arm64" ;;
    armv7l|armv6l)   ARCH="armv6" ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo "Detected: $OS-$ARCH"

# Fetch latest release version from GitHub API
echo "Fetching latest release..."
if command -v curl &> /dev/null; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/$GITHUB_REPO/releases/latest" \
        | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"v\([^"]*\)".*/\1/')
elif command -v wget &> /dev/null; then
    VERSION=$(wget -qO- "https://api.github.com/repos/$GITHUB_REPO/releases/latest" \
        | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"v\([^"]*\)".*/\1/')
else
    echo -e "${RED}Error: curl or wget required${NC}"
    exit 1
fi

if [ -z "$VERSION" ]; then
    echo -e "${RED}Error: could not determine latest version${NC}"
    exit 1
fi

echo "Latest version: v$VERSION"

# Build tarball name and URL (GoReleaser naming convention)
# e.g. git-fire_1.2.3_linux_amd64.tar.gz
ARCHIVE_NAME="git-fire_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/v${VERSION}/${ARCHIVE_NAME}"

# Alternative: build from source if binary not available
BUILD_FROM_SOURCE=false

# Download and extract
echo "Downloading $ARCHIVE_NAME..."
mkdir -p "$INSTALL_DIR"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

if command -v curl &> /dev/null; then
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME" 2>/dev/null; then
        BUILD_FROM_SOURCE=true
    fi
elif command -v wget &> /dev/null; then
    if ! wget -q "$DOWNLOAD_URL" -O "$TMP_DIR/$ARCHIVE_NAME" 2>/dev/null; then
        BUILD_FROM_SOURCE=true
    fi
fi

if [ "$BUILD_FROM_SOURCE" = false ]; then
    tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"
    cp "$TMP_DIR/git-fire" "$INSTALL_DIR/git-fire"
fi

# Build from source if download failed
if [ "$BUILD_FROM_SOURCE" = true ]; then
    echo -e "${YELLOW}Binary not available, building from source...${NC}"

    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is required to build from source${NC}"
        echo "Install Go from: https://golang.org/dl/"
        exit 1
    fi

    BUILD_DIR=$(mktemp -d)
    git clone "https://github.com/$GITHUB_REPO.git" "$BUILD_DIR/git-fire"
    (cd "$BUILD_DIR/git-fire" && go build -ldflags="-s -w" -o "$INSTALL_DIR/git-fire" .)
    rm -rf "$BUILD_DIR"
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

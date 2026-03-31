#!/bin/bash
# Git-Fire Emergency Script
#
# One-liner install and execute:
#   curl -fsSL https://git-fire.sh/emergency | bash
#
# OR if git-fire is hosted on GitHub:
#   curl -fsSL https://raw.githubusercontent.com/git-fire/git-fire/main/scripts/emergency.sh | bash
#
# What this does:
# 1. Check if git-fire binary is installed
# 2. If yes: use it (fastest, full features)
# 3. If no: download and run it, or fall back to pure bash
# 4. Execute emergency backup
# 5. LEAVE BUILDING

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="git-fire"
GITHUB_REPO="git-fire/git-fire"

banner() {
    echo -e "${RED}"
    cat << "EOF"
╔═══════════════════════════════════════╗
║                                       ║
║     🔥 GIT FIRE EMERGENCY MODE 🔥     ║
║                                       ║
║   PUSH ALL REPOS - LEAVE BUILDING!    ║
║                                       ║
╚═══════════════════════════════════════╝
EOF
    echo -e "${NC}\n"
}

# Check if git-fire is already installed
check_installed() {
    if command -v git-fire &> /dev/null; then
        return 0
    fi

    # Check local install
    if [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        export PATH="$INSTALL_DIR:$PATH"
        return 0
    fi

    return 1
}

# Try to download and install git-fire binary
try_download() {
    echo -e "${YELLOW}git-fire not found. Attempting download...${NC}\n"

    # Detect OS and architecture
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$ARCH" in
        x86_64)        ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l|armv6l) ARCH="armv6" ;;
    esac

    # Fetch latest version tag from GitHub API
    VERSION=""
    if command -v curl &> /dev/null; then
        VERSION=$(curl -fsSL "https://api.github.com/repos/$GITHUB_REPO/releases/latest" \
            | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"v\([^"]*\)".*/\1/' 2>/dev/null || true)
    elif command -v wget &> /dev/null; then
        VERSION=$(wget -qO- "https://api.github.com/repos/$GITHUB_REPO/releases/latest" \
            | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"v\([^"]*\)".*/\1/' 2>/dev/null || true)
    fi

    if [ -z "$VERSION" ]; then
        echo -e "${YELLOW}Could not determine latest version, using fallback${NC}\n"
        return 1
    fi

    # GoReleaser archive naming: git-fire_VERSION_OS_ARCH.tar.gz
    ARCHIVE_NAME="git-fire_${VERSION}_${OS}_${ARCH}.tar.gz"
    BINARY_URL="https://github.com/$GITHUB_REPO/releases/download/v${VERSION}/${ARCHIVE_NAME}"
    TMP_DIR=$(mktemp -d)

    # Try to download
    DOWNLOADED=false
    if command -v curl &> /dev/null; then
        if curl -fsSL "$BINARY_URL" -o "$TMP_DIR/$ARCHIVE_NAME" 2>/dev/null; then
            DOWNLOADED=true
        fi
    elif command -v wget &> /dev/null; then
        if wget -q "$BINARY_URL" -O "$TMP_DIR/$ARCHIVE_NAME" 2>/dev/null; then
            DOWNLOADED=true
        fi
    fi

    if [ "$DOWNLOADED" = true ]; then
        mkdir -p "$INSTALL_DIR"
        tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"
        cp "$TMP_DIR/git-fire" "$INSTALL_DIR/$BINARY_NAME"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
        rm -rf "$TMP_DIR"
        export PATH="$INSTALL_DIR:$PATH"
        echo -e "${GREEN}✓ Downloaded git-fire${NC}\n"
        return 0
    fi

    rm -rf "$TMP_DIR"
    echo -e "${YELLOW}Could not download binary, using fallback bash implementation${NC}\n"
    return 1
}

# Pure bash fallback (no dependencies except git)
bash_fallback() {
    echo -e "${BLUE}Using pure bash emergency mode${NC}\n"

    # Find all git repos
    repos=$(find . -name ".git" -type d -prune 2>/dev/null | sed 's|/.git||')

    if [ -z "$repos" ]; then
        echo -e "${RED}No git repositories found!${NC}"
        exit 1
    fi

    total=$(echo "$repos" | wc -l)
    count=0
    success=0
    failed=0

    echo -e "${YELLOW}Found $total repositories${NC}\n"

    # Process each repo
    while IFS= read -r repo; do
        ((count++))
        echo -e "${GREEN}[$count/$total]${NC} $repo"

        (
            cd "$repo" || exit 1

            # Auto-commit if dirty
            if [[ -n $(git status --porcelain 2>/dev/null) ]]; then
                echo "  💥 Dirty - committing..."
                git add -A
                git commit -m "🔥 EMERGENCY FIRE BACKUP - $(date '+%Y-%m-%d %H:%M:%S')" 2>/dev/null || true
            fi

            # Push to all remotes
            remotes=$(git remote 2>/dev/null)
            if [ -z "$remotes" ]; then
                echo "  ⊘ No remotes"
                exit 0
            fi

            for remote in $remotes; do
                branch=$(git branch --show-current 2>/dev/null)
                if [ -n "$branch" ]; then
                    if git push "$remote" "$branch" 2>/dev/null; then
                        echo "  ✓ Pushed to $remote"
                    else
                        echo "  ✗ Failed: $remote"
                        exit 1
                    fi
                fi
            done
        ) && ((success++)) || ((failed++))

    done <<< "$repos"

    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  ✓ Success: $success repos${NC}"
    if [ $failed -gt 0 ]; then
        echo -e "${RED}  ✗ Failed: $failed repos${NC}"
    fi
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Main execution
main() {
    banner

    # Strategy 1: Use installed git-fire
    if check_installed; then
        echo -e "${GREEN}✓ Using installed git-fire${NC}\n"
        exec git-fire --fire
    fi

    # Strategy 2: Try to download binary
    if try_download; then
        exec git-fire --fire
    fi

    # Strategy 3: Fallback to pure bash
    bash_fallback

    # Final message
    echo ""
    echo -e "${RED}"
    cat << "EOF"
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                                  ┃
┃   ✓ BACKUP COMPLETE              ┃
┃   🏃 LEAVE BUILDING NOW!         ┃
┃                                  ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
EOF
    echo -e "${NC}"

    # Auto-install for next time
    if ! check_installed && [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        echo -e "\n${BLUE}Tip: Add to your PATH for faster execution:${NC}"
        echo -e "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo -e "\nOr install properly:"
        echo -e "  curl -fsSL https://git-fire.sh/install | bash"
    fi
}

# Run it
main "$@"

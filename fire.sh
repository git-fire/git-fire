#!/bin/bash
# fire.sh - Minimal emergency git backup script
# Usage: ./fire.sh
#
# What it does:
# 1. Find all git repos in current directory
# 2. Auto-commit everything
# 3. Push to all remotes
# 4. LEAVE BUILDING

set -e  # Exit on any error

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${RED}"
cat << "EOF"
🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥
🔥  GIT FIRE EMERGENCY MODE  🔥
🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥
EOF
echo -e "${NC}"

# Find all git repos
repos=$(find . -name ".git" -type d -prune | sed 's|/.git||')

if [ -z "$repos" ]; then
    echo "No git repositories found!"
    exit 1
fi

total=$(echo "$repos" | wc -l)
count=0

echo -e "${YELLOW}Found $total repositories${NC}\n"

# Process each repo
while IFS= read -r repo; do
    ((count++))
    echo -e "${GREEN}[$count/$total]${NC} Processing: $repo"
    cd "$repo" || continue

    # Check if dirty
    if [[ -n $(git status --porcelain) ]]; then
        echo "  💥 Dirty - auto-committing..."
        git add -A
        git commit -m "🔥 EMERGENCY BACKUP - $(date '+%Y-%m-%d %H:%M:%S')" || true
    else
        echo "  ✓ Clean"
    fi

    # Get remotes
    remotes=$(git remote)

    if [ -z "$remotes" ]; then
        echo "  ⊘ No remotes - skipping"
        cd - > /dev/null
        continue
    fi

    # Push to all remotes
    for remote in $remotes; do
        echo "  🚀 Pushing to $remote..."

        # Get current branch
        branch=$(git branch --show-current)

        # Push (force if needed)
        if git push "$remote" "$branch" 2>/dev/null; then
            echo "    ✓ Pushed to $remote/$branch"
        else
            echo "    ⚠️  Failed to push to $remote (may need credentials)"
        fi
    done

    cd - > /dev/null

done <<< "$repos"

echo ""
echo -e "${RED}"
cat << "EOF"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓ BACKUP COMPLETE
  🏃 LEAVE BUILDING NOW!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF
echo -e "${NC}"

exit 0

#!/bin/bash

set -e

echo "This script will remove import.json from git history"
echo "================================================"

cd /home/radityaharya/projects/gocheck

if [ ! -d .git ]; then
    echo "Error: Not a git repository"
    exit 1
fi

echo "Step 1: Adding import.json to .gitignore"
if ! grep -q "import.json" .gitignore; then
    echo "" >> .gitignore
    echo "# Import data files with secrets" >> .gitignore
    echo "import.json" >> .gitignore
    echo "Added import.json to .gitignore"
else
    echo "import.json already in .gitignore"
fi

echo ""
echo "Step 2: Checking if import.json exists in git history"
if git log --all --full-history --oneline -- import.json | head -1 > /dev/null 2>&1; then
    echo "Found import.json in git history"
    
    echo ""
    echo "Step 3: Installing git-filter-repo (if not installed)"
    if ! command -v git-filter-repo &> /dev/null; then
        echo "git-filter-repo not found. Installing..."
        pip3 install git-filter-repo || {
            echo "Failed to install git-filter-repo"
            echo "Please install it manually: pip3 install git-filter-repo"
            echo ""
            echo "Alternative: You can use git filter-branch (slower):"
            echo "git filter-branch --force --index-filter 'git rm --cached --ignore-unmatch import.json' --prune-empty --tag-name-filter cat -- --all"
            exit 1
        }
    fi
    
    echo ""
    echo "Step 4: Removing import.json from git history"
    echo "Creating backup branch..."
    git branch backup-before-filter-$(date +%Y%m%d-%H%M%S) 2>/dev/null || true
    
    echo "Removing file from history..."
    git filter-repo --path import.json --invert-paths --force
    
    echo ""
    echo "Step 5: Cleaning up"
    git reflog expire --expire=now --all
    git gc --prune=now --aggressive
    
    echo ""
    echo "SUCCESS! import.json has been removed from git history"
    echo ""
    echo "IMPORTANT NEXT STEPS:"
    echo "1. Review the changes: git log --all --oneline"
    echo "2. If you have a remote repository, you MUST force push:"
    echo "   git push origin --force --all"
    echo "   git push origin --force --tags"
    echo ""
    echo "3. All team members must re-clone the repository or run:"
    echo "   git fetch origin"
    echo "   git reset --hard origin/main  # or your branch name"
    echo ""
    echo "WARNING: Force pushing rewrites history. Coordinate with your team!"
else
    echo "import.json not found in git history"
fi

echo ""
echo "Step 6: Removing import.json from working directory (if exists)"
if [ -f import.json ]; then
    rm import.json
    echo "Removed import.json from working directory"
else
    echo "import.json not found in working directory"
fi

echo ""
echo "Done!"

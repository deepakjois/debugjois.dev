#!/bin/bash

set -e

# Check if git repo is clean
if [[ -n $(git status -s) ]]; then
    echo "Error: Git repository is not clean. Please commit or stash changes."
    exit 1
fi

# Check if OBSIDIAN_REPO environment variable is set
if [[ -z "$OBSIDIAN_REPO" ]]; then
    echo "Error: OBSIDIAN_REPO environment variable is not set."
    exit 1
fi

# Update repo
git pull

# Rsync contents
rsync -au --out-format="%n" "$OBSIDIAN_REPO/daily/" content/daily-notes/

# Check if there are changes
if [[ -n $(git status -s) ]]; then
    # Commit changes with date and time
    current_datetime=$(date "+%Y-%m-%d %H:%M:%S")
    git add content/daily-notes/
    git commit -m "Obsidian Sync $current_datetime"
    echo "Changes committed successfully."
    git push
else
    echo "No changes to commit."
fi

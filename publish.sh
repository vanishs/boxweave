#!/usr/bin/env bash
# Publish this folder as a new public GitHub repo using the owner's own gh credentials.
set -Eeuo pipefail
cd "$(dirname "$0")"
repo="${1:-vanishs/boxweave}"
[[ "$repo" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || { echo 'Usage: bash publish.sh OWNER/REPO' >&2; exit 2; }
command -v git >/dev/null || { echo 'Install git first.' >&2; exit 1; }
command -v gh >/dev/null || { echo 'Install GitHub CLI gh and authenticate with gh auth login first.' >&2; exit 1; }
gh auth status >/dev/null || { echo 'Please run gh auth login first.' >&2; exit 1; }
if gh repo view "$repo" >/dev/null 2>&1; then
  echo "Repository $repo exists. Refusing automatic overwrite; push manually."
  exit 1
fi
if [[ -d .git ]]; then
  echo 'Existing .git detected. Refusing to rewrite history.' >&2
  exit 1
fi
git init -b main
git add README.md LICENSE install.sh publish.sh publish.ps1 .gitignore clients server agents providers docs tests .github
git commit -m 'Initial release: BoxWeave'
gh repo create "$repo" --public --source=. --remote=origin --push
printf '\nCreated: https://github.com/%s\n' "$repo"
printf 'Ubuntu installer: https://raw.githubusercontent.com/%s/main/install.sh\n' "$repo"

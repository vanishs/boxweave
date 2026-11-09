#!/bin/sh
set -eu
root="$(git rev-parse --show-toplevel)"
cd "$root"
test -f .githooks/pre-commit
test -f .githooks/pre-push
chmod +x .githooks/pre-commit .githooks/pre-push
git config --local core.hooksPath .githooks
printf 'Installed repository-scoped pre-commit/pre-push hooks.\n'
printf 'Server-side PR status checks still need an enabled GitHub ruleset.\n'

#!/usr/bin/env bash
# Explicit source-checkout or installed-package management entry point.
set -Eeuo pipefail
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
if [[ -f "$ROOT/bwmini/manage.py" ]]; then
  if [[ -x "$ROOT/.venv/bin/python" ]]; then
    exec "$ROOT/.venv/bin/python" -m bwmini.manage "$@"
  fi
  exec python3 -m bwmini.manage "$@"
fi
printf '%s\n' 'BoxWeave management sources are missing; use a reviewed application release.' >&2
exit 2

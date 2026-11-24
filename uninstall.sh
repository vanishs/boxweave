#!/bin/sh
# BoxWeave reviewed uninstall entrypoint. No args -> help, no host action.
set -eu
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
CORE="$SCRIPT_DIR/scripts/uninstall_core.py"
if [ ! -f "$CORE" ]; then
    echo "BoxWeave uninstaller: source wrapper requires scripts/uninstall_core.py; use the complete checkout or the self-contained matching Release uninstall.sh." >&2
    exit 1
fi
exec python3 "$CORE" "$@"

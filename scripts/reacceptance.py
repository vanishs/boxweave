#!/usr/bin/env python3
"""Use the packaged BoxWeave reacceptance engine from a source checkout.

This wrapper exists for developer ergonomics. On an installed host the module
is invoked in the non-root BoxWeave application environment instead.
"""
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from bwmini.reacceptance import main

if __name__ == "__main__":
    raise SystemExit(main())


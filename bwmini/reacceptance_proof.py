"""Legacy source-checkout compatibility only; excluded from customer ZIP.

Use python -m tools.reacceptance.cli for the developer runner.
"""
import importlib as _importlib
import sys as _sys

if __name__ == "__main__":
    raise SystemExit(_importlib.import_module("tools.reacceptance.proof").main())
_sys.modules[__name__] = _importlib.import_module("tools.reacceptance.proof")

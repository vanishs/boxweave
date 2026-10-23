#!/usr/bin/env python3
"""Source-checkout entry point; run on the host of the inner BoxWeave MCP."""
from pathlib import Path
import sys

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'server'))
from smoke import main

if __name__=='__main__': raise SystemExit(main())

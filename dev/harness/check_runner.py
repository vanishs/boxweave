"""Run a nonempty unittest discovery, rather than accepting zero-test OK."""
import argparse
import sys
import unittest


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument("pattern")
    args = parser.parse_args(argv)
    suite = unittest.defaultTestLoader.discover("scripts", pattern=args.pattern)
    if not suite.countTestCases():
        print("No tests discovered; refusing PASS", file=sys.stderr)
        return 2
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    return 0 if result.wasSuccessful() and result.testsRun else 1


if __name__ == "__main__":
    raise SystemExit(main())
